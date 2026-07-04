package apimw

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"github.com/labstack/echo/v4"
)

func newMiddlewareTestContext(method, target, contentType, body string) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestReadRequestBodyAndRestorePreservesBodyForDownstream(t *testing.T) {
	body := `{"identifier":"Alice@example.com","password":"secret"}`
	c := newMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMEApplicationJSON, body)

	readBody, err := readRequestBodyAndRestore(c, 1024)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if string(readBody) != body {
		t.Fatalf("readRequestBodyAndRestore() body = %q, want %q", string(readBody), body)
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreSkipsLargeFixedLengthBody(t *testing.T) {
	body := strings.Repeat("a", 128)
	c := newMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMETextPlain, body)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreDoesNotTrustContentLengthForLimiterReads(t *testing.T) {
	body := strings.Repeat("z", 128)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.ContentLength = 1
	req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreSkipsLargeUnknownLengthBody(t *testing.T) {
	body := strings.Repeat("x", 64)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.ContentLength = -1
	req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestRequiredAPIKeyScopeUsesReadForRegularGetRoute(t *testing.T) {
	c := newMiddlewareTestContext(http.MethodGet, "/api/subscriptions", "", "")
	c.SetPath("/api/subscriptions")

	got := requiredAPIKeyScope(c)
	if got != apikeyservice.APIKeyScopeRead {
		t.Fatalf("requiredAPIKeyScope() = %q, want %q", got, apikeyservice.APIKeyScopeRead)
	}
}

func TestIsAPIKeyRouteAllowed(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/subscriptions", want: true},
		{path: "/api/auth", want: false},
		{path: "/api/auth/me", want: true},
		{path: "/api/auth/totp/setup", want: false},
		{path: "/api/auth/passkeys/register/start", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isAPIKeyRouteAllowed(tt.path)
			if got != tt.want {
				t.Fatalf("isAPIKeyRouteAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
