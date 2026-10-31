package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
)

func passwordLoginForTOTP(t *testing.T, e *echo.Echo, email string) string {
	t.Helper()
	body := fmt.Sprintf(`{"identifier":%q,"password":%q}`, email, reauthGateTestPassword)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var resp struct {
		RequiresTotp bool   `json:"requires_totp"`
		TotpToken    string `json:"totp_token"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &resp) != nil || !resp.RequiresTotp || resp.TotpToken == "" {
		t.Fatalf("password stage status = %d body = %s, want a pending TOTP token", rec.Code, rec.Body.String())
	}
	return resp.TotpToken
}

func verifyTOTPLogin(t *testing.T, e *echo.Echo, pending, code string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"totp_token":%q,"code":%q}`, pending, code)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/totp/verify-login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestTOTPLoginCodeAndPendingTokenCannotBeReplayed(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	const secret = "JBSWY3DPEHPK3PXP"
	enableReauthGateTestTOTP(t, db, user.ID, secret)
	e := newHumanOnlyRouteTestServer(t, db)

	pending := passwordLoginForTOTP(t, e, user.Email)
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if rec := verifyTOTPLogin(t, e, pending, code); rec.Code != http.StatusOK {
		t.Fatalf("first verify status = %d body = %s", rec.Code, rec.Body.String())
	}
	rec := verifyTOTPLogin(t, e, pending, code)
	if rec.Code != http.StatusUnauthorized || !hasErrorCode(rec.Body.String(), "invalid_code") {
		t.Fatalf("replayed verify status = %d body = %s, want 401 invalid_code", rec.Code, rec.Body.String())
	}
}

func TestLogoutAllInvalidatesPendingTOTPLogin(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	const secret = "JBSWY3DPEHPK3PXP"
	enableReauthGateTestTOTP(t, db, user.ID, secret)
	e := newHumanOnlyRouteTestServer(t, db)

	pending := passwordLoginForTOTP(t, e, user.Email)
	if rec := sessionEpochRequest(t, e, http.MethodPost, "/api/auth/logout-all", reauthGateTestToken(t, user), ""); rec.Code != http.StatusNoContent {
		t.Fatalf("logout-all status = %d body = %s", rec.Code, rec.Body.String())
	}
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	rec := verifyTOTPLogin(t, e, pending, code)
	if rec.Code != http.StatusUnauthorized || !hasErrorCode(rec.Body.String(), "invalid_or_expired_session") {
		t.Fatalf("verify after logout-all status = %d body = %s, want invalid_or_expired_session", rec.Code, rec.Body.String())
	}
}
