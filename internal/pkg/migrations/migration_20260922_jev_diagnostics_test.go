package migrations

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func TestJevDiagnosticsMigrationPreservesSettingsAndAddsMetrics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "jev-diagnostics.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateUserJevSettings(db); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "diagnostics", Email: "diagnostics@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO user_jev_settings (user_id, revision, enabled, api_key) VALUES (?, 4, true, 'ciphertext')`, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateJevDiagnostics(db); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if err := db.Model(&model.UserJevSetting{}).Where("user_id = ?", user.ID).Updates(map[string]any{
		"connection_status":          "available",
		"last_checked_at":            now,
		"last_success_at":            now,
		"classification_requests":    3,
		"classification_suggestions": 2,
	}).Error; err != nil {
		t.Fatal(err)
	}
	var got model.UserJevSetting
	if err := db.First(&got, "user_id = ?", user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Revision != 4 || !got.Enabled || got.APIKey != "ciphertext" || got.ConnectionStatus != "available" ||
		got.LastCheckedAt == nil || got.LastSuccessAt == nil || got.ClassificationRequests != 3 || got.ClassificationSuggestions != 2 {
		t.Fatalf("unexpected migrated settings: %+v", got)
	}
}
