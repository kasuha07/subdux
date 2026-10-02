package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/mcpoauth"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

func oauthRequest(t *testing.T, e *echo.Echo, method, path, token, contentType string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	if path == "/mcp" && method == "POST" {
		var message map[string]interface{}
		if err := json.Unmarshal(body, &message); err != nil {
			t.Fatal(err)
		}
		params, ok := message["params"].(map[string]interface{})
		if !ok {
			params = map[string]interface{}{}
			message["params"] = params
		}
		params["_meta"] = map[string]interface{}{
			"io.modelcontextprotocol/protocolVersion":    "2026-07-28",
			"io.modelcontextprotocol/clientInfo":         map[string]string{"name": "test-client", "version": "1"},
			"io.modelcontextprotocol/clientCapabilities": map[string]interface{}{},
		}
		var err error
		body, err = json.Marshal(message)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if path == "/mcp" {
		req.Header.Set("MCP-Protocol-Version", "2026-07-28")
		var message struct {
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		_ = json.Unmarshal(body, &message)
		req.Header.Set("Mcp-Method", message.Method)
		if message.Params.Name != "" {
			req.Header.Set("Mcp-Name", message.Params.Name)
		}
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func oauthFixture(t *testing.T) (*echo.Echo, *gorm.DB, model.User, string) {
	t.Helper()
	db := newMCPRouteTestDB(t)
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.MCPOAuthClient{}, &model.MCPOAuthRequest{}, &model.MCPOAuthGrant{}, &model.MCPOAuthToken{}); err != nil {
		t.Fatal(err)
	}
	user := createMCPRouteTestUser(t, db)
	enableMCPRoute(t, db)
	if err := db.Create(&model.SystemSetting{Key: "site_url", Value: "https://subdux.example"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	e := echo.New()
	NewApp(ctx, db, serviceutil.NewBackgroundTaskMonitor()).Mount(e)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatal(err)
	}
	return e, db, user, token
}

func oauthRouteToken(t *testing.T, e *echo.Echo, human string, write bool) *mcpoauth.TokenResponse {
	t.Helper()
	rec := oauthRequest(t, e, "POST", "/oauth/register", "", "application/json", []byte(`{"client_name":"Verified test fixture","redirect_uris":["http://127.0.0.1/callback"],"token_endpoint_auth_method":"none"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status %d", rec.Code)
	}
	var client mcpoauth.ClientMetadata
	if err := json.Unmarshal(rec.Body.Bytes(), &client); err != nil {
		t.Fatal(err)
	}
	verifier := oauth2.GenerateVerifier()
	params := url.Values{"response_type": {"code"}, "client_id": {client.ClientID}, "redirect_uri": {"http://127.0.0.1:5555/callback"}, "resource": {"https://subdux.example/mcp"}, "scope": {"read write"}, "state": {"fixture-state"}, "code_challenge": {oauth2.S256ChallengeFromVerifier(verifier)}, "code_challenge_method": {"S256"}}
	rec = oauthRequest(t, e, "GET", "/oauth/authorize?"+params.Encode(), "", "", nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("authorize status %d", rec.Code)
	}
	location, _ := url.Parse(rec.Header().Get("Location"))
	requestPath := "/api/mcp/oauth/requests/" + location.Query().Get("request")
	rec = oauthRequest(t, e, "GET", requestPath, human, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("consent status %d", rec.Code)
	}
	decision, _ := json.Marshal(map[string]bool{"approve": true, "allow_write": write})
	rec = oauthRequest(t, e, "POST", requestPath, human, "application/json", decision)
	if rec.Code != http.StatusOK {
		t.Fatalf("decision status %d", rec.Code)
	}
	var response struct {
		RedirectURI string `json:"redirect_uri"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	redirect, _ := url.Parse(response.RedirectURI)
	if redirect.Query().Get("iss") != "https://subdux.example" || redirect.Query().Get("state") != "fixture-state" {
		t.Fatal("issuer/state lost")
	}
	params = url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ClientID}, "redirect_uri": {params.Get("redirect_uri")}, "resource": {params.Get("resource")}, "code": {redirect.Query().Get("code")}, "code_verifier": {verifier}}
	rec = oauthRequest(t, e, "POST", "/oauth/token", "", "application/x-www-form-urlencoded", []byte(params.Encode()))
	if rec.Code != http.StatusOK {
		t.Fatalf("token status %d", rec.Code)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("tokens could be cached")
	}
	var result mcpoauth.TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return &result
}

func TestMCPOAuthRoutesDiscoveryAccessAndRevocation(t *testing.T) {
	e, db, user, human := oauthFixture(t)
	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-authorization-server"} {
		rec := oauthRequest(t, e, "GET", path, "", "", nil)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), "https://subdux.example") {
			t.Fatal("discovery failed", path)
		}
	}
	rec := oauthRequest(t, e, "GET", "/mcp", "", "", nil)
	if rec.Code != 401 || !strings.Contains(rec.Header().Get("WWW-Authenticate"), "resource_metadata=") {
		t.Fatal("no OAuth discovery challenge")
	}
	read := oauthRouteToken(t, e, human, false)
	rec = oauthRequest(t, e, "POST", "/mcp", read.AccessToken, "application/json", []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "search_subscriptions") || !strings.Contains(rec.Body.String(), "securitySchemes") {
		t.Fatalf("OAuth MCP tool discovery failed: status %d, %s", rec.Code, rec.Body.String())
	}
	create := []byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"create_subscription","arguments":{"idempotency_key":"oauth-write","name":"OAuth subscription","amount":12,"next_billing_date":"2026-11-01"}}}`)
	rec = oauthRequest(t, e, "POST", "/mcp", read.AccessToken, "application/json", create)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "insufficient_scope") || !strings.Contains(rec.Body.String(), "mcp/www_authenticate") {
		t.Fatal("read token did not reject writes with an OAuth challenge")
	}
	for _, path := range []string{"/api/auth/me", "/api/subscriptions", "/api/api-keys", "/api/admin/users", "/api/mcp/oauth/grants"} {
		rec = oauthRequest(t, e, "GET", path, read.AccessToken, "", nil)
		if rec.Code != 401 {
			t.Fatalf("OAuth token entered REST %s: %d", path, rec.Code)
		}
	}
	write := oauthRouteToken(t, e, human, true)
	for range 2 {
		rec = oauthRequest(t, e, "POST", "/mcp", write.AccessToken, "application/json", create)
		if rec.Code != 200 || strings.Contains(rec.Body.String(), `"isError":true`) || strings.Contains(rec.Body.String(), `"error":`) {
			t.Fatal("OAuth write or idempotent replay failed", rec.Body.String())
		}
	}
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatal("write replay duplicated subscription", err)
	}
	var event model.AuditEvent
	if err := db.Where("user_id = ?", user.ID).Take(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.KeyID != 0 || event.KeyKind != "mcp_oauth" || event.OAuthGrantID == nil || event.OAuthClientID == "" {
		t.Fatal("OAuth audit attribution missing")
	}
	grantID := strconv.Itoa(int(*event.OAuthGrantID))
	rec = oauthRequest(t, e, "DELETE", "/api/mcp/oauth/grants/"+grantID, human, "", nil)
	if rec.Code != 204 {
		t.Fatal("grant revocation failed")
	}
	rec = oauthRequest(t, e, "POST", "/mcp", write.AccessToken, "application/json", []byte(`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`))
	if rec.Code != 401 {
		t.Fatal("revoked token remained valid")
	}
	// A human session is never an MCP access token.
	rec = oauthRequest(t, e, "POST", "/mcp", human, "application/json", []byte(`{"jsonrpc":"2.0","id":4,"method":"tools/list"}`))
	if rec.Code != 401 {
		t.Fatal("human session accepted at MCP")
	}
}

func TestMCPOAuthMalformedRequestsAndDisabledBoundary(t *testing.T) {
	e, db, user, _ := oauthFixture(t)
	for _, test := range []struct{ path, body, content string }{
		{"/oauth/token", `{"grant_type":"authorization_code"}`, "application/json"},
		{"/oauth/token", "grant_type=authorization_code&grant_type=refresh_token", "application/x-www-form-urlencoded"},
		{"/oauth/register", "not JSON", "application/json"},
		{"/oauth/authorize?client_id=a&client_id=b", "", ""},
	} {
		method := "POST"
		if strings.HasPrefix(test.path, "/oauth/authorize") {
			method = "GET"
		}
		rec := oauthRequest(t, e, method, test.path, "", test.content, []byte(test.body))
		if rec.Code != 400 {
			t.Fatalf("malformed request status %d", rec.Code)
		}
	}
	key := createMCPRouteAPIKey(t, db, user)
	req := httptest.NewRequest("GET", "/api/mcp/oauth/grants", nil)
	req.Header.Set("X-API-Key", key)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatal("machine key could manage OAuth grants")
	}
	if err := db.Model(&model.SystemSetting{}).Where("key = ?", "mcp_enabled").Update("value", "false").Error; err != nil {
		t.Fatal(err)
	}
	rec = oauthRequest(t, e, "GET", "/.well-known/oauth-authorization-server", "", "", nil)
	if rec.Code != 404 {
		t.Fatal("disabled MCP advertised OAuth")
	}
}
