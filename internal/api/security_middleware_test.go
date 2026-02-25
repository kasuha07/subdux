package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

func newSecurityMiddlewareTestContext(method string, target string, contentType string, body string) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestLoginAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/login",
		echo.MIMEApplicationForm,
		"identifier=Alice%40Example.com&password=secret",
	)

	got := loginAccountKey(c)
	want := "alice@example.com"
	if got != want {
		t.Fatalf("loginAccountKey() = %q, want %q", got, want)
	}
}

func TestRegisterAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/register",
		echo.MIMEApplicationForm,
		"email=Test%2Balias%40Example.com&username=alice",
	)

	got := registerAccountKey(c)
	want := "email:test+alias@example.com"
	if got != want {
		t.Fatalf("registerAccountKey() = %q, want %q", got, want)
	}
}

func TestEmailAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/password/forgot",
		echo.MIMEApplicationForm,
		"email=Recover%40Example.com",
	)

	got := emailAccountKey(c)
	want := "email:recover@example.com"
	if got != want {
		t.Fatalf("emailAccountKey() = %q, want %q", got, want)
	}
}

func TestReadRequestBodyAndRestorePreservesBodyForDownstream(t *testing.T) {
	body := `{"identifier":"Alice@example.com","password":"secret"}`
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMEApplicationJSON, body)

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
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMETextPlain, body)

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

func TestRequiredAPIKeyScopeUsesWriteForStateChangingGetRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/totp/setup", "", "")
	c.SetPath("/api/auth/totp/setup")

	got := requiredAPIKeyScope(c)
	if got != service.APIKeyScopeWrite {
		t.Fatalf("requiredAPIKeyScope() = %q, want %q", got, service.APIKeyScopeWrite)
	}
}

func TestRequiredAPIKeyScopeUsesReadForRegularGetRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/subscriptions", "", "")
	c.SetPath("/api/subscriptions")

	got := requiredAPIKeyScope(c)
	if got != service.APIKeyScopeRead {
		t.Fatalf("requiredAPIKeyScope() = %q, want %q", got, service.APIKeyScopeRead)
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

func TestAPIKeyScopeMiddlewareBlocksAuthNamespaceRoot(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth", "", "")
	c.SetPath("/api/auth")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeWrite},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if calledNext {
		t.Fatal("APIKeyScopeMiddleware() called next handler for /api/auth, want blocked")
	}

	if got := c.Response().Status; got != http.StatusForbidden {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusForbidden)
	}
}

func TestAPIKeyScopeMiddlewareBlocksRestrictedAuthRoutes(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/totp/setup", "", "")
	c.SetPath("/api/auth/totp/setup")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeWrite},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if calledNext {
		t.Fatal("APIKeyScopeMiddleware() called next handler for restricted auth route, want blocked")
	}

	if got := c.Response().Status; got != http.StatusForbidden {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusForbidden)
	}
}

func TestAPIKeyScopeMiddlewareAllowsAuthMeRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/me", "", "")
	c.SetPath("/api/auth/me")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeRead},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if !calledNext {
		t.Fatal("APIKeyScopeMiddleware() blocked /api/auth/me for api key, want allowed")
	}

	if got := c.Response().Status; got != http.StatusNoContent {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusNoContent)
	}
}
