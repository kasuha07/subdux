package service

import (
	"github.com/kasuha07/subdux/internal/service/userstatus"
	"gorm.io/gorm"
)

var errUserNotActive = userstatus.ErrUserNotActive

func ensureUserActive(tx *gorm.DB, userID uint) error {
	return userstatus.EnsureActive(tx, userID)
}
