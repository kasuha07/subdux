package service

import (
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestValidateBcryptPasswordLength(t *testing.T) {
	valid := strings.Repeat("a", bcryptMaxPasswordBytes)
	if err := validateBcryptPasswordLength(valid); err != nil {
		t.Fatalf("expected %d-byte password to pass, got %v", bcryptMaxPasswordBytes, err)
	}

	tooLong := strings.Repeat("a", bcryptMaxPasswordBytes+1)
	if err := validateBcryptPasswordLength(tooLong); err != ErrPasswordTooLong {
		t.Fatalf("expected ErrPasswordTooLong, got %v", err)
	}
}

func TestUpdateSettingsEncryptsSMTPPasswordAndDecryptsOnRead(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	service := NewAdminService(db)

	enabled := true
	host := "smtp.example.com"
	port := int64(587)
	username := "mailer"
	password := "smtp-password"
	fromEmail := "noreply@example.com"

	if err := service.UpdateSettings(UpdateSettingsInput{
		SMTPEnabled:   &enabled,
		SMTPHost:      &host,
		SMTPPort:      &port,
		SMTPUsername:  &username,
		SMTPPassword:  &password,
		SMTPFromEmail: &fromEmail,
	}); err != nil {
		t.Fatalf("UpdateSettings() failed: %v", err)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", "smtp_password").First(&stored).Error; err != nil {
		t.Fatalf("failed to read stored smtp password: %v", err)
	}
	if stored.Value == password {
		t.Fatal("smtp password should not be stored in plaintext")
	}
	if !strings.HasPrefix(stored.Value, "enc:v1:") {
		t.Fatalf("expected encrypted smtp password prefix, got %q", stored.Value)
	}

	cfg, err := loadSMTPRuntimeConfig(db)
	if err != nil {
		t.Fatalf("loadSMTPRuntimeConfig() failed: %v", err)
	}
	if cfg.Password != password {
		t.Fatalf("decrypted smtp password = %q, want %q", cfg.Password, password)
	}
}

func TestLoadSMTPRuntimeConfigSupportsLegacyPlaintextPassword(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	entries := []model.SystemSetting{
		{Key: "smtp_enabled", Value: "true"},
		{Key: "smtp_host", Value: "smtp.example.com"},
		{Key: "smtp_port", Value: "587"},
		{Key: "smtp_username", Value: "mailer"},
		{Key: "smtp_password", Value: "legacy-password"},
		{Key: "smtp_from_email", Value: "noreply@example.com"},
	}
	for _, entry := range entries {
		if err := db.Where("key = ?", entry.Key).Assign(model.SystemSetting{Value: entry.Value}).FirstOrCreate(&model.SystemSetting{Key: entry.Key}).Error; err != nil {
			t.Fatalf("failed to seed setting %q: %v", entry.Key, err)
		}
	}

	cfg, err := loadSMTPRuntimeConfig(db)
	if err != nil {
		t.Fatalf("loadSMTPRuntimeConfig() failed: %v", err)
	}
	if cfg.Password != "legacy-password" {
		t.Fatalf("smtp password = %q, want %q", cfg.Password, "legacy-password")
	}
}
