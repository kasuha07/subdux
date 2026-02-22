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

// GetDataPath returns the root data directory from the DATA_PATH environment
// variable, falling back to "data" when unset. The database, assets, and any
// other persistent files are stored under this directory.
func GetDataPath() string {
	if p := os.Getenv("DATA_PATH"); p != "" {
		return p
	}
	return "data"
}

func InitDB() *gorm.DB {
	dbPath := filepath.Join(GetDataPath(), "subdux.db")

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
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	return db
}
