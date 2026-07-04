package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

func newSystemSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-system-settings-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings: %v", err)
	}
	return db
}

func TestSystemSettingsServiceSeedDefaultsIsIdempotent(t *testing.T) {
	db := newSystemSettingsTestDB(t)
	if err := db.Create(&model.SystemSetting{Key: "site_name", Value: "Custom"}).Error; err != nil {
		t.Fatalf("failed to seed custom site_name: %v", err)
	}

	svc := systemsettings.NewService(db)
	if err := svc.SeedDefaults(); err != nil {
		t.Fatalf("SeedDefaults() error = %v", err)
	}
	if err := svc.SeedDefaults(); err != nil {
		t.Fatalf("SeedDefaults() second call error = %v", err)
	}

	var siteName model.SystemSetting
	if err := db.Where("key = ?", "site_name").First(&siteName).Error; err != nil {
		t.Fatalf("failed to load site_name: %v", err)
	}
	if siteName.Value != "Custom" {
		t.Fatalf("site_name = %q, want Custom", siteName.Value)
	}

	var mcpEnabled model.SystemSetting
	if err := db.Where("key = ?", "mcp_enabled").First(&mcpEnabled).Error; err != nil {
		t.Fatalf("failed to load mcp_enabled: %v", err)
	}
	if mcpEnabled.Value != "false" {
		t.Fatalf("mcp_enabled = %q, want false", mcpEnabled.Value)
	}

	var ssrfProtectionEnabled model.SystemSetting
	if err := db.Where("key = ?", serviceoutbound.ProtectionEnabledKey).First(&ssrfProtectionEnabled).Error; err != nil {
		t.Fatalf("failed to load ssrf_protection_enabled: %v", err)
	}
	if ssrfProtectionEnabled.Value != "true" {
		t.Fatalf("ssrf_protection_enabled = %q, want true", ssrfProtectionEnabled.Value)
	}

	var ssrfFilterResolvedIPs model.SystemSetting
	if err := db.Where("key = ?", serviceoutbound.FilterResolvedIPsKey).First(&ssrfFilterResolvedIPs).Error; err != nil {
		t.Fatalf("failed to load ssrf_filter_resolved_ips: %v", err)
	}
	if ssrfFilterResolvedIPs.Value != "true" {
		t.Fatalf("ssrf_filter_resolved_ips = %q, want true", ssrfFilterResolvedIPs.Value)
	}
}

func TestSystemSettingsServiceGetSiteInfo(t *testing.T) {
	db := newSystemSettingsTestDB(t)
	svc := systemsettings.NewService(db)

	siteInfo, err := svc.GetSiteInfo()
	if err != nil {
		t.Fatalf("GetSiteInfo() error = %v", err)
	}
	if siteInfo.SiteName != "Subdux" {
		t.Fatalf("SiteName = %q, want Subdux", siteInfo.SiteName)
	}
	if siteInfo.MCPEnabled {
		t.Fatal("MCPEnabled = true, want false")
	}

	if err := db.Where("key = ?", "site_name").
		Assign(model.SystemSetting{Value: "Team Subdux"}).
		FirstOrCreate(&model.SystemSetting{Key: "site_name"}).Error; err != nil {
		t.Fatalf("failed to save site_name: %v", err)
	}
	if err := db.Where("key = ?", "mcp_enabled").
		Assign(model.SystemSetting{Value: "true"}).
		FirstOrCreate(&model.SystemSetting{Key: "mcp_enabled"}).Error; err != nil {
		t.Fatalf("failed to save mcp_enabled: %v", err)
	}

	siteInfo, err = svc.GetSiteInfo()
	if err != nil {
		t.Fatalf("GetSiteInfo() error = %v", err)
	}
	if siteInfo.SiteName != "Team Subdux" {
		t.Fatalf("SiteName = %q, want Team Subdux", siteInfo.SiteName)
	}
	if !siteInfo.MCPEnabled {
		t.Fatal("MCPEnabled = false, want true")
	}
}

func TestSystemSettingRuntimeHelpers(t *testing.T) {
	db := newSystemSettingsTestDB(t)
	ctx := context.Background()

	if err := db.Create(&model.SystemSetting{Key: "mcp_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed mcp_enabled: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "max_icon_file_size", Value: "8192"}).Error; err != nil {
		t.Fatalf("failed to seed max_icon_file_size: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "smtp_password", Value: "legacy-password"}).Error; err != nil {
		t.Fatalf("failed to seed smtp_password: %v", err)
	}

	missing, err := systemsettings.GetString(ctx, db, "missing_setting", "fallback")
	if err != nil {
		t.Fatalf("GetString() missing error = %v", err)
	}
	if missing != "fallback" {
		t.Fatalf("missing setting = %q, want fallback", missing)
	}

	enabled, err := systemsettings.GetBool(ctx, db, "mcp_enabled", false)
	if err != nil {
		t.Fatalf("GetBool() error = %v", err)
	}
	if !enabled {
		t.Fatal("mcp_enabled = false, want true")
	}

	maxIconFileSize, err := systemsettings.GetInt(ctx, db, "max_icon_file_size", 0)
	if err != nil {
		t.Fatalf("GetInt() error = %v", err)
	}
	if maxIconFileSize != 8192 {
		t.Fatalf("max_icon_file_size = %d, want 8192", maxIconFileSize)
	}

	password, err := systemsettings.GetString(ctx, db, "smtp_password", "")
	if err != nil {
		t.Fatalf("GetString() smtp_password error = %v", err)
	}
	if password != "legacy-password" {
		t.Fatalf("smtp_password runtime value = %q, want legacy-password", password)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", "smtp_password").First(&stored).Error; err != nil {
		t.Fatalf("failed to load stored smtp_password: %v", err)
	}
	if !strings.HasPrefix(stored.Value, "enc:v1:") {
		t.Fatalf("stored smtp_password = %q, want encrypted prefix", stored.Value)
	}
}
