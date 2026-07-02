package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service"
)

func TestLogoutAllRevokesAllUserRefreshTokens(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	e := newHumanOnlyRouteTestServer(t, db)

	authService := service.NewAuthService(db)
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
		Name:  refreshTokenCookieName,
		Value: first.RefreshToken,
		Path:  authRefreshPath,
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
		if cookie.Name == refreshTokenCookieName && cookie.Path == authRefreshPath && cookie.MaxAge < 0 {
			cleared = true
			break
		}
	}
	if !cleared {
		t.Fatalf("response cookies = %#v, want cleared refresh token cookie", cookies)
	}

	if _, err := authService.RefreshSession(first.RefreshToken); !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() first error = %v, want %v", err, service.ErrInvalidRefreshToken)
	}
	if _, err := authService.RefreshSession(second.RefreshToken); !errors.Is(err, service.ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() second error = %v, want %v", err, service.ErrInvalidRefreshToken)
	}
}
