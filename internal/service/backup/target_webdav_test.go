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
	"time"
)

func TestParseWebDAVConfigValidation(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		cfg, err := parseWebDAVConfig(map[string]any{
			"url":      "https://dav.example.com/remote.php/dav/files/user",
			"username": "user",
			"password": "pass",
		})
		if err != nil {
			t.Fatalf("parseWebDAVConfig() error = %v", err)
		}
		if cfg.baseURL != "https://dav.example.com/remote.php/dav/files/user" {
			t.Fatalf("baseURL = %q", cfg.baseURL)
		}
		if cfg.retention != defaultRetentionCount {
			t.Fatalf("retention = %d, want default", cfg.retention)
		}
	})

	t.Run("url and path joined, slashes normalized", func(t *testing.T) {
		cfg, err := parseWebDAVConfig(map[string]any{
			"url":  "https://dav.example.com/dav/",
			"path": "/subdux/nightly/",
		})
		if err != nil {
			t.Fatalf("parseWebDAVConfig() error = %v", err)
		}
		if cfg.baseURL != "https://dav.example.com/dav/subdux/nightly" {
			t.Fatalf("baseURL = %q, want joined without double slashes", cfg.baseURL)
		}
	})

	t.Run("missing url", func(t *testing.T) {
		if _, err := parseWebDAVConfig(map[string]any{}); !errors.Is(err, ErrWebDAVURLRequired) {
			t.Fatalf("error = %v, want ErrWebDAVURLRequired", err)
		}
	})

	t.Run("invalid url scheme", func(t *testing.T) {
		if _, err := parseWebDAVConfig(map[string]any{"url": "ftp://dav.example.com"}); !errors.Is(err, ErrWebDAVURLInvalid) {
			t.Fatalf("error = %v, want ErrWebDAVURLInvalid", err)
		}
	})

	for _, rawURL := range []string{
		"https://user:password@dav.example.com/dav",
		"https://user@dav.example.com/dav",
	} {
		t.Run("rejects URL userinfo "+rawURL, func(t *testing.T) {
			if _, err := parseWebDAVConfig(map[string]any{"url": rawURL}); !errors.Is(err, ErrWebDAVURLInvalid) {
				t.Fatalf("error = %v, want ErrWebDAVURLInvalid", err)
			}
		})
	}

	for _, tc := range []struct {
		name  string
		value string
	}{
		{name: "path dot segment", value: "../outside"},
		{name: "encoded dot segment", value: "/%2e%2e/outside"},
		{name: "query delimiter", value: "subdux?outside=1"},
	} {
		t.Run("rejects unsafe path "+tc.name, func(t *testing.T) {
			if _, err := parseWebDAVConfig(map[string]any{"url": "https://dav.example.com/dav", "path": tc.value}); !errors.Is(err, ErrWebDAVURLInvalid) {
				t.Fatalf("parseWebDAVConfig() error = %v, want ErrWebDAVURLInvalid", err)
			}
		})
	}
}

func TestCreateWebDAVDestinationRejectsEmbeddedURLCredentials(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	_, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "webdav",
		Config: `{"url":"https://user:password@dav.example.com/dav"}`,
	})
	if !errors.Is(err, ErrWebDAVURLInvalid) {
		t.Fatalf("CreateDestination() error = %v, want ErrWebDAVURLInvalid", err)
	}
}

func TestWebDAVDirectHrefNameRequiresConfiguredCollectionAndHost(t *testing.T) {
	target := &webdavTarget{baseURL: "https://dav.example.com/dav/subdux"}
	valid := "subdux-backup-20260615-030000-abc.zip"
	for _, tc := range []struct {
		name string
		href string
		want string
	}{
		{name: "direct child", href: "/dav/subdux/" + valid, want: valid},
		// Servers may answer PROPFIND with a fully qualified href, and names come
		// back percent-encoded; both must still resolve to the bare archive name
		// because retention deletes by that name.
		{name: "absolute url on same host", href: "https://dav.example.com/dav/subdux/" + valid, want: valid},
		{name: "percent encoded name", href: "/dav/subdux/subdux%2Dbackup%2D20260615%2D030000%2Dabc.zip", want: valid},
		{name: "nested child", href: "/dav/subdux/nested/" + valid},
		{name: "sibling collection", href: "/dav/other/" + valid},
		{name: "dot segment", href: "/dav/subdux/../" + valid},
		{name: "other host", href: "https://other.example.com/dav/subdux/" + valid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := target.directHrefName(tc.href)
			if tc.want == "" {
				if ok {
					t.Fatalf("directHrefName() = %q, want rejection", got)
				}
				return
			}
			if !ok || got != tc.want {
				t.Fatalf("directHrefName() = %q, %v, want %q, true", got, ok, tc.want)
			}
		})
	}
}

