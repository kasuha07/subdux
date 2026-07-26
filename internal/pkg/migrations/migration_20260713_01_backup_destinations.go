package migrations

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// backupDestinationMigration20260713 is the schema contract introduced by
// this migration. Keep it local to the migration instead of using the live
// business model: later model fields belong to a later migration.
type backupDestinationMigration20260713 struct {
	ID         uint   `gorm:"primaryKey"`
	Revision   uint64 `gorm:"not null;default:1"`
	Type       string `gorm:"not null;size:20"`
	Enabled    bool   `gorm:"default:false"`
	Config     string `gorm:"type:text"`
	LastRunAt  *time.Time
	LastStatus string `gorm:"size:20;default:''"`
	LastError  string `gorm:"type:text"`
	SortOrder  int    `gorm:"default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (backupDestinationMigration20260713) TableName() string {
	return "backup_destinations"
}

// systemSettingMigration20260713 reads only the stable columns needed for the
// backfill. It deliberately avoids selecting through the mutable application
// settings model as well.
type systemSettingMigration20260713 struct {
	Key   string `gorm:"primaryKey;size:100"`
	Value string `gorm:"size:500"`
}

func (systemSettingMigration20260713) TableName() string {
	return "system_settings"
}

// migrateBackupDestinations introduces the system-scoped backup_destinations
// table and backfills a single "local" destination from the pre-existing
// global backup settings (backup_local_dir + backup_retention_count) so that
// upgrading installations keep delivering the scheduled backup to the same
// directory with no admin action. The legacy settings keys are left in place
// (read-only) to avoid disturbing the existing admin settings read paths; a
// later release may retire them.
//
// The backfill is guarded on an empty table so re-running the migration (or
// running it on a fresh install that already seeded destinations) never
// duplicates the local destination.
func migrateBackupDestinations(db *gorm.DB) error {
	if err := db.AutoMigrate(&backupDestinationMigration20260713{}); err != nil {
		return err
	}
	// Older rows created during the destination rollout may not have an
	// explicit revision when this migration is rerun against an intermediate
	// schema. Normalize them before the revision becomes a ticket binding.
	if err := db.Model(&backupDestinationMigration20260713{}).
		Where("revision = 0").
		Update("revision", 1).Error; err != nil {
		return err
	}

	var existing int64
	if err := db.Model(&backupDestinationMigration20260713{}).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	localDir, err := backupSettingValue(db, "backup_local_dir")
	if err != nil {
		return err
	}
	retentionCount := 7
	if raw, err := backupSettingValue(db, "backup_retention_count"); err != nil {
		return err
	} else if parsed, convErr := strconv.Atoi(strings.TrimSpace(raw)); convErr == nil && parsed >= 1 && parsed <= 1000 {
		retentionCount = parsed
	}

	config := map[string]any{
		"dir":             strings.TrimSpace(localDir),
		"retention_count": retentionCount,
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		return err
	}

	// The local destination config carries no secrets (just a directory and a
	// retention count), so it is stored as plaintext JSON. The config decryption
	// path is a no-op on values without the encryption envelope prefix, so the
	// service layer reads this back unchanged. The migration-local schema types
	// keep this historical step independent of application model evolution.
	destination := backupDestinationMigration20260713{
		Revision:  1,
		Type:      "local",
		Enabled:   true,
		Config:    string(encoded),
		SortOrder: 0,
	}
	return db.Create(&destination).Error
}

// backupSettingValue reads a raw system_settings value without decryption. The
// backup_local_dir and backup_retention_count keys are non-secret and stored in
// plaintext, so a raw read is sufficient and keeps this migration independent of
// the settings service.
func backupSettingValue(db *gorm.DB, key string) (string, error) {
	var setting systemSettingMigration20260713
	err := db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", nil
		}
		return "", err
	}
	return setting.Value, nil
}
