package service

import (
	"errors"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

var errUserNotActive = errors.New("user is not active")

func ensureUserActive(tx *gorm.DB, userID uint) error {
	var user model.User
	if err := tx.Select("id", "status").First(&user, userID).Error; err != nil {
		return err
	}
	if user.Status != "active" {
		return errUserNotActive
	}
	return nil
}
