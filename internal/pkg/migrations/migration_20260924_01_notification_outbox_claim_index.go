package migrations

import "gorm.io/gorm"

// migrateNotificationOutboxClaimIndex adds a PostgreSQL-only index for the
// due-job ordering used by the SKIP LOCKED claim query. SQLite keeps its
// existing schema and claim path unchanged.
func migrateNotificationOutboxClaimIndex(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}

	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_notification_outboxes_due_claim
		ON notification_outboxes (next_attempt_at ASC, id ASC)
		WHERE status IN ('pending', 'processing')`).Error
}
