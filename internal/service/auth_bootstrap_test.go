package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestPublicRegisterDoesNotCreateInitialAdmin(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	_, err := svc.Register(RegisterInput{
		Username: "attacker",
		Email:    "attacker@example.com",
		Password: "password123",
	})
	if !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("Register() error = %v, want %v", err, ErrRegistrationDisabled)
	}

	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if userCount != 0 {
		t.Fatalf("user count = %d, want 0", userCount)
	}
}

func TestEnsureInitialAdminCreatesAdminWhenDatabaseIsEmpty(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	result, err := svc.EnsureInitialAdmin(InitialAdminInput{
		Username: "root",
		Email:    "root@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if result == nil || !result.Created {
		t.Fatal("EnsureInitialAdmin() did not report a created admin")
	}
	if result.Password != "password123" {
		t.Fatalf("password = %q, want password123", result.Password)
	}

	var user model.User
	if err := db.Where("username = ?", "root").First(&user).Error; err != nil {
		t.Fatalf("failed to load initial admin: %v", err)
	}
	if user.Role != "admin" || user.Status != "active" {
		t.Fatalf("user lifecycle = (%q, %q), want (admin, active)", user.Role, user.Status)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("password123")); err != nil {
		t.Fatalf("stored password hash does not match input password: %v", err)
	}

	var currencyCount int64
	if err := db.Model(&model.UserCurrency{}).Where("user_id = ?", user.ID).Count(&currencyCount).Error; err != nil {
		t.Fatalf("failed to count default currencies: %v", err)
	}
	if currencyCount == 0 {
		t.Fatal("initial admin defaults were not seeded")
	}
}

func TestEnsureInitialAdminSkipsWhenUserExists(t *testing.T) {
	db := newTestDB(t)
	existing := createTestUser(t, db)
	svc := NewAuthService(db)

	result, err := svc.EnsureInitialAdmin(InitialAdminInput{
		Username: "root",
		Email:    "root@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if result == nil || result.Created {
		t.Fatal("EnsureInitialAdmin() should not create an admin when a user exists")
	}

	var users []model.User
	if err := db.Order("id ASC").Find(&users).Error; err != nil {
		t.Fatalf("failed to list users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("user count = %d, want 1", len(users))
	}
	if users[0].ID != existing.ID || users[0].Role != "user" {
		t.Fatalf("existing user changed to ID=%d role=%q, want ID=%d role=user", users[0].ID, users[0].Role, existing.ID)
	}
}

func TestEnsureInitialAdminGeneratesPassword(t *testing.T) {
	db := newTestDB(t)
	svc := NewAuthService(db)

	result, err := svc.EnsureInitialAdmin(InitialAdminInput{
		Username: "admin",
		Email:    "admin@example.com",
	})
	if err != nil {
		t.Fatalf("EnsureInitialAdmin() error = %v", err)
	}
	if result == nil || !result.Created {
		t.Fatal("EnsureInitialAdmin() did not create an admin")
	}
	if result.Password == "" {
		t.Fatal("generated password is empty")
	}
	if len(result.Password) < 24 {
		t.Fatalf("generated password length = %d, want at least 24", len(result.Password))
	}
}
