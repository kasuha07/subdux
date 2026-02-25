package service

import (
	"errors"
	"testing"

	"github.com/shiroha/subdux/internal/model"
)

func TestUpdateSettingsEmailDomainWhitelistNormalization(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	input := " @Example.com.;foo.com\nexample.com;sub.example.com "
	if err := svc.UpdateSettings(UpdateSettingsInput{
		EmailDomainWhitelist: &input,
	}); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	settings, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	want := "example.com\nfoo.com\nsub.example.com"
	if settings.EmailDomainWhitelist != want {
		t.Fatalf("EmailDomainWhitelist = %q, want %q", settings.EmailDomainWhitelist, want)
	}
}

func TestUpdateSettingsEmailDomainWhitelistValidationError(t *testing.T) {
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}

	svc := NewAdminService(db)
	input := "http://example.com"
	err := svc.UpdateSettings(UpdateSettingsInput{
		EmailDomainWhitelist: &input,
	})
	if !errors.Is(err, ErrInvalidEmailDomainWhitelist) {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, ErrInvalidEmailDomainWhitelist)
	}
}
