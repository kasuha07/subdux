package backup

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The skip_tls_verify switch is the backup equivalent of the SMTP sender's:
// off by default, and on only for the one destination whose admin turned it on
// for a self-signed endpoint. These tests pin both halves of that contract —
// that it actually relaxes verification for its own destination, and that it
// does not relax it for anything else in the process.

const tlsTestArchiveName = "subdux-backup-20260615-030000-abc123.zip"

func TestS3TargetSkipTLSVerifyReachesSelfSignedEndpoint(t *testing.T) {
	stub := &stubS3Server{objects: map[string][]byte{}}
	server := httptest.NewTLSServer(stub)
	t.Cleanup(server.Close)

	endpoint := strings.TrimPrefix(server.URL, "https://")
	config := func(skip bool) map[string]any {
		return map[string]any{
			"endpoint":          endpoint,
			"use_ssl":           true,
			"use_path_style":    true,
			"region":            "us-east-1",
			"bucket":            "backups",
			"access_key_id":     "AKIA",
			"secret_access_key": "secret",
			"skip_tls_verify":   skip,
		}
	}

	// httptest's certificate is self-signed, so the default policy must refuse
	// it. Without this half the passing case below would prove nothing.
	verifying, err := newS3Target(config(false), nil)
	if err != nil {
		t.Fatalf("newS3Target(skip_tls_verify=false) error = %v", err)
	}
	// A short deadline caps the minio client's handshake retries; the assertion
	// below distinguishes the two outcomes that deadline can produce.
	rejectCtx, cancelReject := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancelReject)
	if _, err := verifying.List(rejectCtx); err == nil {
		t.Fatal("List() succeeded against a self-signed endpoint with verification on, want a certificate error")
	}

	skipping, err := newS3Target(config(true), nil)
	if err != nil {
		t.Fatalf("newS3Target(skip_tls_verify=true) error = %v", err)
	}
	payload := "encrypted-archive-bytes"
	if err := skipping.Put(context.Background(), tlsTestArchiveName, strings.NewReader(payload), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v, want success with verification skipped", err)
	}
	objects, err := skipping.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want success with verification skipped", err)
	}
	if len(objects) != 1 || objects[0].Name != tlsTestArchiveName {
		t.Fatalf("List() = %#v, want the single uploaded archive %q", objects, tlsTestArchiveName)
	}
}

func TestWebDAVTargetSkipTLSVerifyReachesSelfSignedEndpoint(t *testing.T) {
	stub := &stubWebDAVServer{base: "/dav", objects: map[string][]byte{}}
	server := httptest.NewTLSServer(stub)
	t.Cleanup(server.Close)

	config := func(skip bool) map[string]any {
		return map[string]any{
			"url":             server.URL + "/dav",
			"username":        "user",
			"password":        "pass",
			"skip_tls_verify": skip,
		}
	}

	verifying, err := newWebDAVTarget(config(false), nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget(skip_tls_verify=false) error = %v", err)
	}
	if _, err := verifying.List(context.Background()); err == nil {
		t.Fatal("List() succeeded against a self-signed endpoint with verification on, want a certificate error")
	}

	skipping, err := newWebDAVTarget(config(true), nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget(skip_tls_verify=true) error = %v", err)
	}
	payload := "encrypted-archive-bytes"
	if err := skipping.Put(context.Background(), tlsTestArchiveName, strings.NewReader(payload), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v, want success with verification skipped", err)
	}
	if _, ok := stub.objects["/dav/"+tlsTestArchiveName]; !ok {
		t.Fatalf("stored paths = %v, want /dav/%s", keysOf(stub.objects), tlsTestArchiveName)
	}
}

// TestSkipTLSVerifyDoesNotLeakIntoSharedTransport is the reason
// backupDestinationTransport clones instead of adjusting in place: for this
// purpose the outbound constructor hands back http.DefaultTransport, which every
// other outbound caller in the process shares. Turning verification off there
// would silently disable it for OIDC, notifications, and the icon proxy.
func TestSkipTLSVerifyDoesNotLeakIntoSharedTransport(t *testing.T) {
	shared, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Skip("http.DefaultTransport is not an *http.Transport in this build")
	}

	if _, err := newS3Target(map[string]any{
		"endpoint":          "s3.example.com",
		"use_ssl":           true,
		"bucket":            "backups",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
		"skip_tls_verify":   true,
	}, nil); err != nil {
		t.Fatalf("newS3Target() error = %v", err)
	}
	if _, err := newWebDAVTarget(map[string]any{
		"url":             "https://dav.example.com/dav",
		"skip_tls_verify": true,
	}, nil); err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}

	if shared.TLSClientConfig != nil && shared.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("http.DefaultTransport now skips certificate verification; the per-destination switch leaked process-wide")
	}
}

// TestSkipTLSVerifyPreservesResolvedTransportSettings pins that the clone keeps
// the transport configuration resolved for this purpose (proxy, dialer, timeouts)
// rather than replacing it with a bare transport that happens to skip
// verification.
func TestSkipTLSVerifyPreservesResolvedTransportSettings(t *testing.T) {
	base := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        42,
		DisableCompression:  true,
		TLSHandshakeTimeout: 7,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS13, ServerName: "dav.example.com"},
	}

	cloned, err := backupDestinationTransport(base)
	if err != nil {
		t.Fatalf("backupDestinationTransport() error = %v", err)
	}
	if !cloned.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("cloned transport still verifies certificates")
	}
	if cloned.MaxIdleConns != 42 || !cloned.DisableCompression || cloned.TLSHandshakeTimeout != 7 {
		t.Fatalf("cloned transport dropped resolved settings: %+v", cloned)
	}
	if cloned.TLSClientConfig.MinVersion != tls.VersionTLS13 || cloned.TLSClientConfig.ServerName != "dav.example.com" {
		t.Fatalf("cloned TLS config dropped resolved settings: %+v", cloned.TLSClientConfig)
	}
	if base.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("the source transport's TLS config was mutated in place")
	}
}

