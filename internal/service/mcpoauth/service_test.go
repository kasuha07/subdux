package mcpoauth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setup(t *testing.T) (*Service, model.User, *ClientMetadata) {
	t.Helper()
	db := servicetest.NewDB(t).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.MCPOAuthClient{}, &model.MCPOAuthRequest{}, &model.MCPOAuthGrant{}, &model.MCPOAuthToken{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SystemSetting{Key: "site_url", Value: "https://subdux.example"}).Error; err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	client, err := s.Register(ClientMetadata{ClientName: "Test agent", RedirectURIs: []string{"http://127.0.0.1/callback", "https://client.example/callback"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, servicetest.CreateUser(t, db), client
}

func authorize(t *testing.T, s *Service, user model.User, client *ClientMetadata, write bool) (url.Values, string) {
	t.Helper()
	verifier := oauth2.GenerateVerifier()
	params := url.Values{"client_id": {client.ClientID}, "redirect_uri": {"http://127.0.0.1:54321/callback"}, "response_type": {"code"},
		"resource": {"https://subdux.example/mcp"}, "scope": {"read write offline_access"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}, "state": {"client-state"}}
	handle, err := s.Begin(params)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consent(handle, user.ID); err != nil {
		t.Fatal(err)
	}
	redirect, err := s.Decide(handle, user.ID, true, write)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(redirect)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != "client-state" || u.Query().Get("iss") != "https://subdux.example" {
		t.Fatal("lost authorization response binding")
	}
	return url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ClientID}, "redirect_uri": {params.Get("redirect_uri")}, "code": {u.Query().Get("code")}, "code_verifier": {verifier}, "resource": {params.Get("resource")}}, handle
}

func TestGrantLifecycleAndRefreshReplay(t *testing.T) {
	s, user, client := setup(t)
	params, handle := authorize(t, s, user, client, false)
	if _, err := s.Decide(handle, user.ID, true, true); err == nil {
		t.Fatal("interaction could be approved twice")
	}
	verifier := params.Get("code_verifier")
	params.Set("code_verifier", oauth2.GenerateVerifier())
	if _, err := s.Exchange(params); err == nil {
		t.Fatal("wrong PKCE verifier accepted")
	}
	// A caller without the verifier cannot consume the legitimate user's code.
	params.Set("code_verifier", verifier)
	token, err := s.Exchange(params)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(token.Scope, "write") || token.ExpiresIn <= 0 {
		t.Fatal("consent did not narrow scopes")
	}
	p, err := NewService(s.db).ValidateToken(token.AccessToken)
	if err != nil || p.UserID != user.ID || p.ClientID != client.ClientID {
		t.Fatal("persistent access token failed", err)
	}
	refreshParams := url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ClientID}, "resource": {"https://subdux.example/mcp"}, "refresh_token": {token.RefreshToken}}
	refreshParams.Set("scope", "read write")
	if _, err := s.Exchange(refreshParams); err == nil {
		t.Fatal("refresh expanded consent")
	}
	refreshParams.Del("scope")
	rotated, err := s.Exchange(refreshParams)
	if err != nil {
		t.Fatal(err)
	}
	if rotated.RefreshToken == token.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}
	if _, err := s.Exchange(refreshParams); err == nil {
		t.Fatal("refresh replay accepted")
	}
	if _, err := s.ValidateToken(rotated.AccessToken); err == nil {
		t.Fatal("refresh replay did not revoke token family")
	}
	refreshParams.Set("refresh_token", rotated.RefreshToken)
	if _, err := s.Exchange(refreshParams); err == nil {
		t.Fatal("revoked family could refresh")
	}
}

