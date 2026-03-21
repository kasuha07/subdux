package pkg

import (
	"fmt"
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
	dataPath := GetDataPath()
	if err := prepareDataPathRuntimeOwnership(dataPath); err != nil {
		log.Fatalf("Failed to prepare runtime ownership for data directory %q: %v", dataPath, err)
	}
	if err := ensureDataPathWritable(dataPath); err != nil {
		log.Fatalf("Failed to prepare data directory %q: %v", dataPath, err)
	}

	dbPath := filepath.Join(dataPath, "subdux.db")

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
		&model.APIKey{},
		&model.RefreshToken{},
		&model.CalendarToken{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
	if err := backfillSubscriptionLifecycleFields(db); err != nil {
		log.Fatalf("Failed to backfill subscription lifecycle fields: %v", err)
	}
	return db
}

func ensureDataPathWritable(dataPath string) error {
	info, err := os.Stat(dataPath)
	switch {
	case err == nil:
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory")
		}
	case os.IsNotExist(err):
		if err := os.MkdirAll(dataPath, 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	default:
		return fmt.Errorf("failed to inspect directory: %w", err)
	}

	probe, err := os.CreateTemp(dataPath, ".subdux-write-check-*")
	if err != nil {
		return fmt.Errorf("directory is not writable: %w", err)
	}
	probePath := probe.Name()

	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return fmt.Errorf("failed to close write probe: %w", err)
	}

	if err := os.Remove(probePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clean up write probe: %w", err)
	}

	return nil
}
