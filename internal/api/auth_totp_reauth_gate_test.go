package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	authreauth "github.com/kasuha07/subdux/internal/service/authreauth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
)

type totpSetupGateResponse struct {
	SessionID  string `json:"session_id"`
	OtpauthURI string `json:"otpauth_uri"`
	Secret     string `json:"secret"`
}

func postTOTPSetup(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/setup", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postTOTPConfirm(t *testing.T, e *echo.Echo, token, sessionID, code string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"session_id":%q,"code":%q}`, sessionID, code)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/confirm", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postTOTPDisable(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/disable", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeTOTPSetupResponse(t *testing.T, rec *httptest.ResponseRecorder) totpSetupGateResponse {
	t.Helper()

	var resp totpSetupGateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode setup response: %v; body = %s", err, rec.Body.String())
	}
	return resp
}

func TestTOTPDisableRequiresDisableTOTPReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, user)
	const secret = "JBSWY3DPEHPK3PXP"
	enableReauthGateTestTOTP(t, db, user.ID, secret)

	currentCode := func() string {
		t.Helper()
		code, err := totp.GenerateCode(secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("GenerateCode() error = %v", err)
		}
		return code
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postTOTPDisable(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicketWithCode(t, e, token, servicereauth.ReauthOperationBackup, currentCode())
		rec := postTOTPDisable(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}

		var stillEnabled model.User
		if err := db.Select("id", "totp_enabled").First(&stillEnabled, user.ID).Error; err != nil {
			t.Fatalf("load user after rejected disable error = %v", err)
		}
		if !stillEnabled.TotpEnabled {
			t.Fatal("wrong-operation ticket disabled TOTP")
		}
	})

	t.Run("valid ticket disables totp and is single-use", func(t *testing.T) {
		ticket := mintReauthTicketWithCode(t, e, token, servicereauth.ReauthOperationDisableTOTP, currentCode())
		rec := postTOTPDisable(t, e, token, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var disabled model.User
		if err := db.Select("id", "totp_enabled", "totp_secret").First(&disabled, user.ID).Error; err != nil {
			t.Fatalf("load user after disable error = %v", err)
		}
		if disabled.TotpEnabled || disabled.TotpSecret != nil {
			t.Fatalf("user TOTP = enabled:%t secret:%v, want disabled and nil secret", disabled.TotpEnabled, disabled.TotpSecret)
		}

		rec = postTOTPDisable(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})
}

func TestTOTPSetupRequiresEnableTOTPReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, user)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postTOTPSetup(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, "backup")
		rec := postTOTPSetup(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket returns a setup session and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, "enable_totp")

		rec := postTOTPSetup(t, e, token, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		resp := decodeTOTPSetupResponse(t, rec)
		if resp.SessionID == "" || resp.Secret == "" || resp.OtpauthURI == "" {
			t.Fatalf("setup response = %+v, want session_id/secret/otpauth_uri", resp)
		}

		rec = postTOTPSetup(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestTOTPConfirmIsBoundToCurrentSetupSession(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, user)

	first := decodeTOTPSetupResponse(t, postTOTPSetup(t, e, token, mintReauthTicket(t, e, token, "enable_totp")))
	second := decodeTOTPSetupResponse(t, postTOTPSetup(t, e, token, mintReauthTicket(t, e, token, "enable_totp")))

	firstCode, err := totp.GenerateCode(first.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode(first) error = %v, want nil", err)
	}
	rec := postTOTPConfirm(t, e, token, first.SessionID, firstCode)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("stale session status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "setup expired") {
		t.Fatalf("stale session body = %s, want setup expired", rec.Body.String())
	}

	secondCode, err := totp.GenerateCode(second.Secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("GenerateCode(second) error = %v, want nil", err)
	}
	rec = postTOTPConfirm(t, e, token, second.SessionID, secondCode)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "backup_codes") {
		t.Fatalf("body = %s, want backup_codes", rec.Body.String())
	}
}

func TestSetupTOTPInternalErrorsStayInternal(t *testing.T) {
	reauthDB := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, reauthDB)
	authSvc := serviceauth.NewService(reauthDB)
	reauthSvc := servicereauth.NewService(reauthDB, authreauth.Adapt(authSvc))

	ticket, err := reauthSvc.VerifyPassword(user.ID, servicereauth.ReauthOperationEnableTOTP, reauthGateTestPassword, "")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v, want nil", err)
	}

	brokenDB := newHumanOnlyRouteTestDB(t)
	sqlDB, err := brokenDB.DB()
	if err != nil {
		t.Fatalf("brokenDB.DB() error = %v, want nil", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close broken sql DB error = %v, want nil", err)
	}

	handler := NewAuthHandler(authSvc, serviceauth.NewTOTPService(brokenDB), reauthSvc)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/setup", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(apimw.ReauthTicketHeader, ticket)
	rec := httptest.NewRecorder()
	e := echo.New()
	e.HTTPErrorHandler = APIErrorHandler(e.HTTPErrorHandler)
	c := e.NewContext(req, rec)
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			UserID:   user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			AuthType: pkg.AuthTypeUser,
		},
	})

	// Handlers now signal failures by returning an error; the central handler
	// renders the response. Drive it here as the router would.
	if err := handler.SetupTOTP(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if strings.Contains(strings.ToLower(rec.Body.String()), "closed") {
		t.Fatalf("body = %s, should not expose underlying db error", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Fatalf("body = %s, want generic internal server error", rec.Body.String())
	}
}
