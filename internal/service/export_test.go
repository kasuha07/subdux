package service

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

func newExportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "export-service-test-key")

	dbPath := filepath.Join(t.TempDir(), "subdux-export-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationTemplate{},
		&model.CalendarToken{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestExportUserDataRedactsNotificationSecretsByDefault(t *testing.T) {
	db := newExportTestDB(t)
	user := createTestUser(t, db)
	secretConfig := `{"url":"https://example.com/hook","secret":"webhook-secret","headers":{"X-Token":"header-secret"}}`
	encryptedConfig, err := pkg.EncryptNotificationChannelConfig(secretConfig)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}
	if err := db.Create(&model.NotificationChannel{
		UserID:  user.ID,
		Type:    "webhook",
		Enabled: true,
		Config:  encryptedConfig,
	}).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	data, err := NewExportService(db).ExportUserData(user.ID, false)
	if err != nil {
		t.Fatalf("ExportUserData() error = %v", err)
	}
	if data.SecretsIncluded {
		t.Fatal("SecretsIncluded = true, want false")
	}
	if len(data.Notifications.Channels) != 1 {
		t.Fatalf("channels length = %d, want 1", len(data.Notifications.Channels))
	}

	var exportedConfig struct {
		Secret  string            `json:"secret"`
		Headers map[string]string `json:"headers"`
		URL     string            `json:"url"`
	}
	if err := json.Unmarshal([]byte(data.Notifications.Channels[0].Config), &exportedConfig); err != nil {
		t.Fatalf("failed to parse exported config: %v", err)
	}
	if exportedConfig.Secret != "" {
		t.Fatalf("exported secret = %q, want empty string", exportedConfig.Secret)
	}
	if exportedConfig.Headers["X-Token"] != "" {
		t.Fatalf("exported header token = %q, want empty string", exportedConfig.Headers["X-Token"])
	}
	if exportedConfig.URL != "https://example.com/hook" {
		t.Fatalf("exported url = %q, want https://example.com/hook", exportedConfig.URL)
	}
}

func TestExportUserDataCanIncludeNotificationSecrets(t *testing.T) {
	db := newExportTestDB(t)
	user := createTestUser(t, db)
	secretConfig := `{"api_key":"resend-secret","from_email":"from@example.com","to_email":"to@example.com"}`
	encryptedConfig, err := pkg.EncryptNotificationChannelConfig(secretConfig)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}
	if err := db.Create(&model.NotificationChannel{
		UserID:  user.ID,
		Type:    "resend",
		Enabled: true,
		Config:  encryptedConfig,
	}).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	data, err := NewExportService(db).ExportUserData(user.ID, true)
	if err != nil {
		t.Fatalf("ExportUserData() error = %v", err)
	}
	if !data.SecretsIncluded {
		t.Fatal("SecretsIncluded = false, want true")
	}

	var exportedConfig map[string]string
	if err := json.Unmarshal([]byte(data.Notifications.Channels[0].Config), &exportedConfig); err != nil {
		t.Fatalf("failed to parse exported config: %v", err)
	}
	if exportedConfig["api_key"] != "resend-secret" {
		t.Fatalf("exported api_key = %q, want resend-secret", exportedConfig["api_key"])
	}
}
