package subscription

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationOutbox{},
		&model.BackgroundTaskLease{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "tester",
		Email:    "tester@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}
