package migrations

import (
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func TestJevSettingsMigrationPreservesRowsAndCascades(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "jev.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserJevSettings(db); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "migration", Email: "migration@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	row := model.UserJevSetting{UserID: user.ID, Enabled: true, APIKey: "ciphertext", Revision: 3}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateUserJevSettings(db); err != nil {
		t.Fatal(err)
	}
	var got model.UserJevSetting
	if err := db.First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || got.APIKey != "ciphertext" || got.Revision != 3 {
		t.Fatal("migration changed saved settings")
	}
	if err := db.Delete(&user).Error; err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.UserJevSetting{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("user deletion left credentials behind")
	}
}
