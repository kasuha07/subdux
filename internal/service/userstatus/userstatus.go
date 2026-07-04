package userstatus

import (
	"errors"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"
)

var ErrUserNotActive = errors.New("user is not active")

type State struct {
	Status string
}

func (s State) IsActive() bool {
	return s.Status == StatusActive
}

func (s State) EnsureActive() error {
	if !s.IsActive() {
		return ErrUserNotActive
	}
	return nil
}

func Load(db *gorm.DB, userID uint) (State, error) {
	var user model.User
	if err := db.Select("id", "status").First(&user, userID).Error; err != nil {
		return State{}, err
	}
	return State{Status: user.Status}, nil
}

func EnsureActive(db *gorm.DB, userID uint) error {
	state, err := Load(db, userID)
	if err != nil {
		return err
	}
	return state.EnsureActive()
}
