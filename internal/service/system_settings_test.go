package service

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
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

	svc := NewSystemSettingsService(db)
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
}

func TestSystemSettingsServiceGetSiteInfo(t *testing.T) {
	db := newSystemSettingsTestDB(t)
	svc := NewSystemSettingsService(db)

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
