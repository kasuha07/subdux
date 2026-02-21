package pkg

import (
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB() *gorm.DB {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/subdux.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.EmailVerificationCode{},
		&model.Subscription{},
		&model.SystemSetting{},
		&model.ExchangeRate{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.UserBackupCode{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.Category{},
		&model.PaymentMethod{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
