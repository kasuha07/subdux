package migrations

import (
	"time"

	"gorm.io/gorm"
)

// backupDestinationMigration20260717 is the fixed destination schema after
// this migration adds durable retention and bookkeeping status fields. The
// fields introduced by 20260713 remain listed here so GORM can add only the
// columns that are missing from an existing v1 table.
type backupDestinationMigration20260717 struct {
	ID                    uint   `gorm:"primaryKey"`
	Revision              uint64 `gorm:"not null;default:1"`
	Type                  string `gorm:"not null;size:20"`
	Enabled               bool   `gorm:"default:false"`
	Config                string `gorm:"type:text"`
	LastRunAt             *time.Time
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

func (backupDestinationMigration20260717) TableName() string {
	return "backup_destinations"
}

type backupRunMigration20260717 struct {
	ID                      uint      `gorm:"primaryKey"`
	Source                  string    `gorm:"size:20;not null;index"`
	ArchiveName             string    `gorm:"size:255;not null;uniqueIndex"`
	ArchivePath             string    `gorm:"type:text"`
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

func (backupRunMigration20260717) TableName() string {
	return "backup_runs"
}

type backupRunDestinationMigration20260717 struct {
	ID                  uint   `gorm:"primaryKey"`
	RunID               uint   `gorm:"not null;uniqueIndex:idx_backup_run_destinations"`
	DestinationID       uint   `gorm:"not null;uniqueIndex:idx_backup_run_destinations;index"`
	DestinationRevision uint64 `gorm:"not null"`
	Type                string `gorm:"size:20;not null"`
	DeliveryAttempted   bool   `gorm:"not null;default:false"`
	DeliveryStatus      string `gorm:"size:20;not null"`
	DeliveryError       string `gorm:"type:text"`
	RetentionStatus     string `gorm:"size:20;not null"`
	RetentionError      string `gorm:"type:text"`
	BookkeepingStatus   string `gorm:"size:20;not null"`
	BookkeepingError    string `gorm:"type:text"`
	DeliveredAt         *time.Time
	RetentionFinishedAt *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (backupRunDestinationMigration20260717) TableName() string {
	return "backup_run_destinations"
}

// migrateBackupRunState makes delivery, retention, and bookkeeping outcomes
// durable. The destination summary columns are kept for the existing admin
// view, while the run tables retain enough state to resume an incomplete
// scheduled fan-out using the same archive name.
func migrateBackupRunState(db *gorm.DB) error {
	return db.AutoMigrate(
		&backupDestinationMigration20260717{},
		&backupRunMigration20260717{},
		&backupRunDestinationMigration20260717{},
	)
}
