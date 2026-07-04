package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestLogoutAllRevokesAllUserRefreshTokens(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	e := newHumanOnlyRouteTestServer(t, db)

	authService := serviceauth.NewService(db)
	first, err := authService.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() first error = %v, want nil", err)
	}
	second, err := authService.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() second error = %v, want nil", err)
	}

	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout-all", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.AddCookie(&http.Cookie{
		Name:  apimw.RefreshTokenCookieName,
		Value: first.RefreshToken,
		Path:  apimw.AuthRefreshPath,
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNoContent, rec.Body.String())
	}

	var stored []model.RefreshToken
	if err := db.Where("user_id = ?", user.ID).Find(&stored).Error; err != nil {
		t.Fatalf("failed to load refresh tokens: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("refresh token count = %d, want 2", len(stored))
	}
	for _, candidate := range stored {
		if candidate.RevokedAt == nil {
			t.Fatal("LogoutAll route left an active refresh token")
		}
	}

	cookies := rec.Result().Cookies()
	cleared := false
	for _, cookie := range cookies {
		if cookie.Name == apimw.RefreshTokenCookieName && cookie.Path == apimw.AuthRefreshPath && cookie.MaxAge < 0 {
			cleared = true
			break
		}
	}
	if !cleared {
		t.Fatalf("response cookies = %#v, want cleared refresh token cookie", cookies)
	}

	if _, err := authService.RefreshSession(first.RefreshToken); !errors.Is(err, serviceauth.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() first error = %v, want %v", err, serviceauth.ErrInvalidRefreshToken)
	}
	if _, err := authService.RefreshSession(second.RefreshToken); !errors.Is(err, serviceauth.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() second error = %v, want %v", err, serviceauth.ErrInvalidRefreshToken)
	}
}

func seedEmailChangeVerificationCode(t *testing.T, db *gorm.DB, userID uint, email string, code string) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash verification code: %v", err)
	}

	if err := db.Create(&model.EmailVerificationCode{
		UserID:    &userID,
		Email:     strings.ToLower(strings.TrimSpace(email)),
		Purpose:   "change_email",
		CodeHash:  string(hash),
		ExpiresAt: pkg.NowUTC().Add(5 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("failed to seed email verification code: %v", err)
	}
}

func postConfirmEmailChange(t *testing.T, e *echo.Echo, token, ticket, newEmail, verificationCode string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"new_email":%q,"verification_code":%q}`, newEmail, verificationCode)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/email/change/confirm", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postSendEmailChangeCode(t *testing.T, e *echo.Echo, token, ticket, newEmail string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"new_email":%q}`, newEmail)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/email/change/send-code", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestSendEmailChangeCodeGateRequiresChangeEmailReauthTicket(t *testing.T) {
	setup := func(t *testing.T) (*gorm.DB, *echo.Echo, model.User, string) {
		t.Helper()
		db := newHumanOnlyRouteTestDB(t)
		user := createReauthGateTestAdmin(t, db)
		e := newHumanOnlyRouteTestServer(t, db)
		token := reauthGateTestToken(t, user)
		return db, e, user, token
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		_, e, _, token := setup(t)

		rec := postSendEmailChangeCode(t, e, token, "", "updated@example.com")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		_, e, _, token := setup(t)
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)

		rec := postSendEmailChangeCode(t, e, token, backupTicket, "updated@example.com")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is consumed before send attempt", func(t *testing.T) {
		_, e, _, token := setup(t)
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationChangeEmail)

		rec := postSendEmailChangeCode(t, e, token, ticket, "updated@example.com")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), serviceauth.ErrSMTPUnavailable.Error()) {
			t.Fatalf("body = %s, want %q", rec.Body.String(), serviceauth.ErrSMTPUnavailable.Error())
		}

		rec = postSendEmailChangeCode(t, e, token, ticket, "updated-again@example.com")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestConfirmEmailChangeDoesNotRequireReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, user)
	seedEmailChangeVerificationCode(t, db, user.ID, "updated@example.com", "123456")

	rec := postConfirmEmailChange(t, e, token, "", "updated@example.com", "123456")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var updated model.User
	if err := db.First(&updated, user.ID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	if updated.Email != "updated@example.com" {
		t.Fatalf("updated email = %q, want %q", updated.Email, "updated@example.com")
	}
}
