package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
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
