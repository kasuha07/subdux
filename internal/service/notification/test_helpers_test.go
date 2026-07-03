package notification

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
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
		&model.NotificationPolicy{},
		&model.NotificationChannel{},
		&model.NotificationTemplate{},
		&model.NotificationOutbox{},
		&model.NotificationLog{},
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

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return parsed
}

func setNotificationLifecycleTestNow(t *testing.T) time.Time {
	t.Helper()
	now := mustDate(t, "2026-03-15")
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	return normalizeDateUTC(now.In(pkg.GetSystemTimezone()))
}

func seedProxySettings(t *testing.T, db *gorm.DB, enabled string, proxyType string, proxyURL string) {
	t.Helper()

	encryptedProxyURL, err := systemsettings.EncryptValueIfNeeded("system_proxy_url", proxyURL)
	if err != nil {
		t.Fatalf("failed to encrypt proxy url: %v", err)
	}

	entries := []model.SystemSetting{
		{Key: "system_proxy_enabled", Value: enabled},
		{Key: "system_proxy_type", Value: proxyType},
		{Key: "system_proxy_url", Value: encryptedProxyURL},
	}
	for _, entry := range entries {
		if err := db.Where("key = ?", entry.Key).
			Assign(model.SystemSetting{Value: entry.Value}).
			FirstOrCreate(&model.SystemSetting{Key: entry.Key}).Error; err != nil {
			t.Fatalf("failed to seed setting %q: %v", entry.Key, err)
		}
	}
}
