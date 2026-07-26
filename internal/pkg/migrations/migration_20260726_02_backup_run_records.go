package migrations

import (
	"time"

	"gorm.io/gorm"
)

// backupRunMigration20260726 is the fixed backup_runs schema after this
// migration adds the archive size and encryption columns that let the run
// history double as the admin "recent backups" listing. The columns introduced
// by 20260717 remain listed so GORM adds only what is missing.
type backupRunMigration20260726 struct {
	ID                      uint      `gorm:"primaryKey"`
	Source                  string    `gorm:"size:20;not null;index"`
	ArchiveName             string    `gorm:"size:255;not null;uniqueIndex"`
	ArchivePath             string    `gorm:"type:text"`
	SizeBytes               int64     `gorm:"not null;default:0"`
	Encrypted               bool      `gorm:"not null;default:false"`
	Status                  string    `gorm:"size:20;not null;index"`
	DeliveryStatus          string    `gorm:"size:20;not null"`
	RetentionStatus         string    `gorm:"size:20;not null"`
	BookkeepingStatus       string    `gorm:"size:20;not null"`
	GlobalBookkeepingStatus string    `gorm:"size:20;not null"`
	GlobalBookkeepingError  string    `gorm:"type:text"`
	Error                   string    `gorm:"type:text"`
	StartedAt               time.Time `gorm:"not null"`
	FinishedAt              *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func (backupRunMigration20260726) TableName() string {
	return "backup_runs"
}

// migrateBackupRunRecords turns backup_runs into the single record of every
// backup: destination fan-outs keep their existing rows, and browser downloads
// (source "download") start being recorded alongside them. Size and encryption
// are stored per run so the admin history can show them without re-reading
// archives; existing rows default to 0/false, meaning "recorded before sizes
// were tracked".
func migrateBackupRunRecords(db *gorm.DB) error {
	return db.AutoMigrate(&backupRunMigration20260726{})
}
