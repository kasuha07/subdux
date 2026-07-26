package backup

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
)

// authRecordingWebDAVServer wraps the PROPFIND half of a WebDAV collection and
// records the credentials each probe actually presented, so a test can prove
// which secret reached the wire rather than only that the probe succeeded.
type authRecordingWebDAVServer struct {
	mu       sync.Mutex
	base     string
	username string
	password string
	hadAuth  bool
	requests int
}

func (s *authRecordingWebDAVServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	username, password, ok := r.BasicAuth()
	s.username, s.password, s.hadAuth = username, password, ok
	s.requests++
	s.mu.Unlock()

	if r.Method != "PROPFIND" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusMultiStatus)
	fmt.Fprintf(w,
		`<?xml version="1.0"?><d:multistatus xmlns:d="DAV:"><d:response><d:href>%s/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response></d:multistatus>`,
		s.base)
}

func (s *authRecordingWebDAVServer) observed() (string, string, bool, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.username, s.password, s.hadAuth, s.requests
}

// newWebDAVProbeDestination persists a webdav destination pointing at base with
// the given stored password and returns its id.
func newWebDAVProbeDestination(t *testing.T, svc *Service, base, password string) uint {
	t.Helper()

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "webdav",
		Config: fmt.Sprintf(`{"url":%q,"username":"stored-user","password":%q}`, base, password),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	return created.ID
}

// TestTestDestinationConfigProbesUnsavedLocalConfig is the create-flow case: a
// destination that does not exist yet is reachable purely from typed values.
func TestTestDestinationConfigProbesUnsavedLocalConfig(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	dir := t.TempDir()
	archive := filepath.Join(dir, "subdux-backup-20260615-030000-aaa111.zip")
	if err := os.WriteFile(archive, []byte("x"), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	count, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:   "local",
		Config: fmt.Sprintf(`{"dir":%q,"retention_count":5}`, dir),
	})
	if err != nil {
		t.Fatalf("TestDestinationConfig() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("TestDestinationConfig() count = %d, want 1", count)
	}

	var stored int64
	if err := svc.DB.Model(&model.BackupDestination{}).Count(&stored).Error; err != nil {
		t.Fatalf("count destinations: %v", err)
	}
	if stored != 0 {
		t.Fatalf("probe persisted %d destinations, want 0", stored)
	}
}

// TestTestDestinationConfigIgnoresScheduleFields confirms a probe is not blocked
// by the plan half of the config. Encryption on with no password is rejected on
// save, but it says nothing about whether the target is reachable.
func TestTestDestinationConfigIgnoresScheduleFields(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	// An unusable plan: a malformed firing time and encryption on with no
	// password. Both are rejected on save and neither affects reachability.
	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type: "local",
		Config: fmt.Sprintf(
			`{"dir":%q,"time_of_day":"not-a-time","encrypt_enabled":true,"encryption_password":""}`,
			t.TempDir(),
		),
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v, want nil for schedule-only problems", err)
	}

	// The case above only proves parseScheduleConfig is not called;
	// decodeDestinationConfigStrict would skip those keys anyway. A schedule field
	// whose TYPE is invalid survives that skip, so it fails unless the fields are
	// actually stripped before the target is built.
	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:   "local",
		Config: fmt.Sprintf(`{"dir":%q,"time_of_day":42}`, t.TempDir()),
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v, want the schedule fields to be stripped", err)
	}
}

func TestTestDestinationConfigUsesTypedSecret(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &authRecordingWebDAVServer{base: "/dav"}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:   "webdav",
		Config: fmt.Sprintf(`{"url":%q,"username":"typed-user","password":"typed-pass"}`, server.URL+"/dav"),
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v", err)
	}

	username, password, hadAuth, _ := stub.observed()
	if !hadAuth || username != "typed-user" || password != "typed-pass" {
		t.Fatalf("probe presented (%q, %q, auth=%t), want the typed credential", username, password, hadAuth)
	}
}

// TestTestDestinationConfigInheritsSecretForSameAddress covers the ordinary edit:
// the admin changed a field that does not move the credential, so the blank
// (server-masked) secret is filled in from storage.
func TestTestDestinationConfigInheritsSecretForSameAddress(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &authRecordingWebDAVServer{base: "/dav/archive"}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	id := newWebDAVProbeDestination(t, svc, server.URL+"/dav", "stored-pass")

	// Only the subfolder changed, which is the same host and therefore the same
	// place the stored password is already sent.
	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "webdav",
		Config:        fmt.Sprintf(`{"url":%q,"path":"archive","username":"stored-user","password":""}`, server.URL+"/dav"),
		DestinationID: id,
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v", err)
	}

	username, password, hadAuth, _ := stub.observed()
	if !hadAuth || username != "stored-user" || password != "stored-pass" {
		t.Fatalf("probe presented (%q, %q, auth=%t), want the stored credential", username, password, hadAuth)
	}
}

