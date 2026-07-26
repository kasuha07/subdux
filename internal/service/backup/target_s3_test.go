package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
)

func TestParseS3ConfigValidation(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{
			"endpoint":          "s3.example.com",
			"bucket":            "backups",
			"access_key_id":     "AKIA",
			"secret_access_key": "secret",
		}
	}

	t.Run("valid bare host defaults to ssl", func(t *testing.T) {
		cfg, err := parseS3Config(base())
		if err != nil {
			t.Fatalf("parseS3Config() error = %v", err)
		}
		if cfg.endpointHost != "s3.example.com" {
			t.Fatalf("endpointHost = %q, want s3.example.com", cfg.endpointHost)
		}
		if !cfg.secure {
			t.Fatal("secure = false, want true by default")
		}
		if cfg.retention != defaultRetentionCount {
			t.Fatalf("retention = %d, want default %d", cfg.retention, defaultRetentionCount)
		}
	})

	t.Run("http scheme forces insecure", func(t *testing.T) {
		config := base()
		config["endpoint"] = "http://minio.local:9000"
		config["use_ssl"] = true // explicit scheme wins over the flag
		cfg, err := parseS3Config(config)
		if err != nil {
			t.Fatalf("parseS3Config() error = %v", err)
		}
		if cfg.endpointHost != "minio.local:9000" {
			t.Fatalf("endpointHost = %q, want minio.local:9000", cfg.endpointHost)
		}
		if cfg.secure {
			t.Fatal("secure = true, want false for http:// endpoint")
		}
	})

	t.Run("full URL with root path is accepted", func(t *testing.T) {
		config := base()
		config["endpoint"] = "https://s3.example.com/"
		cfg, err := parseS3Config(config)
		if err != nil {
			t.Fatalf("parseS3Config() error = %v", err)
		}
		if cfg.endpointHost != "s3.example.com" {
			t.Fatalf("endpointHost = %q, want s3.example.com", cfg.endpointHost)
		}
		if !cfg.secure {
			t.Fatal("secure = false, want true for https:// endpoint")
		}
	})

	invalidEndpoints := []struct {
		name     string
		endpoint string
	}{
		{name: "path", endpoint: "https://s3.example.com/backups"},
		{name: "query", endpoint: "https://s3.example.com?region=us-east-1"},
		{name: "fragment", endpoint: "https://s3.example.com#backups"},
		{name: "userinfo with password", endpoint: "https://user:password@s3.example.com"},
		{name: "userinfo without password", endpoint: "https://user@s3.example.com"},
		{name: "bare authority userinfo", endpoint: "user:password@s3.example.com"},
	}
	for _, tc := range invalidEndpoints {
		t.Run("rejects endpoint "+tc.name, func(t *testing.T) {
			config := base()
			config["endpoint"] = tc.endpoint
			if _, err := parseS3Config(config); !errors.Is(err, ErrS3EndpointInvalid) {
				t.Fatalf("parseS3Config() error = %v, want ErrS3EndpointInvalid", err)
			}
		})
	}

	t.Run("prefix normalized", func(t *testing.T) {
		config := base()
		config["prefix"] = "/subdux/nightly/"
		cfg, err := parseS3Config(config)
		if err != nil {
			t.Fatalf("parseS3Config() error = %v", err)
		}
		if cfg.prefix != "subdux/nightly" {
			t.Fatalf("prefix = %q, want subdux/nightly", cfg.prefix)
		}
	})

	t.Run("missing endpoint", func(t *testing.T) {
		config := base()
		delete(config, "endpoint")
		if _, err := parseS3Config(config); !errors.Is(err, ErrS3EndpointRequired) {
			t.Fatalf("error = %v, want ErrS3EndpointRequired", err)
		}
	})

	t.Run("missing bucket", func(t *testing.T) {
		config := base()
		delete(config, "bucket")
		if _, err := parseS3Config(config); !errors.Is(err, ErrS3BucketRequired) {
			t.Fatalf("error = %v, want ErrS3BucketRequired", err)
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		config := base()
		delete(config, "secret_access_key")
		if _, err := parseS3Config(config); !errors.Is(err, ErrS3CredentialsRequired) {
			t.Fatalf("error = %v, want ErrS3CredentialsRequired", err)
		}
	})

	t.Run("endpoint with path is invalid", func(t *testing.T) {
		config := base()
		config["endpoint"] = "s3.example.com/some/path"
		if _, err := parseS3Config(config); !errors.Is(err, ErrS3EndpointInvalid) {
			t.Fatalf("error = %v, want ErrS3EndpointInvalid", err)
		}
	})
}

func TestS3ObjectKey(t *testing.T) {
	withPrefix := &s3Target{prefix: "subdux/nightly"}
	if got := withPrefix.objectKey("subdux-backup-x.zip"); got != "subdux/nightly/subdux-backup-x.zip" {
		t.Fatalf("objectKey with prefix = %q", got)
	}
	noPrefix := &s3Target{}
	if got := noPrefix.objectKey("subdux-backup-x.zip"); got != "subdux-backup-x.zip" {
		t.Fatalf("objectKey without prefix = %q", got)
	}
}

