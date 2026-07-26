package migrations

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"gorm.io/gorm"
)

// seedLegacyBackupSchedule prepares the pre-refactor world: the global schedule
// settings plus destinations whose configs carry only transport fields.
func seedLegacyBackupSchedule(t *testing.T, settings map[string]string, configs ...string) *gorm.DB {
	t.Helper()

	db := openRawSQLiteTestDB(t)
	if err := db.AutoMigrate(&systemSettingMigration20260726{}, &backupDestinationMigration20260726{}); err != nil {
		t.Fatalf("migrate fixtures: %v", err)
	}

	for key, value := range settings {
		if err := db.Create(&systemSettingMigration20260726{Key: key, Value: value}).Error; err != nil {
			t.Fatalf("seed setting %q: %v", key, err)
		}
	}
	for _, config := range configs {
		destination := backupDestinationMigration20260726{
			Revision: 1,
			Type:     "local",
			Enabled:  true,
			Config:   config,
		}
		if err := db.Create(&destination).Error; err != nil {
			t.Fatalf("seed destination: %v", err)
		}
	}
	return db
}

func loadMigratedConfig(t *testing.T, db *gorm.DB, id uint) map[string]any {
	t.Helper()

	var destination backupDestinationMigration20260726
	if err := db.First(&destination, id).Error; err != nil {
		t.Fatalf("load destination %d: %v", id, err)
	}
	config := map[string]any{}
	if err := json.Unmarshal([]byte(destination.Config), &config); err != nil {
		t.Fatalf("unmarshal config %q: %v", destination.Config, err)
	}
	return config
}

// The fold must be lossless. An installation producing encrypted, assets-bearing
// archives at 04:30 must keep doing exactly that afterwards; the whole point of
// copying the password forward is that the archives do not silently become
// plaintext when the setting moves.
func TestMigrateBackupPerDestinationScheduleFoldsGlobalSettings(t *testing.T) {
	db := seedLegacyBackupSchedule(t, map[string]string{
		"backup_schedule_enabled":    "true",
		"backup_time_of_day":         "04:30",
		"backup_include_assets":      "true",
		"backup_encrypt_enabled":     "true",
		"backup_encryption_password": "s3cret",
	}, `{"dir":"/data/backups","retention_count":14}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("migrateBackupPerDestinationSchedule() error = %v", err)
	}

	config := loadMigratedConfig(t, db, 1)
	if config["time_of_day"] != "04:30" {
		t.Fatalf("config time_of_day = %v, want 04:30", config["time_of_day"])
	}
	if config["include_assets"] != true {
		t.Fatalf("config include_assets = %v, want true", config["include_assets"])
	}
	if config["encrypt_enabled"] != true {
		t.Fatalf("config encrypt_enabled = %v, want true", config["encrypt_enabled"])
	}
	if config["encryption_password"] != "s3cret" {
		t.Fatalf("config encryption_password = %v, want the folded password", config["encryption_password"])
	}
	// Transport fields must survive the rewrite untouched.
	if config["dir"] != "/data/backups" || config["retention_count"] != float64(14) {
		t.Fatalf("transport fields = {dir:%v retention_count:%v}, want preserved", config["dir"], config["retention_count"])
	}

	var destination backupDestinationMigration20260726
	if err := db.First(&destination, 1).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if !destination.Enabled {
		t.Fatal("destination enabled = false, want true when the global schedule was on")
	}
}

// The old effective state was (schedule_enabled AND destination.enabled). With
// the global switch gone, a destination row left enabled would start backing up
// on its own — so an off schedule must disable every destination.
func TestMigrateBackupPerDestinationScheduleDisablesWhenScheduleWasOff(t *testing.T) {
	db := seedLegacyBackupSchedule(t, map[string]string{
		"backup_schedule_enabled": "false",
		"backup_time_of_day":      "02:15",
	}, `{"dir":""}`, `{"dir":"/other"}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("migrateBackupPerDestinationSchedule() error = %v", err)
	}

	var destinations []backupDestinationMigration20260726
	if err := db.Order("id ASC").Find(&destinations).Error; err != nil {
		t.Fatalf("load destinations: %v", err)
	}
	if len(destinations) != 2 {
		t.Fatalf("destinations = %d, want 2", len(destinations))
	}
	for _, destination := range destinations {
		if destination.Enabled {
			t.Fatalf("destination %d enabled = true, want false when the global schedule was off", destination.ID)
		}
	}
	// The schedule itself is still folded in, so re-enabling the plan in the UI
	// restores the operator's original timing rather than a default.
	if config := loadMigratedConfig(t, db, 1); config["time_of_day"] != "02:15" {
		t.Fatalf("config time_of_day = %v, want 02:15", config["time_of_day"])
	}
}

