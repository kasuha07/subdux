package userstatus

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"gorm.io/gorm"
)

func TestStateEnsureActive(t *testing.T) {
	if err := (State{Status: StatusActive}).EnsureActive(); err != nil {
		t.Fatalf("EnsureActive() active error = %v", err)
	}

	if err := (State{Status: StatusDisabled}).EnsureActive(); !errors.Is(err, ErrUserNotActive) {
		t.Fatalf("EnsureActive() disabled error = %v, want %v", err, ErrUserNotActive)
	}
}

func TestEnsureActive(t *testing.T) {
	db := servicetest.NewDB(t)

	active := model.User{
		Username: "active-user",
		Email:    "active@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   StatusActive,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatalf("failed to create active user: %v", err)
	}

	disabled := model.User{
		Username: "disabled-user",
		Email:    "disabled@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   StatusDisabled,
	}
	if err := db.Create(&disabled).Error; err != nil {
		t.Fatalf("failed to create disabled user: %v", err)
	}

	if err := EnsureActive(db, active.ID); err != nil {
		t.Fatalf("EnsureActive() active error = %v", err)
	}

	if err := EnsureActive(db, disabled.ID); !errors.Is(err, ErrUserNotActive) {
		t.Fatalf("EnsureActive() disabled error = %v, want %v", err, ErrUserNotActive)
	}

	if err := EnsureActive(db, 999999); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("EnsureActive() missing error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}
