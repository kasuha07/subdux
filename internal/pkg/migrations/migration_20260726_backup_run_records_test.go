package migrations

import (
	"testing"
	"time"
)

// TestMigrateBackupRunRecordsAddsSizeAndEncryption pins the additive shape of
// the run-records migration: existing rows written before sizes were tracked
// must survive with 0/false defaults rather than blocking the migration.
func TestMigrateBackupRunRecordsAddsSizeAndEncryption(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := migrateBackupRunState(db); err != nil {
		t.Fatalf("prepare 20260717 backup run schema: %v", err)
	}

	legacy := backupRunMigration20260717{
		Source:                  "scheduled",
		ArchiveName:             "subdux-backup-legacy.zip",
		Status:                  "success",
		DeliveryStatus:          "success",
		RetentionStatus:         "success",
		BookkeepingStatus:       "success",
		GlobalBookkeepingStatus: "success",
		StartedAt:               time.Date(2026, 7, 20, 3, 0, 0, 0, time.UTC),
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("insert legacy backup run: %v", err)
	}

	if err := migrateBackupRunRecords(db); err != nil {
		t.Fatalf("migrateBackupRunRecords() error = %v", err)
	}

	for _, column := range []string{"size_bytes", "encrypted"} {
		if !db.Migrator().HasColumn(&backupRunMigration20260726{}, column) {
			t.Fatalf("backup_runs column %s is missing after migration", column)
		}
	}

	var migrated backupRunMigration20260726
	if err := db.First(&migrated, legacy.ID).Error; err != nil {
		t.Fatalf("reload legacy backup run: %v", err)
	}
	if migrated.SizeBytes != 0 || migrated.Encrypted {
		t.Fatalf("legacy run size/encrypted = %d/%v, want 0/false defaults", migrated.SizeBytes, migrated.Encrypted)
	}
	if migrated.ArchiveName != legacy.ArchiveName || migrated.Status != "success" {
		t.Fatalf("legacy run fields changed: %+v", migrated)
	}
}
