package service

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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

func TestUpdateSettingsEncryptsCurrencyAPIKey(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	service := NewAdminService(db)
	apiKey := "currency-api-secret"
	if err := service.UpdateSettings(UpdateSettingsInput{CurrencyAPIKey: &apiKey}); err != nil {
		t.Fatalf("UpdateSettings() failed: %v", err)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", "currencyapi_key").First(&stored).Error; err != nil {
		t.Fatalf("failed to read stored currency API key: %v", err)
	}
	if stored.Value == apiKey {
		t.Fatal("currency API key should not be stored in plaintext")
	}
	if !strings.HasPrefix(stored.Value, "enc:v1:") {
		t.Fatalf("expected encrypted currency API key prefix, got %q", stored.Value)
	}

	decrypted, err := decryptSystemSettingValueIfNeeded("currencyapi_key", stored.Value)
	if err != nil {
		t.Fatalf("decryptSystemSettingValueIfNeeded() failed: %v", err)
	}
	if decrypted != apiKey {
		t.Fatalf("decrypted currency API key = %q, want %q", decrypted, apiKey)
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

func TestSMTPSkipTLSVerifyDefaultsDisabled(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	adminSvc := NewAdminService(db)
	settings, err := adminSvc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed: %v", err)
	}
	if settings.SMTPSkipTLSVerify {
		t.Fatal("GetSettings() should default SMTPSkipTLSVerify to false")
	}

	systemSettingsSvc := NewSystemSettingsService(db)
	if err := systemSettingsSvc.SeedDefaults(); err != nil {
		t.Fatalf("SeedDefaults() failed: %v", err)
	}
	var stored model.SystemSetting
	if err := db.Where("key = ?", "smtp_skip_tls_verify").First(&stored).Error; err != nil {
		t.Fatalf("failed to read seeded smtp_skip_tls_verify: %v", err)
	}
	if stored.Value != "false" {
		t.Fatalf("seeded smtp_skip_tls_verify = %q, want %q", stored.Value, "false")
	}

	entries := []model.SystemSetting{
		{Key: "smtp_enabled", Value: "true"},
		{Key: "smtp_host", Value: "smtp.example.com"},
		{Key: "smtp_port", Value: "587"},
		{Key: "smtp_username", Value: "mailer"},
		{Key: "smtp_password", Value: "smtp-password"},
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
	if cfg.SkipTLSVerify {
		t.Fatal("loadSMTPRuntimeConfig() should default SkipTLSVerify to false")
	}
}

func TestUpdateSettingsPersistsSMTPRateLimit(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	rateLimitSeconds := int64(30)
	if err := svc.UpdateSettings(UpdateSettingsInput{SMTPRateLimitSeconds: &rateLimitSeconds}); err != nil {
		t.Fatalf("UpdateSettings() failed: %v", err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed: %v", err)
	}
	if settings.SMTPRateLimitSeconds != rateLimitSeconds {
		t.Fatalf("SMTPRateLimitSeconds = %d, want %d", settings.SMTPRateLimitSeconds, rateLimitSeconds)
	}
}

func TestUpdateSettingsRejectsInvalidSMTPRateLimit(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	rateLimitSeconds := int64(-1)
	err := svc.UpdateSettings(UpdateSettingsInput{SMTPRateLimitSeconds: &rateLimitSeconds})
	if !errors.Is(err, ErrInvalidSMTPRateLimit) {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, ErrInvalidSMTPRateLimit)
	}
}

func TestUpdateSettingsClearsSSRFIPFilterList(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)

	ipList := "10.0.0.0/8\n192.168.0.0/16"
	if err := svc.UpdateSettings(UpdateSettingsInput{SSRFIPFilterList: &ipList}); err != nil {
		t.Fatalf("UpdateSettings() failed to set list: %v", err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed: %v", err)
	}
	if settings.SSRFIPFilterList == "" {
		t.Fatal("SSRFIPFilterList should not be empty after setting a list")
	}

	empty := ""
	if err := svc.UpdateSettings(UpdateSettingsInput{SSRFIPFilterList: &empty}); err != nil {
		t.Fatalf("UpdateSettings() failed to clear list: %v", err)
	}

	settings, err = svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed after clear: %v", err)
	}
	if settings.SSRFIPFilterList != "" {
		t.Fatalf("SSRFIPFilterList = %q, want empty after clearing", settings.SSRFIPFilterList)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", ssrfIPFilterListKey).First(&stored).Error; err != nil {
		t.Fatalf("failed to read stored ssrf ip filter list: %v", err)
	}
	if stored.Value != "" {
		t.Fatalf("stored ssrf ip filter list = %q, want empty", stored.Value)
	}
}

func TestUpdateSettingsClearsSSRFDomainFilterList(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)

	domainList := "example.com\ninternal.test"
	if err := svc.UpdateSettings(UpdateSettingsInput{SSRFDomainFilterList: &domainList}); err != nil {
		t.Fatalf("UpdateSettings() failed to set list: %v", err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed: %v", err)
	}
	if settings.SSRFDomainFilterList == "" {
		t.Fatal("SSRFDomainFilterList should not be empty after setting a list")
	}

	empty := ""
	if err := svc.UpdateSettings(UpdateSettingsInput{SSRFDomainFilterList: &empty}); err != nil {
		t.Fatalf("UpdateSettings() failed to clear list: %v", err)
	}

	settings, err = svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() failed after clear: %v", err)
	}
	if settings.SSRFDomainFilterList != "" {
		t.Fatalf("SSRFDomainFilterList = %q, want empty after clearing", settings.SSRFDomainFilterList)
	}
}

func TestReserveSMTPRateLimitSlotRejectsTooFrequentAttempts(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	start := time.Date(2026, 5, 25, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(start)
	t.Cleanup(restoreClock)

	cfg := smtpRuntimeConfig{
		RateLimitSeconds: 60,
		RateLimitDB:      db,
	}
	if err := reserveSMTPRateLimitSlot(cfg); err != nil {
		t.Fatalf("first reserveSMTPRateLimitSlot() error = %v", err)
	}
	if err := reserveSMTPRateLimitSlot(cfg); !errors.Is(err, ErrSMTPRateLimited) {
		t.Fatalf("second reserveSMTPRateLimitSlot() error = %v, want %v", err, ErrSMTPRateLimited)
	}

	restoreClock()
	restoreClock = pkg.SetNowForTest(start.Add(61 * time.Second))
	t.Cleanup(restoreClock)

	if err := reserveSMTPRateLimitSlot(cfg); err != nil {
		t.Fatalf("reserveSMTPRateLimitSlot() after interval error = %v", err)
	}
}