func TestWebDAVPutPropagatesMKCOLFailure(t *testing.T) {
	var putCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "MKCOL" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodPut {
			putCount++
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}
	if err := target.Put(context.Background(), "subdux-backup-20260615-030000-abc.zip", strings.NewReader("archive"), 7); err == nil {
		t.Fatal("Put() error = nil, want MKCOL failure")
	}
	if putCount != 0 {
		t.Fatalf("PUT requests = %d, want none after MKCOL failure", putCount)
	}
}

func TestWebDAVPutRejectsUnsafeObjectNameBeforeRequest(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)
	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}
	if err := target.Put(context.Background(), "../outside.zip", strings.NewReader("archive"), 7); !errors.Is(err, ErrInvalidBackupObjectName) {
		t.Fatalf("Put() error = %v, want ErrInvalidBackupObjectName", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want none for unsafe object name", requests)
	}
}

func TestCreateWebDAVDestinationEncryptsSecret(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "webdav",
		Config: `{"url":"https://dav.example.com/dav","username":"u","password":"topsecret"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	config := decodeStoredConfig(t, svc, created.ID)
	if config["password"] != "topsecret" {
		t.Fatalf("decoded password = %v, want topsecret", config["password"])
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
		t.Fatal("created webdav destination not listed")
	}
	if strings.Contains(view.Config, "topsecret") {
		t.Fatal("listed config leaked the webdav password")
	}
	if len(view.ConfiguredSecretFields) != 1 || view.ConfiguredSecretFields[0] != "password" {
		t.Fatalf("configured secret fields = %v, want [password]", view.ConfiguredSecretFields)
	}
}

// stubWebDAVServer is a minimal WebDAV endpoint: PUT stores bytes, DELETE
// removes them, PROPFIND returns a Depth-1 multistatus listing, MKCOL is a
// no-op OK. It ignores auth; it only records objects in memory keyed by the
// request path.
type stubWebDAVServer struct {
	mu      sync.Mutex
	base    string // path prefix objects live under, e.g. "/dav"
	objects map[string][]byte
}

func (s *stubWebDAVServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch r.Method {
	case "MKCOL":
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		s.objects[r.URL.Path] = body
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		if _, ok := s.objects[r.URL.Path]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		delete(s.objects, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		body, ok := s.objects[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	case "PROPFIND":
		s.writePropfind(w)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *stubWebDAVServer) writePropfind(w http.ResponseWriter) {
	var entries strings.Builder
	// The collection itself.
	fmt.Fprintf(&entries,
		`<d:response><d:href>%s/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`,
		s.base)
	for path, body := range s.objects {
		fmt.Fprintf(&entries,
			`<d:response><d:href>%s</d:href><d:propstat><d:prop><d:getcontentlength>%d</d:getcontentlength><d:getlastmodified>Mon, 15 Jun 2026 03:00:00 GMT</d:getlastmodified><d:resourcetype/></d:prop><d:status>HTTP/1.1 200 OK</d:status></d:propstat></d:response>`,
			path, len(body))
	}
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusMultiStatus)
	fmt.Fprintf(w,
		`<?xml version="1.0"?><d:multistatus xmlns:d="DAV:">%s</d:multistatus>`,
		entries.String())
}

// TestWebDAVTargetRoundTrip drives Put, List, and Delete against the stub to
// confirm the request wiring, path joining, basic-auth, PROPFIND parsing, and
// filename-pattern gating work end to end.
func TestWebDAVTargetRoundTrip(t *testing.T) {
	stub := &stubWebDAVServer{base: "/dav", objects: map[string][]byte{}}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	config := map[string]any{
		"url":             server.URL + "/dav",
		"username":        "user",
		"password":        "pass",
		"retention_count": 5,
	}
	target, err := newWebDAVTarget(config, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}

	const name = "subdux-backup-20260615-030000-abc123.zip"
	payload := []byte("encrypted-archive-bytes")
	if err := target.Put(context.Background(), name, strings.NewReader(string(payload)), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if _, ok := stub.objects["/dav/"+name]; !ok {
		t.Fatalf("stored paths = %v, want /dav/%s", keysOf(stub.objects), name)
	}

	objects, err := target.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("List() = %d objects, want 1 (collection excluded, pattern-gated)", len(objects))
	}
	if objects[0].Name != name {
		t.Fatalf("listed name = %q, want %q", objects[0].Name, name)
	}
	if objects[0].Size != int64(len(payload)) {
		t.Fatalf("listed size = %d, want %d", objects[0].Size, len(payload))
	}

	if err := target.Delete(context.Background(), name); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, ok := stub.objects["/dav/"+name]; ok {
		t.Fatal("object still present after Delete")
	}
}

func TestWebDAVTargetGetRoundTrip(t *testing.T) {
	stub := &stubWebDAVServer{base: "/dav", objects: map[string][]byte{}}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	target, err := newWebDAVTarget(map[string]any{
		"url":      server.URL + "/dav",
		"username": "user",
		"password": "pass",
	}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}

	const name = "subdux-backup-20260615-030000-abc123.zip"
	payload := []byte("encrypted-archive-bytes")
	if err := target.Put(context.Background(), name, strings.NewReader(string(payload)), int64(len(payload))); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	reader, size, err := target.Get(context.Background(), name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if size != int64(len(payload)) {
		t.Fatalf("Get() size = %d, want %d", size, len(payload))
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Get() contents = %q, want %q", got, payload)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

func TestWebDAVGetRejectsUnsafeObjectNameBeforeRequest(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}
	if _, _, err := target.Get(context.Background(), "../outside.zip"); !errors.Is(err, ErrInvalidBackupObjectName) {
		t.Fatalf("Get() error = %v, want ErrInvalidBackupObjectName", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want none for unsafe object name", requests)
	}
}

func TestWebDAVGetPropagatesStatusFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}
	if _, _, err := target.Get(context.Background(), "subdux-backup-20260615-030000-abc123.zip"); err == nil {
		t.Fatal("Get() error = nil, want non-nil for 404 status")
	}
}

func TestWebDAVRetentionFailsClosedWhenLastModifiedUnavailable(t *testing.T) {
	tests := []struct {
		name         string
		lastModified string
		wantError    bool
		wantDeletes  int
	}{
		{name: "missing", lastModified: "", wantError: true},
		{name: "invalid", lastModified: "not-a-http-date", wantError: true},
		{name: "valid", lastModified: "Tue, 16 Jun 2026 03:00:00 GMT", wantDeletes: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var deleteCount int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case "PROPFIND":
					lastModified := ""
					if tc.lastModified != "" {
						lastModified = fmt.Sprintf("<d:getlastmodified>%s</d:getlastmodified>", tc.lastModified)
					}
					w.Header().Set("Content-Type", "application/xml")
					w.WriteHeader(http.StatusMultiStatus)
					fmt.Fprintf(w, `<?xml version="1.0"?><d:multistatus xmlns:d="DAV:">
						<d:response><d:href>/dav/</d:href><d:propstat><d:prop><d:resourcetype><d:collection/></d:resourcetype></d:prop></d:propstat></d:response>
						<d:response><d:href>/dav/subdux-backup-20260616-030000-new.zip</d:href><d:propstat><d:prop><d:getcontentlength>1</d:getcontentlength>%s<d:resourcetype/></d:prop></d:propstat></d:response>
						<d:response><d:href>/dav/subdux-backup-20260615-030000-old.zip</d:href><d:propstat><d:prop><d:getcontentlength>1</d:getcontentlength><d:getlastmodified>Mon, 15 Jun 2026 03:00:00 GMT</d:getlastmodified><d:resourcetype/></d:prop></d:propstat></d:response>
					</d:multistatus>`, lastModified)
				case http.MethodDelete:
					deleteCount++
					w.WriteHeader(http.StatusNoContent)
				default:
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			target, err := newWebDAVTarget(map[string]any{
				"url":             server.URL + "/dav",
				"retention_count": 1,
			}, nil)
			if err != nil {
				t.Fatalf("newWebDAVTarget() error = %v", err)
			}

			err = applyTargetRetention(context.Background(), target)
			if tc.wantError && err == nil {
				t.Error("applyTargetRetention() error = nil, want timestamp validation error")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("applyTargetRetention() error = %v, want nil", err)
			}
			if deleteCount != tc.wantDeletes {
				t.Fatalf("DELETE requests = %d, want %d", deleteCount, tc.wantDeletes)
			}
		})
	}
}

type retentionTestTarget struct {
	objects      []BackupObject
	retention    int
	deletedNames []string
}

func (t *retentionTestTarget) Type() string { return "test" }

func (t *retentionTestTarget) Put(context.Context, string, io.Reader, int64) error {
	return nil
}

func (t *retentionTestTarget) List(context.Context) ([]BackupObject, error) {
	return append([]BackupObject(nil), t.objects...), nil
}

func (t *retentionTestTarget) Get(context.Context, string) (io.ReadCloser, int64, error) {
	return io.NopCloser(strings.NewReader("")), 0, nil
}

func (t *retentionTestTarget) Delete(_ context.Context, name string) error {
	t.deletedNames = append(t.deletedNames, name)
	return nil
}

func (t *retentionTestTarget) RetentionCount() int { return t.retention }

func TestApplyTargetRetentionRejectsUnknownModifiedAtBeforeDelete(t *testing.T) {
	target := &retentionTestTarget{
		retention: 1,
		objects: []BackupObject{
			{Name: "subdux-backup-20260616-030000-new.zip"},
			{
				Name:       "subdux-backup-20260615-030000-old.zip",
				ModifiedAt: time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC),
			},
		},
	}

	if err := applyTargetRetention(context.Background(), target); err == nil {
		t.Fatal("applyTargetRetention() error = nil, want timestamp validation error")
	}
	if len(target.deletedNames) != 0 {
		t.Fatalf("deleted names = %v, want none when ordering is uncertain", target.deletedNames)
	}
}

// TestWebDAVListMissingCollectionIsEmpty confirms a 404 on the collection is
// treated as an empty listing rather than an error, so a first run's retention
// pass is a no-op.
func TestWebDAVListMissingCollectionIsEmpty(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}
	objects, err := target.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v, want nil for 404 collection", err)
	}
	if len(objects) != 0 {
		t.Fatalf("List() = %d objects, want 0", len(objects))
	}
}

// TestTestDestinationProbesWithoutWriting confirms the connectivity probe lists
// (read-only) and reports the existing archive count.
func TestTestDestinationProbesWithoutWriting(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	stub := &stubWebDAVServer{base: "/dav", objects: map[string][]byte{
		"/dav/subdux-backup-20260615-030000-aaa111.zip": []byte("x"),
	}}
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "webdav",
		Config: fmt.Sprintf(`{"url":%q,"username":"u","password":"p"}`, server.URL+"/dav"),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	count, err := svc.TestDestination(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("TestDestination() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("TestDestination() count = %d, want 1", count)
	}
}

// TestWebDAVTargetGetStreamsAfterReturnAndChunked drives Get against a server
// that writes its response body in two flushed chunks with no Content-Length
// header (forcing chunked transfer encoding), with a short sleep between
// chunks. It asserts the reported size is -1 (the documented "unreported"
// contract for a chunked response) and that the full body can still be read
// after Get has already returned. This fails if Get ever regresses to
// `defer cancel()`, which would tear the request context down the moment Get
// hands the reader back, aborting the still-streaming body.
func TestWebDAVTargetGetStreamsAfterReturnAndChunked(t *testing.T) {
	const part1 = "first-chunk-bytes-"
	const part2 = "second-chunk-bytes"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// No Content-Length is set: writing and flushing before the handler
		// returns forces net/http into chunked transfer encoding.
		_, _ = io.WriteString(w, part1)
		flusher.Flush()
		time.Sleep(50 * time.Millisecond)
		_, _ = io.WriteString(w, part2)
	}))
	t.Cleanup(server.Close)

	target, err := newWebDAVTarget(map[string]any{"url": server.URL + "/dav"}, nil)
	if err != nil {
		t.Fatalf("newWebDAVTarget() error = %v", err)
	}

	reader, size, err := target.Get(context.Background(), "subdux-backup-20260615-030000-abc123.zip")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if size != -1 {
		t.Fatalf("Get() size = %d, want -1 (unreported for a chunked response)", size)
	}

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read archive after Get returned: %v", err)
	}
	if string(got) != part1+part2 {
		t.Fatalf("Get() contents = %q, want %q", got, part1+part2)
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}

// TestBackupObjectReaderCloseReleasesCancel proves backupObjectReader.Close
// both closes the underlying body and releases the context it was armed
// with, independent of any particular target's wiring.
func TestBackupObjectReaderCloseReleasesCancel(t *testing.T) {
	called := false
	reader := &backupObjectReader{
		ReadCloser: io.NopCloser(strings.NewReader("x")),
		cancel:     func() { called = true },
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
	if !called {
		t.Fatal("Close() did not call cancel")
	}
}
