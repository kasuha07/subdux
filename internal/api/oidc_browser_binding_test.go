package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/labstack/echo/v4"
)

func findResponseCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func startOIDCLoginForBindingTest(t *testing.T, e *echo.Echo) (state string, binding *http.Cookie) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/login/start", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login start status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode login start response: %v", err)
	}
	parsed, err := url.Parse(resp.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL: %v", err)
	}
	state = parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("authorization URL %q has no state", resp.AuthorizationURL)
	}

	binding = findResponseCookie(rec, apimw.OIDCBindingCookieName)
	if binding == nil || binding.Value == "" {
		t.Fatalf("login start did not set %s cookie; set-cookie = %q", apimw.OIDCBindingCookieName, rec.Header().Values(echo.HeaderSetCookie))
	}
	if !binding.HttpOnly || binding.SameSite != http.SameSiteLaxMode || binding.Path != "/api/auth/oidc/callback" || binding.MaxAge <= 0 {
		t.Fatalf("binding cookie attributes = %+v, want HttpOnly SameSite=Lax Path=/api/auth/oidc/callback with MaxAge", binding)
	}
	if strings.Contains(rec.Body.String(), binding.Value) {
		t.Fatalf("login start body leaks the binding value: %s", rec.Body.String())
	}
	return state, binding
}

func runOIDCCallbackForBindingTest(t *testing.T, e *echo.Echo, state string, binding *http.Cookie) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/callback?state="+url.QueryEscape(state)+"&code=lifted-code", nil)
	if binding != nil {
		req.AddCookie(binding)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("callback status = %d, want %d; body = %s", rec.Code, http.StatusFound, rec.Body.String())
	}
	if got := rec.Header().Get(echo.HeaderLocation); got != "/login?oidc_action=login" {
		t.Fatalf("callback redirect = %q, want /login?oidc_action=login", got)
	}
	cleared := findResponseCookie(rec, apimw.OIDCBindingCookieName)
	if cleared == nil || cleared.MaxAge >= 0 {
		t.Fatalf("callback must clear the binding cookie; got %+v", cleared)
	}
	session := findResponseCookie(rec, apimw.OIDCSessionCookieName)
	if session == nil || session.Value == "" {
		t.Fatalf("callback did not set %s cookie", apimw.OIDCSessionCookieName)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/auth/oidc/session", nil)
	req.AddCookie(session)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("session status = %d, want an error for a fake code; body = %s", rec.Code, rec.Body.String())
	}
	return rec.Body.String()
}

func TestOIDCCallbackRejectsCallbackFromAnotherBrowser(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	provider := seedOIDCConnectTestProvider(t, db)
	defer provider.Close()
	e := newHumanOnlyRouteTestServer(t, db)

	t.Run("lifted callback without the binding cookie", func(t *testing.T) {
		state, _ := startOIDCLoginForBindingTest(t, e)
		body := runOIDCCallbackForBindingTest(t, e, state, nil)
		if !hasErrorCode(body, "oidc_browser_binding_mismatch") {
			t.Fatalf("body = %s, want oidc_browser_binding_mismatch", body)
		}
	})

	t.Run("lifted callback with another flow's binding cookie", func(t *testing.T) {
		attackerState, _ := startOIDCLoginForBindingTest(t, e)
		_, victimBinding := startOIDCLoginForBindingTest(t, e)
		body := runOIDCCallbackForBindingTest(t, e, attackerState, victimBinding)
		if !hasErrorCode(body, "oidc_browser_binding_mismatch") {
			t.Fatalf("body = %s, want oidc_browser_binding_mismatch", body)
		}
	})

	t.Run("callback in the initiating browser passes the binding check", func(t *testing.T) {
		state, binding := startOIDCLoginForBindingTest(t, e)
		body := runOIDCCallbackForBindingTest(t, e, state, binding)
		// The fake provider has no token endpoint, so the flow fails later at
		// code exchange — but never on the browser binding.
		if hasErrorCode(body, "oidc_browser_binding_mismatch") {
			t.Fatalf("body = %s, matching binding must pass the binding check", body)
		}
	})
}

func TestOIDCReauthStartSetsBrowserBindingCookie(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	provider := seedOIDCConnectTestProvider(t, db)
	defer provider.Close()

	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	req := httptest.NewRequest(http.MethodPost, "/api/reauth/oidc/start", strings.NewReader(`{"operation":"backup"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("reauth start status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	binding := findResponseCookie(rec, apimw.OIDCBindingCookieName)
	if binding == nil || binding.Value == "" {
		t.Fatalf("reauth start did not set %s cookie", apimw.OIDCBindingCookieName)
	}
	if binding.Path != "/api/auth/oidc/callback" || !binding.HttpOnly || binding.SameSite != http.SameSiteLaxMode {
		t.Fatalf("binding cookie attributes = %+v", binding)
	}
	if strings.Contains(rec.Body.String(), binding.Value) {
		t.Fatalf("reauth start body leaks the binding value: %s", rec.Body.String())
	}
}