// A fresh install has no global settings at all. Every destination must still
// come out with a complete, usable plan rather than a half-populated config.
func TestMigrateBackupPerDestinationScheduleDefaultsWhenUnset(t *testing.T) {
	db := seedLegacyBackupSchedule(t, nil, `{"dir":""}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("migrateBackupPerDestinationSchedule() error = %v", err)
	}

	config := loadMigratedConfig(t, db, 1)
	if config["time_of_day"] != "03:00" {
		t.Fatalf("config time_of_day = %v, want the 03:00 default", config["time_of_day"])
	}
	if config["include_assets"] != false || config["encrypt_enabled"] != false {
		t.Fatalf("config = %v, want assets and encryption off by default", config)
	}
	if config["encryption_password"] != "" {
		t.Fatalf("config encryption_password = %v, want empty", config["encryption_password"])
	}
}

// The global last-run timestamp becomes the per-destination due anchor.
// Without it, upgrading an installation minutes after its nightly backup would
// immediately fire a second one.
func TestMigrateBackupPerDestinationScheduleSeedsLastScheduledRun(t *testing.T) {
	lastRun := time.Date(2026, 7, 26, 3, 0, 0, 0, time.UTC)
	db := seedLegacyBackupSchedule(t, map[string]string{
		"backup_schedule_enabled": "true",
		"backup_last_run_at":      lastRun.Format(time.RFC3339),
	}, `{"dir":""}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("migrateBackupPerDestinationSchedule() error = %v", err)
	}

	var destination backupDestinationMigration20260726
	if err := db.First(&destination, 1).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if destination.LastScheduledRunAt == nil {
		t.Fatal("last_scheduled_run_at = nil, want the folded global last run")
	}
	if !destination.LastScheduledRunAt.UTC().Equal(lastRun) {
		t.Fatalf("last_scheduled_run_at = %v, want %v", destination.LastScheduledRunAt.UTC(), lastRun)
	}
}

// The retired keys must actually leave the settings table: leaving them behind
// would give the schedule two homes, which is the state this refactor removes.
func TestMigrateBackupPerDestinationScheduleDeletesLegacySettings(t *testing.T) {
	seed := make(map[string]string, len(legacyBackupScheduleKeys))
	for _, key := range legacyBackupScheduleKeys {
		seed[key] = "true"
	}
	seed["site_name"] = "keep me"

	db := seedLegacyBackupSchedule(t, seed, `{"dir":""}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("migrateBackupPerDestinationSchedule() error = %v", err)
	}

	var remaining []string
	if err := db.Model(&systemSettingMigration20260726{}).Pluck("key", &remaining).Error; err != nil {
		t.Fatalf("load remaining settings: %v", err)
	}
	for _, key := range legacyBackupScheduleKeys {
		if slices.Contains(remaining, key) {
			t.Fatalf("setting %q still present, want deleted", key)
		}
	}
	if !slices.Contains(remaining, "site_name") {
		t.Fatal("unrelated setting site_name was deleted, want preserved")
	}
}

// Re-running must not clobber a plan the admin has since edited: the fold only
// fills in schedule keys that are absent.
func TestMigrateBackupPerDestinationScheduleIsIdempotent(t *testing.T) {
	db := seedLegacyBackupSchedule(t, map[string]string{
		"backup_schedule_enabled": "true",
		"backup_time_of_day":      "04:30",
	}, `{"dir":""}`)

	activeSecretCodec = testSecretCodec()
	defer func() { activeSecretCodec = SecretCodec{} }()

	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("first migrateBackupPerDestinationSchedule() error = %v", err)
	}
	if err := db.Model(&backupDestinationMigration20260726{}).
		Where("id = ?", 1).
		Update("config", `{"dir":"","time_of_day":"23:45","include_assets":false,"encrypt_enabled":false,"encryption_password":""}`).
		Error; err != nil {
		t.Fatalf("simulate admin edit: %v", err)
	}
	if err := migrateBackupPerDestinationSchedule(db); err != nil {
		t.Fatalf("second migrateBackupPerDestinationSchedule() error = %v", err)
	}

	if config := loadMigratedConfig(t, db, 1); config["time_of_day"] != "23:45" {
		t.Fatalf("config time_of_day = %v, want the admin's 23:45 preserved", config["time_of_day"])
	}
}

// Without a codec the migration cannot rewrite the encrypted config. Failing is
// mandatory: a pass-through would write the archive password back in plaintext.
func TestMigrateBackupPerDestinationScheduleRequiresSecretCodec(t *testing.T) {
	db := seedLegacyBackupSchedule(t, map[string]string{
		"backup_schedule_enabled": "true",
	}, `{"dir":""}`)

	activeSecretCodec = SecretCodec{}

	if err := migrateBackupPerDestinationSchedule(db); err == nil {
		t.Fatal("migrateBackupPerDestinationSchedule() error = nil, want a missing-codec failure")
	}
}
