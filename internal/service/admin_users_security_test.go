package service

import (
	"strings"
	"testing"
)

func TestAdminCreateUserRejectsPasswordUnder8Characters(t *testing.T) {
	svc := NewAdminService(newTestDB(t))

	_, err := svc.CreateUser(CreateUserInput{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "short7!",
	})
	if err == nil {
		t.Fatal("CreateUser() error = nil, want validation error")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "at least 8 characters") {
		t.Fatalf("CreateUser() error = %q, want 8-character validation error", err.Error())
	}
}