// TestSkipTLSVerifyRejectsUncopyableTransport pins the fail-loud branch: a
// transport the switch cannot be applied to must surface as an error rather
// than quietly keeping verification on, which would leave the admin staring at
// handshake failures the toggle appeared to have addressed.
func TestSkipTLSVerifyRejectsUncopyableTransport(t *testing.T) {
	_, err := backupDestinationTransport(roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, nil
	}))
	if !errors.Is(err, ErrBackupSkipTLSVerifyUnsupported) {
		t.Fatalf("error = %v, want ErrBackupSkipTLSVerifyUnsupported", err)
	}
}

// TestSkipTLSVerifySetsMinimumTLSVersion pins that relaxing certificate
// verification does not also drop the connection to a legacy protocol version:
// the switch is about an unverifiable certificate, not about accepting TLS 1.0.
func TestSkipTLSVerifySetsMinimumTLSVersion(t *testing.T) {
	cloned, err := backupDestinationTransport(nil)
	if err != nil {
		t.Fatalf("backupDestinationTransport(nil) error = %v", err)
	}
	if cloned.TLSClientConfig == nil || cloned.TLSClientConfig.MinVersion < tls.VersionTLS12 {
		t.Fatalf("TLS config = %+v, want MinVersion at least TLS 1.2", cloned.TLSClientConfig)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestSkipTLSVerifySchemaScope pins the field to the transports that actually
// speak TLS. A local destination writes to disk, so accepting the field there
// would advertise a switch that can never do anything.
func TestSkipTLSVerifySchemaScope(t *testing.T) {
	if _, err := newLocalTarget(map[string]any{
		"dir":             t.TempDir(),
		"skip_tls_verify": true,
	}); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
		t.Fatalf("local newLocalTarget() error = %v, want ErrInvalidBackupDestinationConfig", err)
	}

	s3Cfg, err := parseS3Config(map[string]any{
		"endpoint":          "s3.example.com",
		"bucket":            "backups",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
		"skip_tls_verify":   true,
	})
	if err != nil {
		t.Fatalf("parseS3Config() error = %v", err)
	}
	if !s3Cfg.skipTLSVerify {
		t.Fatal("parseS3Config() dropped skip_tls_verify")
	}

	webdavCfg, err := parseWebDAVConfig(map[string]any{
		"url":             "https://dav.example.com/dav",
		"skip_tls_verify": true,
	})
	if err != nil {
		t.Fatalf("parseWebDAVConfig() error = %v", err)
	}
	if !webdavCfg.skipTLSVerify {
		t.Fatal("parseWebDAVConfig() dropped skip_tls_verify")
	}

	// A non-boolean value must be refused rather than coerced, like every other
	// field in the explicit schema.
	if _, err := parseWebDAVConfig(map[string]any{
		"url":             "https://dav.example.com/dav",
		"skip_tls_verify": "yes",
	}); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
		t.Fatalf("parseWebDAVConfig(string) error = %v, want ErrInvalidBackupDestinationConfig", err)
	}
}

// TestSkipTLSVerifyDefaultsOff pins the default for a config saved before the
// field existed: absent means verification stays on.
func TestSkipTLSVerifyDefaultsOff(t *testing.T) {
	s3Cfg, err := parseS3Config(map[string]any{
		"endpoint":          "s3.example.com",
		"bucket":            "backups",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
	})
	if err != nil {
		t.Fatalf("parseS3Config() error = %v", err)
	}
	if s3Cfg.skipTLSVerify {
		t.Fatal("s3 skip_tls_verify defaulted to true")
	}

	webdavCfg, err := parseWebDAVConfig(map[string]any{"url": "https://dav.example.com/dav"})
	if err != nil {
		t.Fatalf("parseWebDAVConfig() error = %v", err)
	}
	if webdavCfg.skipTLSVerify {
		t.Fatal("webdav skip_tls_verify defaulted to true")
	}
}

// TestSkipTLSVerifySurvivesSanitize pins that the flag round-trips to the admin
// UI. It is not a secret, so unlike the credentials it must come back with its
// stored value rather than blanked — otherwise reopening the edit dialog would
// silently present the switch as off and save it that way.
func TestSkipTLSVerifySurvivesSanitize(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	for _, tc := range []struct {
		destType string
		config   string
	}{
		{
			destType: "s3",
			config:   `{"endpoint":"s3.example.com","bucket":"backups","access_key_id":"AKIA","secret_access_key":"secret","skip_tls_verify":true}`,
		},
		{
			destType: "webdav",
			config:   `{"url":"https://dav.example.com/dav","skip_tls_verify":true}`,
		},
	} {
		t.Run(tc.destType, func(t *testing.T) {
			created, err := svc.CreateDestination(CreateDestinationInput{
				Type:   tc.destType,
				Config: tc.config,
			})
			if err != nil {
				t.Fatalf("CreateDestination() error = %v", err)
			}
			if !strings.Contains(created.Config, `"skip_tls_verify":true`) {
				t.Fatalf("sanitized config = %s, want skip_tls_verify preserved", created.Config)
			}
		})
	}
}