// TestCreateS3DestinationEncryptsSecret confirms the CRUD path accepts an s3
// destination, validates it without network I/O, and stores the secret
// encrypted rather than in plaintext.
func TestCreateS3DestinationEncryptsSecret(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"topsecret"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	config := decodeStoredConfig(t, svc, created.ID)
	if config["secret_access_key"] != "topsecret" {
		t.Fatalf("decoded secret = %v, want topsecret", config["secret_access_key"])
	}

	views, err := svc.ListDestinations()
	if err != nil {
		t.Fatalf("ListDestinations() error = %v", err)
	}
	var view *DestinationView
	for i := range views {
		if views[i].ID == created.ID {
			view = &views[i]
		}
	}
	if view == nil {
		t.Fatal("created s3 destination not listed")
	}
	if strings.Contains(view.Config, "topsecret") {
		t.Fatal("listed config leaked the secret in plaintext")
	}
	if len(view.ConfiguredSecretFields) != 1 || view.ConfiguredSecretFields[0] != "secret_access_key" {
		t.Fatalf("configured secret fields = %v, want [secret_access_key]", view.ConfiguredSecretFields)
	}
}

func TestCreateS3DestinationRejectsInvalidConfig(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	if _, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","access_key_id":"AKIA","secret_access_key":"s"}`,
	}); !errors.Is(err, ErrS3BucketRequired) {
		t.Fatalf("CreateDestination() error = %v, want ErrS3BucketRequired", err)
	}
}

func TestCreateS3DestinationRejectsUnsafeEndpoint(t *testing.T) {
	endpoints := []struct {
		name     string
		endpoint string
	}{
		{name: "path", endpoint: "https://s3.example.com/backups"},
		{name: "query", endpoint: "https://s3.example.com?region=us-east-1"},
		{name: "userinfo", endpoint: "https://user:password@s3.example.com"},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newBackupTestDB(t)
			_, err := svc.CreateDestination(CreateDestinationInput{
				Type: "s3",
				Config: fmt.Sprintf(
					`{"endpoint":%q,"bucket":"backups","access_key_id":"AKIA","secret_access_key":"secret"}`,
					tc.endpoint,
				),
			})
			if !errors.Is(err, ErrS3EndpointInvalid) {
				t.Fatalf("CreateDestination() error = %v, want ErrS3EndpointInvalid", err)
			}
		})
	}
}

// stubS3Server is a minimal S3-compatible endpoint sufficient to exercise the
// minio client wiring for Put, List (V2), and Delete against path-style
// addressing. It does not verify signatures; it only records objects in memory.
type stubS3Server struct {
	mu         sync.Mutex
	objects    map[string][]byte
	listStatus int
}

