package auth

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginDoesNotEnumerateDisabledAccounts(t *testing.T) {
	db := newTestDB(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	disabled := model.User{Username: "disabled-user", Email: "disabled@example.com", Password: string(hash), Role: "user", Status: "disabled"}
	active := model.User{Username: "active-user", Email: "active@example.com", Password: string(hash), Role: "user", Status: "active"}
	for _, user := range []*model.User{&disabled, &active} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create user: %v", err)
		}
	}
	service := NewService(db)

	errorCode := func(t *testing.T, err error) string {
		t.Helper()
		var typed *serviceerr.Error
		if !errors.As(err, &typed) {
			t.Fatalf("error = %v (%T), want *serviceerr.Error", err, err)
		}
		return typed.Code
	}

	for _, tc := range []struct {
		name       string
		identifier string
		password   string
	}{
		{name: "disabled account wrong password", identifier: "disabled@example.com", password: "wrong-password"},
		{name: "disabled account by username", identifier: "disabled-user", password: "wrong-password"},
		{name: "unknown identifier", identifier: "nobody@example.com", password: "wrong-password"},
		{name: "active account wrong password", identifier: "active@example.com", password: "wrong-password"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Login(LoginInput{Identifier: tc.identifier, Password: tc.password})
			if err == nil {
				t.Fatal("login succeeded, want failure")
			}
			if code := errorCode(t, err); code != "invalid_credentials" {
				t.Fatalf("error code = %q, want invalid_credentials", code)
			}
		})
	}

	t.Run("disabled account with correct password reports disabled", func(t *testing.T) {
		_, err := service.Login(LoginInput{Identifier: "disabled@example.com", Password: "correct-password"})
		if !errors.Is(err, ErrAccountDisabled) {
			t.Fatalf("error = %v, want ErrAccountDisabled", err)
		}
	})
}