// TestTestDestinationConfigRefusesSecretForChangedHost is the security case: a
// probe must never post a stored credential to an address the same request
// introduced, because that would bypass the step-up gate on destination writes.
func TestTestDestinationConfigRefusesSecretForChangedHost(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	original := &authRecordingWebDAVServer{base: "/dav"}
	originalServer := httptest.NewServer(original)
	t.Cleanup(originalServer.Close)

	attacker := &authRecordingWebDAVServer{base: "/dav"}
	attackerServer := httptest.NewServer(attacker)
	t.Cleanup(attackerServer.Close)

	id := newWebDAVProbeDestination(t, svc, originalServer.URL+"/dav", "stored-pass")

	_, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "webdav",
		Config:        fmt.Sprintf(`{"url":%q,"username":"stored-user","password":""}`, attackerServer.URL+"/dav"),
		DestinationID: id,
	})
	if !errors.Is(err, ErrBackupDestinationSecretRequiredForTest) {
		t.Fatalf("TestDestinationConfig() error = %v, want ErrBackupDestinationSecretRequiredForTest", err)
	}

	if _, _, _, requests := attacker.observed(); requests != 0 {
		t.Fatalf("probe made %d requests to the changed host, want 0", requests)
	}
}

// TestTestDestinationConfigRefusesSecretWhenTLSVerifyRelaxed guards the other way
// a credential's destination can move: keeping the URL but turning certificate
// verification off.
func TestTestDestinationConfigRefusesSecretWhenTLSVerifyRelaxed(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &authRecordingWebDAVServer{base: "/dav"}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	id := newWebDAVProbeDestination(t, svc, server.URL+"/dav", "stored-pass")

	_, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "webdav",
		Config:        fmt.Sprintf(`{"url":%q,"username":"stored-user","password":"","skip_tls_verify":true}`, server.URL+"/dav"),
		DestinationID: id,
	})
	if !errors.Is(err, ErrBackupDestinationSecretRequiredForTest) {
		t.Fatalf("TestDestinationConfig() error = %v, want ErrBackupDestinationSecretRequiredForTest", err)
	}
}

// TestTestDestinationConfigHonoursClearedSecret confirms an explicitly cleared
// secret is probed as absent instead of being revived from storage.
func TestTestDestinationConfigHonoursClearedSecret(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &authRecordingWebDAVServer{base: "/dav"}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	id := newWebDAVProbeDestination(t, svc, server.URL+"/dav", "stored-pass")

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:                "webdav",
		Config:              fmt.Sprintf(`{"url":%q,"password":""}`, server.URL+"/dav"),
		DestinationID:       id,
		ClearedSecretFields: []string{"password"},
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v", err)
	}

	if _, password, hadAuth, _ := stub.observed(); hadAuth || password != "" {
		t.Fatalf("probe presented a credential (%q, auth=%t) after the secret was cleared", password, hadAuth)
	}
}

// TestTestDestinationConfigInheritsAcrossEquivalentS3Endpoints confirms the
// address pinning compares resolved endpoints rather than raw strings, so an
// admin who spells the same endpoint differently is not asked to retype a
// secret that is still going to the same host.
func TestTestDestinationConfigInheritsAcrossEquivalentS3Endpoints(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","use_ssl":true,"bucket":"b","access_key_id":"AKIA","secret_access_key":"stored"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	// "https://s3.example.com" resolves to the same host with the same TLS
	// decision, and only the prefix actually changed.
	config := map[string]any{
		"endpoint":          "https://s3.example.com",
		"use_ssl":           true,
		"bucket":            "b",
		"prefix":            "moved/",
		"access_key_id":     "AKIA",
		"secret_access_key": "",
	}
	if err := inheritProbeSecrets("s3", loadDestination(t, svc, created.ID), config, nil); err != nil {
		t.Fatalf("inheritProbeSecrets() error = %v", err)
	}
	if config["secret_access_key"] != "stored" {
		t.Fatalf("secret_access_key = %v, want the stored secret", config["secret_access_key"])
	}

	// Moving the endpoint to another host must not carry the secret along.
	config["endpoint"] = "https://s3.attacker.example"
	config["secret_access_key"] = ""
	if err := inheritProbeSecrets("s3", loadDestination(t, svc, created.ID), config, nil); !errors.Is(err, ErrBackupDestinationSecretRequiredForTest) {
		t.Fatalf("inheritProbeSecrets() error = %v, want ErrBackupDestinationSecretRequiredForTest", err)
	}
}

