package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v4"
)

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	server := newHTTPServer(":8080", http.NewServeMux())

	if got, want := server.Addr, ":8080"; got != want {
		t.Fatalf("Addr = %q, want %q", got, want)
	}
	if got, want := server.ReadHeaderTimeout, httpReadHeaderTimeout; got != want {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := server.ReadTimeout, httpReadTimeout; got != want {
		t.Fatalf("ReadTimeout = %s, want %s", got, want)
	}
	if got, want := server.WriteTimeout, httpWriteTimeout; got != want {
		t.Fatalf("WriteTimeout = %s, want %s", got, want)
	}
	if got, want := server.IdleTimeout, httpIdleTimeout; got != want {
		t.Fatalf("IdleTimeout = %s, want %s", got, want)
	}
}

func TestCleanUploadedAssetPathAllowsOnlyManagedIcons(t *testing.T) {
	tests := []struct {
		raw    string
		wantOK bool
	}{
		{raw: "icons/logo.png", wantOK: true},
		{raw: "/icons/logo.jpeg", wantOK: true},
		{raw: "icons/logo.ico", wantOK: true},
		{raw: "logo.png", wantOK: false},
		{raw: "icons/nested/logo.png", wantOK: false},
		{raw: "icons/logo.svg", wantOK: false},
		{raw: "icons/pwn.html", wantOK: false},
		{raw: "../subdux.db", wantOK: false},
		{raw: "icons/%2e%2e/pwn.png", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, gotOK := cleanUploadedAssetPath(tt.raw)
			if gotOK != tt.wantOK {
				t.Fatalf("cleanUploadedAssetPath(%q) ok = %v, want %v", tt.raw, gotOK, tt.wantOK)
			}
		})
	}
}

func TestServeUploadedAssetAddsNoScriptHeaders(t *testing.T) {
	assetsRoot := t.TempDir()
	iconDir := filepath.Join(assetsRoot, "icons")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		t.Fatalf("failed to create icon dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(iconDir, "logo.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("failed to write icon: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/uploads/icons/logo.png", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("*")
	c.SetParamValues("icons/logo.png")

	if err := serveUploadedAsset(c, assetsRoot); err != nil {
		t.Fatalf("serveUploadedAsset() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Header().Get("Content-Type"), "image/png"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got, want := rec.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
		t.Fatalf("X-Content-Type-Options = %q, want %q", got, want)
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'none'") || !strings.Contains(got, "sandbox") {
		t.Fatalf("Content-Security-Policy = %q, want no-script sandbox policy", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "inline;") || !strings.Contains(got, "logo.png") {
		t.Fatalf("Content-Disposition = %q, want inline filename", got)
	}
}

func TestServeUploadedAssetRejectsExecutableFiles(t *testing.T) {
	assetsRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(assetsRoot, "pwn.html"), []byte("<script>evil()</script>"), 0o644); err != nil {
		t.Fatalf("failed to write html: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/uploads/pwn.html", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("*")
	c.SetParamValues("pwn.html")

	err := serveUploadedAsset(c, assetsRoot)
	if err == nil {
		t.Fatal("serveUploadedAsset() error = nil, want not found")
	}
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusNotFound {
		t.Fatalf("serveUploadedAsset() error = %v, want 404", err)
	}
}

func TestSetupSPANoStoresIndexAndFallback(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":          &fstest.MapFile{Data: []byte("<!doctype html><script src=\"/assets/index-new.js\"></script>")},
		"assets/index-new.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	tests := []struct {
		name string
		path string
	}{
		{name: "root", path: "/"},
		{name: "index", path: "/index.html"},
		{name: "fallback", path: "/settings/account"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			setupSPA(e, dist)

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
				t.Fatalf("Content-Type = %q, want %q", got, want)
			}
			if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") || !strings.Contains(got, "must-revalidate") {
				t.Fatalf("Cache-Control = %q, want no-store and must-revalidate", got)
			}
			if got, want := rec.Header().Get("Pragma"), "no-cache"; got != want {
				t.Fatalf("Pragma = %q, want %q", got, want)
			}
			if got, want := rec.Header().Get("Expires"), "0"; got != want {
				t.Fatalf("Expires = %q, want %q", got, want)
			}
		})
	}
}

func TestSetupSPAServesStaticAsset(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":          &fstest.MapFile{Data: []byte("<!doctype html>")},
		"assets/index-new.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
	}

	e := echo.New()
	setupSPA(e, dist)

	req := httptest.NewRequest(http.MethodGet, "/assets/index-new.js", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatal("asset response returned index.html")
	}
	if got := rec.Body.String(); got != "console.log('ok')" {
		t.Fatalf("body = %q, want asset content", got)
	}
	if got, want := rec.Header().Get("Cache-Control"), immutableCacheControl; got != want {
		t.Fatalf("Cache-Control = %q, want %q", got, want)
	}
}

func TestSetupSPAFallbackDoesNotCacheImmutable(t *testing.T) {
	dist := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html>")},
	}

	e := echo.New()
	setupSPA(e, dist)

	// A path under /assets/ that does not exist must fall back to index.html,
	// and must never inherit the immutable cache header (otherwise a hashed URL
	// that 404s would be cached forever as the SPA shell).
	req := httptest.NewRequest(http.MethodGet, "/assets/missing-Dead0000.js", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Cache-Control"); !strings.Contains(got, "no-store") {
		t.Fatalf("Cache-Control = %q, want no-store fallback", got)
	}
}