func (s *stubS3Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Path-style addressing: /<bucket>/<key...>. A list request is GET on the
	// bucket with list-type=2.
	trimmed := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(trimmed, "/", 2)
	key := ""
	if len(parts) == 2 {
		key = parts[1]
	}

	switch r.Method {
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		s.objects[key] = body
		w.Header().Set("ETag", `"stub"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodDelete:
		delete(s.objects, key)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		if r.URL.Query().Get("list-type") == "2" {
			if s.listStatus != 0 {
				w.WriteHeader(s.listStatus)
				return
			}
			s.writeListResponse(w, r.URL.Query().Get("prefix"))
			return
		}
		if body, ok := s.objects[key]; ok {
			_, _ = w.Write(body)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *stubS3Server) writeListResponse(w http.ResponseWriter, prefix string) {
	var contents strings.Builder
	for key, body := range s.objects {
		if prefix != "" && !strings.HasPrefix(key, prefix) {
			continue
		}
		fmt.Fprintf(&contents,
			`<Contents><Key>%s</Key><Size>%d</Size><LastModified>2026-06-15T03:00:00.000Z</LastModified></Contents>`,
			key, len(body))
	}
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprintf(w,
		`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>backups</Name><Prefix>%s</Prefix><KeyCount>1</KeyCount><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated>%s</ListBucketResult>`,
		prefix, contents.String())
}

// TestS3TargetRoundTrip drives Put, List, and Delete against the stub server to
// confirm the minio client wiring, path-style addressing, prefix handling, and
// filename-pattern gating all work end to end.
func TestS3TargetRoundTrip(t *testing.T) {
	stub := &stubS3Server{objects: map[string][]byte{}}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	endpoint := strings.TrimPrefix(server.URL, "http://")
	config := map[string]any{
		"endpoint":          endpoint,
		"use_ssl":           false,
		"use_path_style":    true,
		"region":            "us-east-1",
		"bucket":            "backups",
		"prefix":            "nightly",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
		"retention_count":   5,
	}

	target, err := newS3Target(config, nil)
	if err != nil {
		t.Fatalf("newS3Target() error = %v", err)
	}

	const name = "subdux-backup-20260615-030000-abc123.zip"
	payload := []byte("encrypted-archive-bytes")
	if err := target.Put(context.Background(), name, strings.NewReader(string(payload)), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	// The object must have been stored under the prefixed key.
	if _, ok := stub.objects["nightly/"+name]; !ok {
		t.Fatalf("stored keys = %v, want nightly/%s", keysOf(stub.objects), name)
	}

	objects, err := target.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("List() = %d objects, want 1", len(objects))
	}
	if objects[0].Name != name {
		t.Fatalf("listed name = %q, want %q (base name, prefix stripped)", objects[0].Name, name)
	}

	if err := target.Delete(context.Background(), name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, ok := stub.objects["nightly/"+name]; ok {
		t.Fatal("object still present after Delete")
	}
}

func TestS3TargetListScopesPrefixToObjectDirectory(t *testing.T) {
	const name = "subdux-backup-20260615-030000-abc123.zip"
	const adjacentName = "subdux-backup-20260615-040000-def456.zip"
	const nestedName = "subdux-backup-20260615-050000-ghi789.zip"
	stub := &stubS3Server{objects: map[string][]byte{
		"nightly/" + name:                 []byte("archive"),
		"nightly-archive/" + adjacentName: []byte("unrelated archive"),
		"nightly/nested/" + nestedName:    []byte("nested archive"),
	}}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	target, err := newS3Target(map[string]any{
		"endpoint":          strings.TrimPrefix(server.URL, "http://"),
		"use_ssl":           false,
		"use_path_style":    true,
		"region":            "us-east-1",
		"bucket":            "backups",
		"access_key_id":     "AKIA",
		"secret_access_key": "secret",
		"prefix":            "nightly",
	}, nil)
	if err != nil {
		t.Fatalf("newS3Target() error = %v", err)
	}

	if err := target.Put(context.Background(), name, strings.NewReader("archive"), int64(len("archive"))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	objects, err := target.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 || objects[0].Name != name {
		t.Fatalf("List() = %#v, want only direct child %q", objects, name)
	}

	if err := target.Delete(context.Background(), "nested/"+nestedName); err != nil {
		t.Fatalf("Delete(nested object) error = %v", err)
	}
	if _, ok := stub.objects["nightly/nested/"+nestedName]; !ok {
		t.Fatal("nested object was deleted by a base-name reconstruction")
	}
}

func TestRunDestinationBackupCountsUploadSuccessWhenS3RetentionFails(t *testing.T) {
	stub := &stubS3Server{
		objects:    map[string][]byte{},
		listStatus: http.StatusInternalServerError,
	}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	svc, _ := newBackupTestDB(t)
	endpoint := strings.TrimPrefix(server.URL, "http://")
	destination, err := svc.CreateDestination(CreateDestinationInput{
		Type:    "s3",
		Enabled: true,
		Config:  fmt.Sprintf(`{"endpoint":%q,"use_ssl":false,"use_path_style":true,"region":"us-east-1","bucket":"backups","access_key_id":"AKIA","secret_access_key":"secret"}`, endpoint),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	result, err := svc.RunDestinationBackup(context.Background(), destination.ID)
	if err != nil {
		t.Fatalf("RunDestinationBackup() error = %v, result = %+v, want nil after successful upload", err, result)
	}
	if len(result.Results) != 1 {
		t.Fatalf("RunDestinationBackup() results = %d, want 1", len(result.Results))
	}
	if result.Status != StatusPartial || result.RetentionStatus != StatusPartial {
		t.Fatalf("RunDestinationBackup() aggregate status = %q retention = %q, want partial/partial", result.Status, result.RetentionStatus)
	}
	outcome := result.Results[0]
	if outcome.DestinationID != destination.ID || !outcome.Success {
		t.Fatalf("delivery result = %+v, want successful delivery", outcome)
	}
	if outcome.RetentionStatus != StatusFailed || outcome.RetentionError == "" {
		t.Fatalf("retention result = %+v, want independent failure", outcome)
	}
	if outcome.Error != "" {
		t.Fatalf("delivery error = %q, want empty after successful upload", outcome.Error)
	}

	var persisted model.BackupDestination
	if err := svc.DB.First(&persisted, destination.ID).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if persisted.LastStatus != StatusOK || persisted.LastRunAt == nil {
		t.Fatalf("persisted delivery status = %q/%v, want success/set", persisted.LastStatus, persisted.LastRunAt)
	}
	if persisted.LastRetentionStatus != StatusFailed || persisted.LastRetentionError == "" {
		t.Fatalf("persisted retention status = %q error = %q, want failed/non-empty", persisted.LastRetentionStatus, persisted.LastRetentionError)
	}
	if persisted.LastBookkeepingStatus != StatusOK || persisted.LastBookkeepingError != "" {
		t.Fatalf("persisted bookkeeping status = %q error = %q, want success/empty", persisted.LastBookkeepingStatus, persisted.LastBookkeepingError)
	}

	stub.mu.Lock()
	_, uploaded := stub.objects[result.ArchiveName]
	stub.mu.Unlock()
	if !uploaded {
		t.Fatalf("uploaded object %q not found", result.ArchiveName)
	}
}

func keysOf(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
