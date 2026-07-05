package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

func TestWriteOIDCSessionResultSuccess(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/session", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := writeOIDCSessionResult(c, &serviceauth.OIDCSessionResult{
		Purpose:      "login",
		Token:        "access-token",
		RefreshToken: "refresh-token",
		User: &model.User{
			ID:          1,
			Username:    "alice",
			Email:       "alice@example.com",
			Role:        "user",
			Status:      "active",
			TotpEnabled: true,
		},
	})
	if err != nil {
		t.Fatalf("writeOIDCSessionResult() error = %v, want nil", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Header().Get(echo.HeaderSetCookie), apimw.RefreshTokenCookieName+"=refresh-token") {
		t.Fatalf("set-cookie = %q, want refresh token cookie", rec.Header().Get(echo.HeaderSetCookie))
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["access_token"] != "access-token" {
		t.Fatalf("access_token = %v, want access-token", payload["access_token"])
	}
	if _, ok := payload["error_code"]; ok {
		t.Fatalf("payload should not include error_code: %s", rec.Body.String())
	}
}

func TestWriteOIDCSessionResultErrorUsesStoredKind(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oidc/session", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := writeOIDCSessionResult(c, &serviceauth.OIDCSessionResult{
		Purpose:     "connect",
		ErrorCode:   "email_already_registered_connect_oidc_from_account_settings",
		ErrorParams: map[string]any{"provider": "OIDC"},
		ErrorKind:   serviceerr.KindConflict,
	})
	if err != nil {
		t.Fatalf("writeOIDCSessionResult() error = %v, want nil", err)
	}
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if rec.Header().Get(echo.HeaderSetCookie) != "" {
		t.Fatalf("set-cookie = %q, want empty on error", rec.Header().Get(echo.HeaderSetCookie))
	}
	if !hasErrorCode(rec.Body.String(), "email_already_registered_connect_oidc_from_account_settings") {
		t.Fatalf("body = %q, want error_code", rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	params, ok := payload["error_params"].(map[string]any)
	if !ok {
		t.Fatalf("error_params = %T, want map[string]any", payload["error_params"])
	}
	if params["provider"] != "OIDC" {
		t.Fatalf("error_params[provider] = %v, want OIDC", params["provider"])
	}
}