// TestTestDestinationConfigReportsConfigErrorsBeforeSecretRequirement keeps the
// diagnostics useful: a config the admin broke in some other way should name
// that problem rather than blaming a secret they never touched.
func TestTestDestinationConfigReportsConfigErrorsBeforeSecretRequirement(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"stored"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	_, err = svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "s3",
		Config:        `{"endpoint":"s3.example.com","bucket":"","access_key_id":"AKIA","secret_access_key":""}`,
		DestinationID: created.ID,
	})
	if !errors.Is(err, ErrS3BucketRequired) {
		t.Fatalf("TestDestinationConfig() error = %v, want ErrS3BucketRequired", err)
	}
}

func TestTestDestinationConfigTypeComesFromStoredDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	id := createLocalDestination(t, svc, t.TempDir(), 5)

	// Seed the probed directory so the count proves WHICH target ran, not merely
	// that no error came back.
	probeDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(probeDir, "subdux-backup-20260615-030000-aaa111.zip"),
		[]byte("x"), 0o600,
	); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	// A request claiming another type must not reinterpret the stored row: the
	// local config below would be nonsense under the s3 schema.
	count, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "s3",
		Config:        fmt.Sprintf(`{"dir":%q}`, probeDir),
		DestinationID: id,
	})
	if err != nil {
		t.Fatalf("TestDestinationConfig() error = %v, want the stored local type to be used", err)
	}
	if count != 1 {
		t.Fatalf("TestDestinationConfig() count = %d, want 1 from the probed local directory", count)
	}
}

func TestTestDestinationConfigRejectsUnknownTypeAndMissingDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:   "dropbox",
		Config: `{}`,
	}); !errors.Is(err, ErrInvalidBackupDestinationType) {
		t.Fatalf("TestDestinationConfig() error = %v, want ErrInvalidBackupDestinationType", err)
	}

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "local",
		Config:        `{}`,
		DestinationID: 4242,
	}); !errors.Is(err, ErrBackupDestinationNotFound) {
		t.Fatalf("TestDestinationConfig() error = %v, want ErrBackupDestinationNotFound", err)
	}
}

