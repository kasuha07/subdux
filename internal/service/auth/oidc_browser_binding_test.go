package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

func seedOIDCBindingTestProvider(t *testing.T, authService *Service) {
	t.Helper()

	var issuerURL string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
			"issuer": %q,
			"authorization_endpoint": %q,
			"token_endpoint": %q,
			"jwks_uri": %q,
			"response_types_supported": ["code"],
			"subject_types_supported": ["public"],
			"id_token_signing_alg_values_supported": ["RS256"]
		}`, issuerURL, issuerURL+"/authorize", issuerURL+"/token", issuerURL+"/jwks")
	}))
	t.Cleanup(provider.Close)
	issuerURL = provider.URL

	seedSystemSetting(t, authService.DB, "oidc_enabled", "true")
	seedSystemSetting(t, authService.DB, "oidc_issuer_url", issuerURL)
	seedSystemSetting(t, authService.DB, "oidc_client_id", "client-id")
	seedSystemSetting(t, authService.DB, "oidc_client_secret", "client-secret")
	seedSystemSetting(t, authService.DB, "oidc_redirect_url", "https://app.example.com/api/auth/oidc/callback")
}

func oidcStateFromAuthorizationURL(t *testing.T, authorizationURL string) string {
	t.Helper()
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatalf("failed to parse authorization URL: %v", err)
	}
	state := parsed.Query().Get("state")
	if state == "" {
		t.Fatalf("authorization URL %q has no state", authorizationURL)
	}
	return state
}

func TestBeginOIDCFlowsMintDistinctBrowserBindings(t *testing.T) {
	authService := NewService(newTestDB(t))
	seedOIDCBindingTestProvider(t, authService)

	login, err := authService.BeginOIDCLogin()
	if err != nil {
		t.Fatalf("BeginOIDCLogin() error = %v", err)
	}
	connect, err := authService.BeginOIDCConnect(1)
	if err != nil {
		t.Fatalf("BeginOIDCConnect() error = %v", err)
	}
	reauth, err := authService.BeginOIDCReauth(1, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("BeginOIDCReauth() error = %v", err)
	}

	seen := map[string]bool{}
	for name, start := range map[string]*OIDCStartResult{"login": login, "connect": connect, "reauth": reauth} {
		if start.BrowserBinding == "" {
			t.Fatalf("%s BrowserBinding is empty", name)
		}
		if seen[start.BrowserBinding] {
			t.Fatalf("%s BrowserBinding reused across flows", name)
		}
		seen[start.BrowserBinding] = true

		state := oidcStateFromAuthorizationURL(t, start.AuthorizationURL)
		if state == start.BrowserBinding || strings.Contains(start.AuthorizationURL, start.BrowserBinding) {
			t.Fatalf("%s BrowserBinding leaks into the authorization URL", name)
		}

		authService.oidcMu.Lock()
		stored := authService.oidcStateSessions[state].BrowserBinding
		authService.oidcMu.Unlock()
		if stored != start.BrowserBinding {
			t.Fatalf("%s stored binding = %q, want %q", name, stored, start.BrowserBinding)
		}

		payload, err := json.Marshal(start)
		if err != nil {
			t.Fatalf("Marshal(%s) error = %v", name, err)
		}
		if strings.Contains(string(payload), start.BrowserBinding) {
			t.Fatalf("%s JSON response leaks BrowserBinding: %s", name, payload)
		}
	}
}

func TestHandleOIDCCallbackRequiresMatchingBrowserBinding(t *testing.T) {
	cases := []struct {
		name      string
		purpose   string
		presented string
	}{
		{name: "login without cookie", purpose: oidcPurposeLogin, presented: ""},
		{name: "login with foreign cookie", purpose: oidcPurposeLogin, presented: "attacker-browser-binding"},
		{name: "connect without cookie", purpose: oidcPurposeConnect, presented: ""},
		{name: "connect with foreign cookie", purpose: oidcPurposeConnect, presented: "attacker-browser-binding"},
		{name: "reauth without cookie", purpose: oidcPurposeReauth, presented: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			authService := NewService(db)
			user := createTestUser(t, db)

			authService.storeOIDCStateSession("state-1", oidcStateSession{
				Purpose:        tc.purpose,
				UserID:         user.ID,
				CodeVerifier:   "verifier",
				Nonce:          "nonce",
				BrowserBinding: "victim-browser-binding",
				ExpiresAt:      pkg.NowUTC().Add(oidcStateSessionTTL),
			})

			callback, err := authService.HandleOIDCCallback("state-1", tc.presented, "attacker-code", "", "")
			if err != nil {
				t.Fatalf("HandleOIDCCallback() error = %v", err)
			}
			if callback.Purpose != tc.purpose {
				t.Fatalf("callback purpose = %q, want %q", callback.Purpose, tc.purpose)
			}

			result, err := authService.ConsumeOIDCSessionResult(callback.SessionID)
			if err != nil {
				t.Fatalf("ConsumeOIDCSessionResult() error = %v", err)
			}
			if result.ErrorCode != "oidc_browser_binding_mismatch" {
				t.Fatalf("error code = %q, want oidc_browser_binding_mismatch", result.ErrorCode)
			}
			if result.Token != "" || result.RefreshToken != "" || result.Connected {
				t.Fatalf("mismatched binding must not issue tokens or connect: %+v", result)
			}

			var connections int64
			if err := db.Model(&model.OIDCConnection{}).Count(&connections).Error; err != nil {
				t.Fatalf("count connections: %v", err)
			}
			if connections != 0 {
				t.Fatalf("connections = %d, want 0", connections)
			}

			// The state is spent even on mismatch, so the lifted callback URL
			// cannot be retried with a different cookie.
			retry, err := authService.HandleOIDCCallback("state-1", "victim-browser-binding", "attacker-code", "", "")
			if err != nil {
				t.Fatalf("retry HandleOIDCCallback() error = %v", err)
			}
			retryResult, err := authService.ConsumeOIDCSessionResult(retry.SessionID)
			if err != nil {
				t.Fatalf("retry ConsumeOIDCSessionResult() error = %v", err)
			}
			if retryResult.ErrorCode != "invalid_or_expired_oidc_session" {
				t.Fatalf("retry error code = %q, want invalid_or_expired_oidc_session", retryResult.ErrorCode)
			}
		})
	}
}

func TestHandleOIDCCallbackAcceptsMatchingBrowserBinding(t *testing.T) {
	authService := NewService(newTestDB(t))

	authService.storeOIDCStateSession("state-1", oidcStateSession{
		Purpose:        oidcPurposeLogin,
		CodeVerifier:   "verifier",
		Nonce:          "nonce",
		BrowserBinding: "browser-binding",
		ExpiresAt:      pkg.NowUTC().Add(oidcStateSessionTTL),
	})

	// An empty code fails after the binding check, proving the binding passed.
	callback, err := authService.HandleOIDCCallback("state-1", " browser-binding ", "", "", "")
	if err != nil {
		t.Fatalf("HandleOIDCCallback() error = %v", err)
	}
	result, err := authService.ConsumeOIDCSessionResult(callback.SessionID)
	if err != nil {
		t.Fatalf("ConsumeOIDCSessionResult() error = %v", err)
	}
	if result.ErrorCode != "missing_oidc_authorization_code" {
		t.Fatalf("error code = %q, want missing_oidc_authorization_code", result.ErrorCode)
	}
}

func TestHandleOIDCCallbackRejectsStateWithoutStoredBinding(t *testing.T) {
	authService := NewService(newTestDB(t))

	authService.storeOIDCStateSession("state-1", oidcStateSession{
		Purpose:   oidcPurposeLogin,
		ExpiresAt: pkg.NowUTC().Add(oidcStateSessionTTL),
	})

	callback, err := authService.HandleOIDCCallback("state-1", "", "", "", "")
	if err != nil {
		t.Fatalf("HandleOIDCCallback() error = %v", err)
	}
	result, err := authService.ConsumeOIDCSessionResult(callback.SessionID)
	if err != nil {
		t.Fatalf("ConsumeOIDCSessionResult() error = %v", err)
	}
	if result.ErrorCode != "oidc_browser_binding_mismatch" {
		t.Fatalf("error code = %q, want oidc_browser_binding_mismatch", result.ErrorCode)
	}
}
