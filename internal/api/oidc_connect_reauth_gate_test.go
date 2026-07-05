package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func seedOIDCConnectTestProvider(t *testing.T, db *gorm.DB) *httptest.Server {
	t.Helper()

	var issuerURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
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
		default:
			http.NotFound(w, r)
		}
	}))
	issuerURL = server.URL

	settings := map[string]string{
		"oidc_enabled":       "true",
		"oidc_provider_name": "Test OIDC",
		"oidc_issuer_url":    issuerURL,
		"oidc_client_id":     "client-id",
		"oidc_client_secret": "client-secret",
		"oidc_redirect_url":  "http://app.example.com/api/auth/oidc/callback",
	}
	for key, value := range settings {
		if err := db.Create(&model.SystemSetting{Key: key, Value: value}).Error; err != nil {
			server.Close()
			t.Fatalf("failed to seed setting %q: %v", key, err)
		}
	}

	return server
}

func postOIDCConnectStart(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/oidc/connect/start", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func assertOIDCConnectStartAccepted(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode connect start response: %v; body = %s", err, rec.Body.String())
	}
	if resp.AuthorizationURL == "" {
		t.Fatalf("authorization_url is empty; body = %s", rec.Body.String())
	}
}

func TestOIDCConnectStartGateRequiresReauthWhenUnlinked(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	provider := seedOIDCConnectTestProvider(t, db)
	defer provider.Close()

	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postOIDCConnectStart(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused and spent", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postOIDCConnectStart(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}

		rec = postBackup(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("spent ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("spent ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationConnectOIDC)
		assertOIDCConnectStartAccepted(t, postOIDCConnectStart(t, e, token, ticket))

		rec := postOIDCConnectStart(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestOIDCConnectStartAllowsExistingConnectionWithoutReauth(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	provider := seedOIDCConnectTestProvider(t, db)
	defer provider.Close()

	admin := createReauthGateTestAdmin(t, db)
	if err := db.Create(&model.OIDCConnection{
		UserID:   admin.ID,
		Provider: "oidc",
		Subject:  "subject-1",
		Email:    admin.Email,
	}).Error; err != nil {
		t.Fatalf("failed to seed oidc connection: %v", err)
	}

	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	assertOIDCConnectStartAccepted(t, postOIDCConnectStart(t, e, token, ""))
}
