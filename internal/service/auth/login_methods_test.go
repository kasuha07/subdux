package auth

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func createLoginMethodTestUser(t *testing.T, db *gorm.DB, name string, generated bool) model.User {
	t.Helper()
	user := model.User{Username: name, Email: name + "@example.com", Password: "hash", Role: "user", Status: "active", PasswordGenerated: generated}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	return user
}

func createLoginMethodTestOIDC(t *testing.T, db *gorm.DB, user model.User) model.OIDCConnection {
	t.Helper()
	connection := model.OIDCConnection{UserID: user.ID, Provider: "oidc", Issuer: "https://idp.example.com", Subject: "sub-" + user.Username, Email: user.Email}
	if err := db.Create(&connection).Error; err != nil {
		t.Fatalf("create oidc connection: %v", err)
	}
	return connection
}

func createLoginMethodTestPasskey(t *testing.T, db *gorm.DB, user model.User, credentialID string) model.PasskeyCredential {
	t.Helper()
	passkey := model.PasskeyCredential{UserID: user.ID, Name: "key", CredentialID: credentialID, Credential: []byte("{}")}
	if err := db.Create(&passkey).Error; err != nil {
		t.Fatalf("create passkey: %v", err)
	}
	return passkey
}

func TestDeleteOIDCConnectionRefusesToRemoveLastLoginMethod(t *testing.T) {
	db := newTestDB(t)
	service := NewService(db)

	t.Run("oidc-only account with generated password", func(t *testing.T) {
		user := createLoginMethodTestUser(t, db, "oidc-only", true)
		connection := createLoginMethodTestOIDC(t, db, user)
		if err := service.DeleteOIDCConnection(user.ID, connection.ID); !errors.Is(err, ErrCannotRemoveLastLoginMethod) {
			t.Fatalf("err = %v, want ErrCannotRemoveLastLoginMethod", err)
		}
		var remaining int64
		db.Model(&model.OIDCConnection{}).Where("id = ?", connection.ID).Count(&remaining)
		if remaining != 1 {
			t.Fatal("refused delete removed the connection")
		}
	})

	t.Run("generated password but a passkey remains", func(t *testing.T) {
		user := createLoginMethodTestUser(t, db, "oidc-passkey", true)
		connection := createLoginMethodTestOIDC(t, db, user)
		createLoginMethodTestPasskey(t, db, user, "cred-oidc-passkey")
		if err := service.DeleteOIDCConnection(user.ID, connection.ID); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("user-known password", func(t *testing.T) {
		user := createLoginMethodTestUser(t, db, "has-password", false)
		connection := createLoginMethodTestOIDC(t, db, user)
		if err := service.DeleteOIDCConnection(user.ID, connection.ID); err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
	})

	t.Run("another user's connection is not found", func(t *testing.T) {
		owner := createLoginMethodTestUser(t, db, "owner", false)
		other := createLoginMethodTestUser(t, db, "other", false)
		connection := createLoginMethodTestOIDC(t, db, owner)
		err := service.DeleteOIDCConnection(other.ID, connection.ID)
		if err == nil || errors.Is(err, ErrCannotRemoveLastLoginMethod) {
			t.Fatalf("err = %v, want oidc_connection_not_found", err)
		}
	})
}

func TestDeletePasskeyRefusesToRemoveLastLoginMethod(t *testing.T) {
	db := newTestDB(t)
	service := NewService(db)

	user := createLoginMethodTestUser(t, db, "passkey-only", true)
	first := createLoginMethodTestPasskey(t, db, user, "cred-1")
	second := createLoginMethodTestPasskey(t, db, user, "cred-2")

	if err := service.DeletePasskey(user.ID, first.ID); err != nil {
		t.Fatalf("deleting a non-last passkey: %v", err)
	}
	if err := service.DeletePasskey(user.ID, second.ID); !errors.Is(err, ErrCannotRemoveLastLoginMethod) {
		t.Fatalf("err = %v, want ErrCannotRemoveLastLoginMethod", err)
	}
}

func TestChangePasswordClearsGeneratedFlag(t *testing.T) {
	db := newTestDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "change-me", Email: "change-me@example.com", Password: string(hash), Role: "user", Status: "active", PasswordGenerated: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	connection := createLoginMethodTestOIDC(t, db, user)
	service := NewService(db)

	if err := service.ChangePassword(user.ID, ChangePasswordInput{CurrentPassword: "old-password", NewPassword: "new-password-123"}); err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	var reloaded model.User
	if err := db.Select("id", "password_generated").First(&reloaded, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reloaded.PasswordGenerated {
		t.Fatal("password_generated still set after the user chose a password")
	}
	if err := service.DeleteOIDCConnection(user.ID, connection.ID); err != nil {
		t.Fatalf("after the user sets a password the connection may be removed: %v", err)
	}
}
