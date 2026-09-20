package auth

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginLegacyCrossFieldCollision(t *testing.T) {
	t.Setenv("JWT_SECRET", "identity-test-secret-at-least-32-characters")
	db := newTestDB(t)
	hash := func(password string) string {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		return string(h)
	}
	older := model.User{Username: "victim@example.com", Email: "older@example.com", Password: hash("older-password"), Role: "user", Status: "active"}
	victim := model.User{Username: "victim", Email: "victim@example.com", Password: hash("victim-password"), Role: "user", Status: "active"}
	for _, u := range []*model.User{&older, &victim} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(db)
	for _, tc := range []struct {
		identifier, password string
		id                   uint
	}{
		{"VICTIM@example.com", "victim-password", victim.ID},
		{"victim", "victim-password", victim.ID},
		{"older@example.com", "older-password", older.ID},
	} {
		result, err := svc.Login(LoginInput{Identifier: tc.identifier, Password: tc.password})
		if err != nil {
			t.Fatalf("%s: %v", tc.identifier, err)
		}
		if result.User == nil || result.User.ID != tc.id {
			t.Fatalf("%s: wrong account", tc.identifier)
		}
	}
	if _, err := svc.Login(LoginInput{Identifier: "victim@example.com", Password: "older-password"}); err == nil {
		t.Fatal("must not fall back to conflicting username after email password failure")
	}
}

func TestRegistrationRejectsCrossFieldCollision(t *testing.T) {
	db := newTestDB(t)
	seedSystemSetting(t, db, "registration_enabled", "true")
	existing := model.User{Username: "reserved@example.com", Email: "owner@example.com", Password: "unused", Status: "active"}
	if err := db.Create(&existing).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	for _, tc := range []struct {
		username, email string
		want            error
	}{
		{"OWNER@example.com", "new@example.com", ErrUsernameAlreadyTaken},
		{"new-user", "RESERVED@example.com", ErrEmailAlreadyRegistered},
	} {
		_, err := svc.Register(RegisterInput{Username: tc.username, Email: tc.email, Password: "valid-password"})
		if !errors.Is(err, tc.want) {
			t.Fatalf("%s / %s: got %v, want %v", tc.username, tc.email, err, tc.want)
		}
	}
	exists, err := svc.emailExists("reserved@example.com", existing.ID+1)
	if err != nil || !exists {
		t.Fatalf("email-change collision check: exists=%v err=%v", exists, err)
	}
}

func TestOIDCRegistrationRejectsUsernameEmailCollision(t *testing.T) {
	db := newTestDB(t)
	user := model.User{Username: "reserved@example.com", Email: "owner@example.com", Password: "unused"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	if _, err := svc.createOIDCUser(&oidcIdentityClaims{Email: "RESERVED@example.com", Subject: "new-subject"}); err == nil {
		t.Fatal("OIDC registration accepted another user's username as email")
	}
}
