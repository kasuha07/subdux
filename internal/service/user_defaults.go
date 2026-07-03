package service

import (
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
)

func SeedUserDefaults(tx *gorm.DB, userID uint) error {
	return serviceutil.SeedUserDefaults(tx, userID)
}
