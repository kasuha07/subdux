package service

import (
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

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
