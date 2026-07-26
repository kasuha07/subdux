package model

import "time"

// BackupDestination is a system-scoped backup plan: one storage target plus the
// schedule that delivers to it. Unlike NotificationChannel (which is per-user),
// backup is a global admin feature, so destinations have no owning user. Each
// destination carries its own time of day, include-assets choice, and archive
// encryption, so "nightly encrypted offsite copy" and "hourly plain local copy"
// are two independent plans rather than one global schedule with a fan-out list.
//
// Destinations that come due at the same time and agree on the archive contents
// (include-assets plus encryption password) still share a single archive, so the
// 3-2-1 case — a local copy and an offsite S3/WebDAV copy of the same bytes —
// costs one snapshot rather than one per destination.
//
// Config holds a destination-type-specific JSON object, encrypted at rest with
// the same envelope used by notification channel configs so secrets (S3 secret
// keys, WebDAV passwords, the archive encryption password) never land in
// plaintext. LastRunAt/LastStatus/LastError are recorded per destination
// because a shared-archive run can succeed for one target and fail for another.
type BackupDestination struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Revision uint64 `gorm:"not null;default:1" json:"revision"`
	Type     string `gorm:"not null;size:20" json:"type"` // local | s3 | webdav
	Enabled  bool   `gorm:"default:false" json:"enabled"`
	Config   string `gorm:"type:text" json:"config"`

	// LastRunAt advances on every successful delivery, manual or scheduled.
	// LastScheduledRunAt advances only for scheduled runs and is what the due
	// check consults, so running a destination by hand does not consume its
	// slot for the day.
	LastRunAt          *time.Time `json:"last_run_at"`
	LastScheduledRunAt *time.Time `json:"last_scheduled_run_at"`

	LastStatus            string    `gorm:"size:20;default:''" json:"last_status"` // success | failed
	LastError             string    `gorm:"type:text" json:"last_error"`
	LastRetentionStatus   string    `gorm:"size:20;default:'not_attempted'" json:"last_retention_status"`
	LastRetentionError    string    `gorm:"type:text" json:"last_retention_error"`
	LastBookkeepingStatus string    `gorm:"size:20;default:'pending'" json:"last_bookkeeping_status"`
	LastBookkeepingError  string    `gorm:"type:text" json:"last_bookkeeping_error"`
	SortOrder             int       `gorm:"default:0" json:"sort_order"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// BackupRun is the durable aggregate for one archive fan-out attempt. The
// archive name is fixed before any delivery begins so a scheduled retry can
// resume the same run without creating a second object for a destination that
// already accepted it.
type BackupRun struct {
	ID                      uint       `gorm:"primaryKey" json:"id"`
	Source                  string     `gorm:"size:20;not null;index" json:"source"` // manual | scheduled
	ArchiveName             string     `gorm:"size:255;not null;uniqueIndex" json:"archive_name"`
	ArchivePath             string     `gorm:"type:text" json:"-"`
	Status                  string     `gorm:"size:20;not null;index" json:"status"` // pending | success | partial | failed | superseded
	DeliveryStatus          string     `gorm:"size:20;not null" json:"delivery_status"`
	RetentionStatus         string     `gorm:"size:20;not null" json:"retention_status"`
	BookkeepingStatus       string     `gorm:"size:20;not null" json:"bookkeeping_status"`
	GlobalBookkeepingStatus string     `gorm:"size:20;not null" json:"global_bookkeeping_status"`
	GlobalBookkeepingError  string     `gorm:"type:text" json:"global_bookkeeping_error"`
	Error                   string     `gorm:"type:text" json:"error"`
	StartedAt               time.Time  `gorm:"not null" json:"started_at"`
	FinishedAt              *time.Time `json:"finished_at"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

// BackupRunDestination stores each stage of a run for one destination. It is
// separate from BackupDestination's last-result summary so interrupted or
// partially successful scheduled runs can be resumed safely.
type BackupRunDestination struct {
	ID                  uint       `gorm:"primaryKey" json:"id"`
	RunID               uint       `gorm:"not null;uniqueIndex:idx_backup_run_destinations" json:"run_id"`
	DestinationID       uint       `gorm:"not null;uniqueIndex:idx_backup_run_destinations;index" json:"destination_id"`
	DestinationRevision uint64     `gorm:"not null" json:"destination_revision"`
	Type                string     `gorm:"size:20;not null" json:"type"`
	DeliveryAttempted   bool       `gorm:"not null;default:false" json:"delivery_attempted"`
	DeliveryStatus      string     `gorm:"size:20;not null" json:"delivery_status"`
	DeliveryError       string     `gorm:"type:text" json:"delivery_error"`
	RetentionStatus     string     `gorm:"size:20;not null" json:"retention_status"`
	RetentionError      string     `gorm:"type:text" json:"retention_error"`
	BookkeepingStatus   string     `gorm:"size:20;not null" json:"bookkeeping_status"`
	BookkeepingError    string     `gorm:"type:text" json:"bookkeeping_error"`
	DeliveredAt         *time.Time `json:"delivered_at"`
	RetentionFinishedAt *time.Time `json:"retention_finished_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}
