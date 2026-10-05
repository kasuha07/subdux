package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestLogoutRevokesRefreshToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate refresh tokens: %v", err)
	}

	user := createTestUser(t, db)
	service := NewService(db)

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
	service := NewService(db)

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

func TestLogoutAllRevokesAllRefreshTokens(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.RefreshToken{}); err != nil {
		t.Fatalf("failed to migrate refresh tokens: %v", err)
	}

	user := createTestUser(t, db)
	service := NewService(db)

	first, err := service.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() first error = %v, want nil", err)
	}
	second, err := service.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession() second error = %v, want nil", err)
	}

	if err := service.LogoutAll(user.ID); err != nil {
		t.Fatalf("LogoutAll() error = %v, want nil", err)
	}

	var stored []model.RefreshToken
	if err := db.Where("user_id = ?", user.ID).Find(&stored).Error; err != nil {
		t.Fatalf("failed to load refresh tokens: %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("refresh token count = %d, want 2", len(stored))
	}
	for _, token := range stored {
		if token.RevokedAt == nil {
			t.Fatal("LogoutAll() left an active refresh token")
		}
	}

	if _, err := service.RefreshSession(first.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() first error = %v, want %v", err, ErrInvalidRefreshToken)
	}
	if _, err := service.RefreshSession(second.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("RefreshSession() second error = %v, want %v", err, ErrInvalidRefreshToken)
	}
}

func TestChangePasswordRevokesMCPOAuthAccess(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if err := db.Model(&user).Update("password", string(hash)).Error; err != nil {
		t.Fatalf("failed to set password: %v", err)
	}
	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	future := time.Now().Add(time.Hour)
	for _, grant := range []model.MCPOAuthGrant{
		{UserID: user.ID, ClientID: "client", ClientName: "Client", Scopes: "read write", Resource: "https://subdux.example/mcp", ExpiresAt: future},
		{UserID: other.ID, ClientID: "client", ClientName: "Client", Scopes: "read", Resource: "https://subdux.example/mcp", ExpiresAt: future},
	} {
		if err := db.Create(&grant).Error; err != nil {
			t.Fatalf("failed to create grant: %v", err)
		}
	}
	pending := model.MCPOAuthRequest{ID: "pending", ClientID: "client", ClientName: "Client", RedirectURI: "http://127.0.0.1/cb",
		Scopes: "read", Resource: "https://subdux.example/mcp", CodeChallenge: strings.Repeat("a", 43), UserID: &user.ID, ExpiresAt: future}
	if err := db.Create(&pending).Error; err != nil {
		t.Fatalf("failed to create pending request: %v", err)
	}

	if err := NewService(db).ChangePassword(user.ID, ChangePasswordInput{CurrentPassword: "old-password", NewPassword: "new-password"}); err != nil {
		t.Fatalf("ChangePassword() error = %v, want nil", err)
	}

	var grants []model.MCPOAuthGrant
	if err := db.Order("user_id").Find(&grants).Error; err != nil {
		t.Fatalf("failed to load grants: %v", err)
	}
	for _, grant := range grants {
		revoked := grant.RevokedAt != nil
		if revoked != (grant.UserID == user.ID) {
			t.Fatalf("grant for user %d revoked = %v, want only the changed user's grant revoked", grant.UserID, revoked)
		}
	}
	if err := db.Take(&pending, "id = ?", "pending").Error; err != nil {
		t.Fatalf("failed to reload request: %v", err)
	}
	if pending.ExpiresAt.After(time.Now()) {
		t.Fatal("pending MCP authorization request still usable after password change")
	}
}