// TestProbeCredentialAddressExcludesNonAddressFields pins the intent of the
// pinning set itself: bucket, prefix, region, and WebDAV subfolder all speak to
// the same host, so they must not force a secret retype.
func TestProbeCredentialAddressExcludesNonAddressFields(t *testing.T) {
	base, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "b", "access_key_id": "AKIA",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	moved, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "other", "prefix": "p/", "region": "eu", "access_key_id": "AKIA",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if base != moved {
		t.Fatalf("address changed with bucket/prefix/region: %+v vs %+v", base, moved)
	}

	// use_path_style is excluded for the reason documented on
	// probeCredentialAddress: switching addressing modes can change which host
	// minio-go contacts, but SigV4 means only a signature and the (non-secret)
	// access key id ever travel there.
	pathStyle, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "b", "access_key_id": "AKIA", "use_path_style": true,
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if base != pathStyle {
		t.Fatalf("address changed with use_path_style: %+v vs %+v", base, pathStyle)
	}

	insecure, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "http://s3.example.com", "bucket": "b", "access_key_id": "AKIA",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if insecure == base {
		t.Fatalf("dropping TLS did not change the address identity: %+v", insecure)
	}

	// Relaxing certificate verification must change the identity, since it widens
	// who can present themselves as the pinned host.
	relaxed, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "b", "access_key_id": "AKIA", "skip_tls_verify": true,
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if relaxed == base {
		t.Fatalf("skip_tls_verify did not change the address identity: %+v", relaxed)
	}

	davBase, err := probeCredentialAddress("webdav", map[string]any{"url": "https://dav.example.com/remote"})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	davSubfolder, err := probeCredentialAddress("webdav", map[string]any{
		"url": "https://dav.example.com/remote", "path": "nested/folder",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if davBase != davSubfolder {
		t.Fatalf("webdav address changed with a subfolder: %+v vs %+v", davBase, davSubfolder)
	}
	if davBase.host != mustHost(t, "https://dav.example.com") {
		t.Fatalf("webdav address %+v does not identify the host", davBase)
	}
}

// TestProbeCredentialAddressPinsTheAuthenticatedAccount covers the
// password-testing oracle: a blank password must not be inherited and offered
// under a different WebDAV username on the same server.
func TestProbeCredentialAddressPinsTheAuthenticatedAccount(t *testing.T) {
	base, err := probeCredentialAddress("webdav", map[string]any{
		"url": "https://dav.example.com/dav", "username": "backup-bot",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	otherAccount, err := probeCredentialAddress("webdav", map[string]any{
		"url": "https://dav.example.com/dav", "username": "admin",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if credentialAddressPinned(base, otherAccount) {
		t.Fatal("a different webdav username stayed pinned, allowing a password-testing oracle")
	}

	s3Base, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "b", "access_key_id": "AKIA1",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	s3Rotated, err := probeCredentialAddress("s3", map[string]any{
		"endpoint": "s3.example.com", "bucket": "b", "access_key_id": "AKIA2",
	})
	if err != nil {
		t.Fatalf("probeCredentialAddress() error = %v", err)
	}
	if credentialAddressPinned(s3Base, s3Rotated) {
		t.Fatal("a different s3 access key id stayed pinned")
	}
}

// TestCredentialAddressPinningIsAsymmetric locks in that only the weakening
// direction unpins. Tightening transport protection cannot move a credential
// anywhere new, so refusing it would be friction reported under a message about
// a changed endpoint that never changed.
func TestCredentialAddressPinningIsAsymmetric(t *testing.T) {
	relaxed := credentialAddress{kind: "webdav", host: "dav.example.com", secure: true, skipTLSVerify: true}
	verifying := credentialAddress{kind: "webdav", host: "dav.example.com", secure: true}
	plaintext := credentialAddress{kind: "webdav", host: "dav.example.com"}

	if !credentialAddressPinned(relaxed, verifying) {
		t.Fatal("turning skip_tls_verify OFF unpinned the credential; tightening must stay pinned")
	}
	if credentialAddressPinned(verifying, relaxed) {
		t.Fatal("turning skip_tls_verify ON stayed pinned; relaxing must unpin")
	}
	if !credentialAddressPinned(plaintext, verifying) {
		t.Fatal("upgrading http to https unpinned the credential; tightening must stay pinned")
	}
	if credentialAddressPinned(verifying, plaintext) {
		t.Fatal("downgrading https to http stayed pinned; that puts a credential on the clear")
	}

	other := credentialAddress{kind: "webdav", host: "attacker.example", secure: true}
	if credentialAddressPinned(verifying, other) {
		t.Fatal("a different host stayed pinned")
	}
}

// TestTestDestinationConfigInheritsWhenTLSVerifyTightened is the end-to-end form
// of the asymmetry: switching verification back on must not demand the secret.
func TestTestDestinationConfigInheritsWhenTLSVerifyTightened(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &authRecordingWebDAVServer{base: "/dav"}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type: "webdav",
		Config: fmt.Sprintf(
			`{"url":%q,"username":"stored-user","password":"stored-pass","skip_tls_verify":true}`,
			server.URL+"/dav",
		),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:          "webdav",
		Config:        fmt.Sprintf(`{"url":%q,"username":"stored-user","password":"","skip_tls_verify":false}`, server.URL+"/dav"),
		DestinationID: created.ID,
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v, want nil when verification is tightened", err)
	}

	if _, password, hadAuth, _ := stub.observed(); !hadAuth || password != "stored-pass" {
		t.Fatalf("probe presented (%q, auth=%t), want the stored credential", password, hadAuth)
	}
}

// TestTestDestinationConfigIgnoresRetentionCount confirms retention is stripped
// like the schedule: List never reads it, so an out-of-range value must not fail
// a reachability check.
func TestTestDestinationConfigIgnoresRetentionCount(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	if _, err := svc.TestDestinationConfig(context.Background(), TestDestinationConfigInput{
		Type:   "local",
		Config: fmt.Sprintf(`{"dir":%q,"retention_count":9999}`, t.TempDir()),
	}); err != nil {
		t.Fatalf("TestDestinationConfig() error = %v, want nil for an out-of-range retention", err)
	}
}

func mustHost(t *testing.T, raw string) string {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed.Host
}
