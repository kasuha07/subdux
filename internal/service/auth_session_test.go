package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate refresh tokens: %v", err)
	}

	user := createTestUser(t, db)
	service := NewAuthService(db)

	authResp, err := service.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v, want nil", err)
	}

	if err := service.Logout(authResp.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}

	var stored model.RefreshToken
	if err := db.Where("user_id = ?", user.ID).First(&stored).Error; err != nil {
		t.Fatalf("failed to load refresh token: %v", err)
	}
	if stored.RevokedAt == nil {
		t.Fatal("Logout() did not revoke refresh token")
	}
	if stored.LastUsedAt == nil {
		t.Fatal("Logout() did not update last_used_at")
	}
}

func TestRefreshSessionRejectsLoggedOutToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate refresh tokens: %v", err)
	}

	user := createTestUser(t, db)
	service := NewAuthService(db)

	authResp, err := service.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() error = %v, want nil", err)
	}

	if err := service.Logout(authResp.RefreshToken); err != nil {
		t.Fatalf("Logout() error = %v, want nil", err)
	}

	if _, err := service.RefreshSession(authResp.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() error = %v, want %v", err, ErrInvalidRefreshToken)
	}
}
