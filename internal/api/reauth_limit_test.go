package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

// createReauthTestAdmin creates an active admin user for exercising the
// step-up re-auth endpoints, which sit behind the admin JWT middleware.
func createReauthTestAdmin(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	admin := model.User{
		Username: "reauth-admin",
		Email:    "reauth-admin@example.com",
		Password: "hashed-password",
		Role:     "admin",
		Status:   "active",
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	return admin
}

// TestReauthPasswordRateLimitedPerUser proves that the step-up password
// endpoint is bounded per authenticated principal, so an attacker holding an
// admin session but not the password cannot guess it without limit. Every
// attempt uses a wrong password (400), but the limiter runs first; the 7th
// attempt within the window is rejected with 429.
func TestReauthPasswordRateLimitedPerUser(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)

	token, err := pkg.GenerateAccessToken(admin.ID, admin.Username, admin.Email, admin.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	body := `{"operation":"backup","password":"wrong-password"}`
	const limit = 6

	for attempt := 1; attempt <= limit; attempt++ {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/reauth/password", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d rate limited too early: status = %d", attempt, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/reauth/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "too many attempts") {
		t.Fatalf("body = %s, want too many attempts error", rec.Body.String())
	}
}