func TestTokenAndUserBoundaries(t *testing.T) {
	for _, test := range []string{"resource", "redirect", "client", "expiry", "disable", "revoke", "code-replay", "refresh-is-not-access"} {
		t.Run(test, func(t *testing.T) {
			s, user, client := setup(t)
			params, _ := authorize(t, s, user, client, true)
			switch test {
			case "resource":
				params.Set("resource", "https://another.example/mcp")
			case "redirect":
				params.Set("redirect_uri", "https://client.example/other")
			case "client":
				other, err := s.Register(ClientMetadata{RedirectURIs: client.RedirectURIs})
				if err != nil {
					t.Fatal(err)
				}
				params.Set("client_id", other.ClientID)
			}
			token, err := s.Exchange(params)
			if test == "resource" || test == "redirect" || test == "client" {
				if err == nil {
					t.Fatal("authorization code binding bypassed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			p, err := s.ValidateToken(token.AccessToken)
			if err != nil {
				t.Fatal(err)
			}
			switch test {
			case "expiry":
				if err := s.db.Model(&model.MCPOAuthToken{}).Where("hash = ?", hash(token.AccessToken)).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
					t.Fatal(err)
				}
			case "disable":
				if err := s.db.Model(&model.User{}).Where("id = ?", user.ID).Update("status", "disabled").Error; err != nil {
					t.Fatal(err)
				}
			case "revoke":
				if err := s.RevokeGrant(user.ID+1, p.GrantID); err == nil {
					t.Fatal("another user revoked the grant")
				}
				if err := s.RevokeGrant(user.ID, p.GrantID); err != nil {
					t.Fatal(err)
				}
			case "code-replay":
				if _, err := s.Exchange(params); err == nil {
					t.Fatal("authorization code replay accepted")
				}
			case "refresh-is-not-access":
				if _, err := s.ValidateToken(token.RefreshToken); err == nil {
					t.Fatal("refresh token accepted as access token")
				}
				return
			}
			if _, err := s.ValidateToken(token.AccessToken); err == nil {
				t.Fatal("invalid authorization remained usable")
			}
		})
	}
}

func TestCodeReplayWithoutVerifierDoesNotRevokeGrant(t *testing.T) {
	s, user, client := setup(t)
	params, _ := authorize(t, s, user, client, true)
	token, err := s.Exchange(params)
	if err != nil {
		t.Fatal(err)
	}
	params.Set("code_verifier", oauth2.GenerateVerifier())
	if _, err := s.Exchange(params); err == nil {
		t.Fatal("authorization code replay with an invalid verifier was accepted")
	}
	if _, err := s.ValidateToken(token.AccessToken); err != nil {
		t.Fatalf("a caller without the verifier revoked the legitimate grant: %v", err)
	}
}

func TestRejectedCodeExchangeDoesNotConsumeCode(t *testing.T) {
	for _, reason := range []string{"expired", "connection-limit"} {
		t.Run(reason, func(t *testing.T) {
			s, user, client := setup(t)
			params, handle := authorize(t, s, user, client, true)
			if reason == "expired" {
				if err := s.db.Model(&model.MCPOAuthRequest{}).Where("id = ?", hash(handle)).Update("expires_at", time.Now().Add(-time.Minute)).Error; err != nil {
					t.Fatal(err)
				}
			} else {
				seedOAuthTestGrants(t, s.db, user.ID, client, 50)
			}
			if _, err := s.Exchange(params); err == nil {
				t.Fatalf("code exchange ignored %s", reason)
			}
			var request model.MCPOAuthRequest
			if err := s.db.Where("id = ?", hash(handle)).Take(&request).Error; err != nil {
				t.Fatal(err)
			}
			if request.ConsumedAt != nil || request.GrantID != nil {
				t.Fatal("rejected exchange consumed the code or created a grant")
			}
		})
	}
}

func seedOAuthTestGrants(t *testing.T, db *gorm.DB, userID uint, client *ClientMetadata, count int) {
	t.Helper()
	grants := make([]model.MCPOAuthGrant, count)
	for i := range grants {
		grants[i] = model.MCPOAuthGrant{UserID: userID, ClientID: client.ClientID, ClientName: client.ClientName,
			Scopes: "read", Resource: "https://subdux.example/mcp", ExpiresAt: time.Now().Add(time.Hour)}
	}
	if err := db.Create(&grants).Error; err != nil {
		t.Fatal(err)
	}
}

func TestConsentOwnershipAndDenial(t *testing.T) {
	s, user, client := setup(t)
	verifier := oauth2.GenerateVerifier()
	params := url.Values{"client_id": {client.ClientID}, "redirect_uri": {"https://client.example/callback"}, "response_type": {"code"},
		"resource": {"https://subdux.example/mcp"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}}
	handle, err := s.Begin(params)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Decide(handle, user.ID, true, true); err == nil {
		t.Fatal("approval without a displayed, user-bound request accepted")
	}
	if _, err := s.Consent(handle, user.ID); err != nil {
		t.Fatal(err)
	}
	other := model.User{Username: "other", Email: "other@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := s.db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Consent(handle, other.ID); err == nil {
		t.Fatal("request ownership changed")
	}
	if _, err := s.Decide(handle, other.ID, true, true); err == nil {
		t.Fatal("another user approved a bound request")
	}
	redirect, err := s.Decide(handle, user.ID, false, false)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(redirect)
	if u.Query().Get("error") != "access_denied" || u.Query().Get("code") != "" {
		t.Fatal("denial issued a code")
	}
}

func TestRejectUnsafeAuthorizationAndClientMetadata(t *testing.T) {
	s, _, client := setup(t)
	for _, redirect := range []string{"http://evil.example/callback", "https://user:pass@evil.example/", "https://evil.example/#fragment", "javascript:alert(1)", "http://127.0.0.2/callback", "http://localhost.evil.example/callback"} {
		if _, err := s.Register(ClientMetadata{RedirectURIs: []string{redirect}}); err == nil {
			t.Fatal("accepted unsafe redirect", redirect)
		}
	}
	for _, m := range []ClientMetadata{
		{RedirectURIs: client.RedirectURIs, TokenEndpointAuthMethod: "client_secret_post"},
		{RedirectURIs: client.RedirectURIs, GrantTypes: []string{"client_credentials"}},
		{RedirectURIs: client.RedirectURIs, ResponseTypes: []string{"token"}},
	} {
		if _, err := s.Register(m); err == nil {
			t.Fatal("unsupported OAuth client accepted")
		}
	}
	params := url.Values{"client_id": {client.ClientID}, "redirect_uri": {"https://client.example/callback"}, "response_type": {"code"}, "resource": {"https://subdux.example/mcp"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(oauth2.GenerateVerifier())}, "code_challenge_method": {"S256"}}
	for name, value := range map[string]string{"redirect_uri": "https://attacker.example/callback", "resource": "https://other.example/mcp", "code_challenge_method": "plain", "scope": "admin", "response_type": "token"} {
		copy := url.Values{}
		for key, values := range params {
			copy[key] = append([]string{}, values...)
		}
		copy.Set(name, value)
		if _, err := s.Begin(copy); err == nil {
			t.Fatal("accepted invalid authorization", name)
		}
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestCIMDValidationCachingAndSSRF(t *testing.T) {
	s, _, _ := setup(t)
	id := "https://client.example/oauth/client.json"
	calls := 0
	s.client = &http.Client{Transport: roundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		body, _ := json.Marshal(ClientMetadata{ClientID: id, ClientName: "CIMD agent", RedirectURIs: []string{"http://127.0.0.1/callback"}, TokenEndpointAuthMethodsSupported: []string{"private_key_jwt", "none"}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(body))), Header: make(http.Header)}, nil
	})}
	if _, err := s.clientByID(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.clientByID(id); err != nil || calls != 1 {
		t.Fatal("metadata was not cached", err)
	}
	if _, err := s.clientByID("https://another.example/client.json"); err == nil {
		t.Fatal("mismatched client_id accepted")
	}
	strict := NewService(s.db)
	for _, id := range []string{"https://127.0.0.1/client.json", "https://[::1]/client.json", "https://169.254.169.254/client.json", "http://client.example/client.json", "https://client.example"} {
		if _, err := strict.clientByID(id); err == nil {
			t.Fatal("unsafe metadata URL accepted", id)
		}
	}
}

func TestConcurrentCodeExchangeIssuesAtMostOneFamily(t *testing.T) {
	s, user, client := setup(t)
	params, _ := authorize(t, s, user, client, true)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for range 8 {
		wg.Go(func() {
			if _, err := s.Exchange(params); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		})
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("exchanges succeeded %d times", successes)
	}
	var count int64
	if err := s.db.Model(&model.MCPOAuthGrant{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("issued duplicate grants", err)
	}
}
