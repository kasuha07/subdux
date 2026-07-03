package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
)

// Registering a passkey is a sensitive account change, so
// AuthHandler.FinishPasskeyRegistration calls ReauthService.Consume with the
// add_passkey operation before it parses or persists the credential. These
// tests drive the real router end-to-end (the same way reauth_gate_test.go does
// for backup/restore) to prove the gate refuses a request with no ticket, a
// wrong-operation ticket, or an already-spent ticket, and only lets a request
// through once a valid add_passkey ticket is presented.
//
// A request that clears the gate then fails downstream on the placeholder
// credential ("invalid credential payload") rather than on re-auth — that
// distinguishes "gate passed" from "gate refused" without standing up a full
// WebAuthn ceremony.

func postFinishPasskeyRegistration(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(
		`{"session_id":"nonexistent-session","credential":{"stub":true},"reauth_ticket":%q}`,
		ticket,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/passkeys/register/finish", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func deletePasskey(t *testing.T, e *echo.Echo, token string, passkeyID uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/auth/passkeys/%d", passkeyID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestFinishPasskeyRegistrationGateRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postFinishPasskeyRegistration(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, "backup")
		rec := postFinishPasskeyRegistration(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket clears the gate and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, "add_passkey")

		// A valid ticket passes the gate; the request then fails downstream on the
		// stub credential rather than on re-auth, proving the gate let it through.
		rec := postFinishPasskeyRegistration(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, ticket should have cleared the gate", rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "invalid credential payload") {
			t.Fatalf("body = %s, want invalid credential payload", rec.Body.String())
		}

		// The gate consumed the ticket even though the request failed after it, so
		// a retry with the same ticket is refused at the gate.
		rec = postFinishPasskeyRegistration(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestDeletePasskeyGateRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	const secret = "JBSWY3DPEHPK3PXP"
	enableReauthGateTestTOTP(t, db, admin.ID, secret)
	currentCode := func() string {
		t.Helper()
		code, err := totp.GenerateCode(secret, time.Now().UTC())
		if err != nil {
			t.Fatalf("GenerateCode() error = %v", err)
		}
		return code
	}

	passkey := model.PasskeyCredential{
		UserID:       admin.ID,
		Name:         "delete target",
		CredentialID: "delete-passkey-target",
		Credential:   []byte("{}"),
	}
	if err := db.Create(&passkey).Error; err != nil {
		t.Fatalf("failed to create passkey: %v", err)
	}
	countPasskey := func() int64 {
		t.Helper()
		var count int64
		if err := db.Model(&model.PasskeyCredential{}).Where("id = ?", passkey.ID).Count(&count).Error; err != nil {
			t.Fatalf("count passkey error = %v", err)
		}
		return count
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := deletePasskey(t, e, token, passkey.ID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		if count := countPasskey(); count != 1 {
			t.Fatalf("passkey count = %d, want 1", count)
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicketWithCode(t, e, token, service.ReauthOperationBackup, currentCode())
		rec := deletePasskey(t, e, token, passkey.ID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		if count := countPasskey(); count != 1 {
			t.Fatalf("passkey count = %d, want 1", count)
		}
	})

	t.Run("valid ticket deletes passkey and is single-use", func(t *testing.T) {
		ticket := mintReauthTicketWithCode(t, e, token, service.ReauthOperationDeletePasskey, currentCode())
		rec := deletePasskey(t, e, token, passkey.ID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if count := countPasskey(); count != 0 {
			t.Fatalf("passkey count = %d, want 0", count)
		}

		rec = deletePasskey(t, e, token, passkey.ID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}
