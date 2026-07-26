package migrations

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestMigrateBackupDestinationsBackfillsLocal verifies the upgrade path: an
// installation that already had a configured local directory and retention
// count gets exactly one enabled local destination carrying those values.
func TestMigrateBackupDestinationsBackfillsLocal(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260713{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}

	seed := []systemSettingMigration20260713{
		{Key: "backup_local_dir", Value: "/data/mybackups"},
		{Key: "backup_retention_count", Value: "14"},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("migrateBackupDestinations() error = %v", err)
	}

	var destinations []backupDestinationMigration20260713
	if err := db.Find(&destinations).Error; err != nil {
		t.Fatalf("load destinations: %v", err)
	}
	if len(destinations) != 1 {
		t.Fatalf("destinations = %d, want 1", len(destinations))
	}
	got := destinations[0]
	if got.Type != "local" || !got.Enabled {
		t.Fatalf("destination = {type:%q enabled:%v}, want local/enabled", got.Type, got.Enabled)
	}
	if got.Revision != 1 {
		t.Fatalf("destination revision = %d, want 1", got.Revision)
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(got.Config), &config); err != nil {
		t.Fatalf("unmarshal config %q: %v", got.Config, err)
	}
	if config["dir"] != "/data/mybackups" {
		t.Fatalf("config dir = %v, want /data/mybackups", config["dir"])
	}
	if config["retention_count"] != float64(14) {
		t.Fatalf("config retention_count = %v, want 14", config["retention_count"])
	}
}

// TestMigrateBackupDestinationsDefaultsWhenUnset verifies a fresh install with
// no prior backup settings still gets a usable default local destination.
func TestMigrateBackupDestinationsDefaultsWhenUnset(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260713{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}

	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("migrateBackupDestinations() error = %v", err)
	}

	var destination backupDestinationMigration20260713
	if err := db.First(&destination).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if destination.Revision != 1 {
		t.Fatalf("destination revision = %d, want 1", destination.Revision)
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(destination.Config), &config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if config["dir"] != "" {
		t.Fatalf("config dir = %v, want empty (default)", config["dir"])
	}
	if config["retention_count"] != float64(7) {
		t.Fatalf("config retention_count = %v, want 7", config["retention_count"])
	}
}

func TestMigrateBackupDestinationsDefaultsWhenRetentionExceedsMaximum(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260713{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}
	if err := db.Create(&systemSettingMigration20260713{Key: "backup_retention_count", Value: "1001"}).Error; err != nil {
		t.Fatalf("seed retention setting: %v", err)
	}

	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("migrateBackupDestinations() error = %v", err)
	}

	var destination backupDestinationMigration20260713
	if err := db.First(&destination).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	var config map[string]any
	if err := json.Unmarshal([]byte(destination.Config), &config); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if config["retention_count"] != float64(7) {
		t.Fatalf("config retention_count = %v, want default 7", config["retention_count"])
	}
}

// TestMigrateBackupDestinationsIsIdempotent verifies a second run does not
// duplicate the local destination.
func TestMigrateBackupDestinationsIsIdempotent(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260713{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}

	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("first migrateBackupDestinations() error = %v", err)
	}
	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("second migrateBackupDestinations() error = %v", err)
	}

	var count int64
	if err := db.Model(&backupDestinationMigration20260713{}).Count(&count).Error; err != nil {
		t.Fatalf("count destinations: %v", err)
	}
	if count != 1 {
		t.Fatalf("destinations after two runs = %d, want 1", count)
	}
}

func TestMigrateBackupDestinationsCreatesHistoricalSchema(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260713{}); err != nil {
		t.Fatalf("migrate system settings: %v", err)
	}

	if err := migrateBackupDestinations(db); err != nil {
		t.Fatalf("migrateBackupDestinations() error = %v", err)
	}

	var columns []string
	if err := db.Raw("SELECT name FROM pragma_table_info(?) ORDER BY cid", "backup_destinations").Pluck("name", &columns).Error; err != nil {
		t.Fatalf("inspect backup_destinations schema: %v", err)
	}
	want := []string{
		"id",
		"revision",
		"type",
		"enabled",
		"config",
		"last_run_at",
		"last_status",
		"last_error",
		"sort_order",
		"created_at",
		"updated_at",
	}
	if !reflect.DeepEqual(columns, want) {
		t.Fatalf("backup_destinations columns = %v, want %v", columns, want)
	}
}
