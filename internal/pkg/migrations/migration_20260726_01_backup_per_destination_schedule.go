package migrations

import (
	"encoding/json"
	"strings"
	"time"

	"gorm.io/gorm"
)

// backupDestinationMigration20260726 is the destination schema after this
// migration adds last_scheduled_run_at. The earlier fields remain listed so
// GORM adds only the column that is missing from an existing table.
type backupDestinationMigration20260726 struct {
	ID                    uint   `gorm:"primaryKey"`
	Revision              uint64 `gorm:"not null;default:1"`
	Type                  string `gorm:"not null;size:20"`
	Enabled               bool   `gorm:"default:false"`
	Config                string `gorm:"type:text"`
	LastRunAt             *time.Time
	LastScheduledRunAt    *time.Time
	LastStatus            string `gorm:"size:20;default:''"`
	LastError             string `gorm:"type:text"`
	LastRetentionStatus   string `gorm:"size:20;default:'not_attempted'"`
	LastRetentionError    string `gorm:"type:text"`
	LastBookkeepingStatus string `gorm:"size:20;default:'pending'"`
	LastBookkeepingError  string `gorm:"type:text"`
	SortOrder             int    `gorm:"default:0"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func (backupDestinationMigration20260726) TableName() string {
	return "backup_destinations"
}

type systemSettingMigration20260726 struct {
	Key   string `gorm:"primaryKey;size:100"`
	Value string `gorm:"size:500"`
}

func (systemSettingMigration20260726) TableName() string {
	return "system_settings"
}

// legacyBackupScheduleKeys are the global settings this migration folds into
// each destination and then removes. They are listed once so the backfill and
// the delete cannot drift apart.
var legacyBackupScheduleKeys = []string{
	"backup_schedule_enabled",
	"backup_time_of_day",
	"backup_include_assets",
	"backup_encrypt_enabled",
	"backup_encryption_password",
	"backup_last_run_at",
	"backup_last_status",
	"backup_last_error",
}

// migrateBackupPerDestinationSchedule turns each backup destination into a
// self-contained backup plan. The single global schedule (enabled, time of day,
// include assets, archive encryption) is folded into every destination's config
// and then deleted, so there is exactly one place a schedule can live.
//
// The fold is deliberately faithful rather than defaulted. Copying the archive
// encryption password forward matters most: an installation that was producing
// encrypted archives must not silently begin writing plaintext ones because the
// setting moved. For the same reason the old effective state
// (schedule_enabled AND destination.enabled) is preserved by disabling every
// destination when the global schedule was off — otherwise turning the schedule
// off would be forgotten and backups would resume unrequested.
func migrateBackupPerDestinationSchedule(db *gorm.DB) error {
	if err := db.AutoMigrate(&backupDestinationMigration20260726{}); err != nil {
		return err
	}

	legacy, err := loadLegacyBackupSchedule(db)
	if err != nil {
		return err
	}

	var destinations []backupDestinationMigration20260726
	if err := db.Find(&destinations).Error; err != nil {
		return err
	}

	for i := range destinations {
		if err := applyLegacyScheduleToDestination(db, &destinations[i], legacy); err != nil {
			return err
		}
	}

	return db.Where("key IN ?", legacyBackupScheduleKeys).
		Delete(&systemSettingMigration20260726{}).Error
}

// legacyBackupSchedule is the decoded global schedule being retired.
type legacyBackupSchedule struct {
	ScheduleEnabled    bool
	TimeOfDay          string
	IncludeAssets      bool
	EncryptEnabled     bool
	EncryptionPassword string
	LastRunAt          *time.Time
}

func loadLegacyBackupSchedule(db *gorm.DB) (legacyBackupSchedule, error) {
	schedule := legacyBackupSchedule{TimeOfDay: "03:00"}

	var rows []systemSettingMigration20260726
	if err := db.Where("key IN ?", legacyBackupScheduleKeys).Find(&rows).Error; err != nil {
		return schedule, err
	}

	values := make(map[string]string, len(rows))
	for _, row := range rows {
		values[row.Key] = row.Value
	}

	schedule.ScheduleEnabled = values["backup_schedule_enabled"] == "true"
	if timeOfDay := strings.TrimSpace(values["backup_time_of_day"]); timeOfDay != "" {
		schedule.TimeOfDay = timeOfDay
	}
	schedule.IncludeAssets = values["backup_include_assets"] == "true"
	schedule.EncryptEnabled = values["backup_encrypt_enabled"] == "true"

	if stored := strings.TrimSpace(values["backup_encryption_password"]); stored != "" {
		// The password was stored under the system-setting envelope. A decrypt
		// failure must not be swallowed into an empty password: that would turn
		// an encrypted-backup installation into a plaintext one.
		decrypted, err := activeSecretCodec.decrypt(stored)
		if err != nil {
			return schedule, err
		}
		schedule.EncryptionPassword = decrypted
	}

	// Seed the per-destination due anchor from the global last run so a schedule
	// that already fired today is not immediately fired again after upgrade.
	if raw := strings.TrimSpace(values["backup_last_run_at"]); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			schedule.LastRunAt = &parsed
		}
	}

	return schedule, nil
}

// applyLegacyScheduleToDestination merges the retired global schedule into one
// destination's encrypted config. Existing transport fields are preserved
// verbatim; only the schedule keys are written, and only when the destination
// does not already carry them (so a re-run is a no-op).
func applyLegacyScheduleToDestination(
	db *gorm.DB,
	destination *backupDestinationMigration20260726,
	legacy legacyBackupSchedule,
) error {
	plain, err := activeSecretCodec.decrypt(destination.Config)
	if err != nil {
		return err
	}

	config := map[string]any{}
	if trimmed := strings.TrimSpace(plain); trimmed != "" {
		if err := json.Unmarshal([]byte(trimmed), &config); err != nil {
			return err
		}
	}

	if _, present := config["time_of_day"]; !present {
		config["time_of_day"] = legacy.TimeOfDay
	}
	if _, present := config["include_assets"]; !present {
		config["include_assets"] = legacy.IncludeAssets
	}
	if _, present := config["encrypt_enabled"]; !present {
		config["encrypt_enabled"] = legacy.EncryptEnabled
	}
	if _, present := config["encryption_password"]; !present {
		config["encryption_password"] = legacy.EncryptionPassword
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}
	encrypted, err := activeSecretCodec.encrypt(string(encoded))
	if err != nil {
		return err
	}

	updates := map[string]any{
		"config":   encrypted,
		"revision": gorm.Expr("revision + 1"),
	}
	if !legacy.ScheduleEnabled {
		updates["enabled"] = false
	}
	if destination.LastScheduledRunAt == nil && legacy.LastRunAt != nil {
		updates["last_scheduled_run_at"] = *legacy.LastRunAt
	}

	return db.Model(&backupDestinationMigration20260726{}).
		Where("id = ?", destination.ID).
		Updates(updates).Error
}
