package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/pkg"
)

func TestBeginOIDCLoginCachesProviderDiscovery(t *testing.T) {
	db := newTestDB(t)
	authService := NewAuthService(db)

	var discoveryHits atomic.Int32
	var issuerURL string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		discoveryHits.Add(1)
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
	defer provider.Close()
	issuerURL = provider.URL

	seedSystemSetting(t, db, "oidc_enabled", "true")
	seedSystemSetting(t, db, "oidc_issuer_url", issuerURL)
	seedSystemSetting(t, db, "oidc_client_id", "client-id")
	seedSystemSetting(t, db, "oidc_client_secret", "client-secret")
	seedSystemSetting(t, db, "oidc_redirect_url", "https://app.example.com/api/auth/oidc/callback")

	first, err := authService.BeginOIDCLogin()
	if err != nil {
		t.Fatalf("first BeginOIDCLogin() error = %v, want nil", err)
	}
	if !strings.HasPrefix(first.AuthorizationURL, issuerURL+"/authorize?") {
		t.Fatalf("first AuthorizationURL = %q, want provider authorize endpoint", first.AuthorizationURL)
	}

	second, err := authService.WithContext(t.Context()).BeginOIDCLogin()
	if err != nil {
		t.Fatalf("second BeginOIDCLogin() error = %v, want nil", err)
	}
	if !strings.HasPrefix(second.AuthorizationURL, issuerURL+"/authorize?") {
		t.Fatalf("second AuthorizationURL = %q, want provider authorize endpoint", second.AuthorizationURL)
	}

	if got := discoveryHits.Load(); got != 1 {
		t.Fatalf("discovery hits = %d, want 1", got)
	}

	authService.oidcMu.Lock()
	entry := authService.oidcProviderCache[issuerURL]
	entry.ExpiresAt = pkg.NowUTC().Add(-time.Second)
	authService.oidcProviderCache[issuerURL] = entry
	authService.oidcMu.Unlock()

	if _, err := authService.BeginOIDCLogin(); err != nil {
		t.Fatalf("BeginOIDCLogin() after cache expiry error = %v, want nil", err)
	}
	if got := discoveryHits.Load(); got != 2 {
		t.Fatalf("discovery hits after cache expiry = %d, want 2", got)
	}
}

func TestBeginOIDCReauthRequiresFreshLoginAuthorizationParams(t *testing.T) {
	db := newTestDB(t)
	authService := NewAuthService(db)

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
	defer provider.Close()
	issuerURL = provider.URL

	seedSystemSetting(t, db, "oidc_enabled", "true")
	seedSystemSetting(t, db, "oidc_issuer_url", issuerURL)
	seedSystemSetting(t, db, "oidc_client_id", "client-id")
	seedSystemSetting(t, db, "oidc_client_secret", "client-secret")
	seedSystemSetting(t, db, "oidc_redirect_url", "https://app.example.com/api/auth/oidc/callback")
	seedSystemSetting(t, db, "oidc_extra_auth_params", "prompt=none&max_age=3600&login_hint=admin@example.com")

	start, err := authService.BeginOIDCReauth(1, ReauthOperationBackup)
	if err != nil {
		t.Fatalf("BeginOIDCReauth() error = %v, want nil", err)
	}

	parsed, err := url.Parse(start.AuthorizationURL)
	if err != nil {
		t.Fatalf("failed to parse AuthorizationURL: %v", err)
	}
	values := parsed.Query()
	if got := values.Get("prompt"); got != "login" {
		t.Fatalf("prompt = %q, want login", got)
	}
	if got := values.Get("max_age"); got != "0" {
		t.Fatalf("max_age = %q, want 0", got)
	}
	if got := values.Get("login_hint"); got != "admin@example.com" {
		t.Fatalf("login_hint = %q, want preserved extra auth param", got)
	}
}
