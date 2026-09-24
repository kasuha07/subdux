package migrations

import "gorm.io/gorm"

// The event field list remains text in the shared SQLite/PostgreSQL model and
// backup format. PostgreSQL can index its JSON array representation directly.
func migrateSubscriptionEventFieldsIndex(db *gorm.DB) error {
	if db.Dialector.Name() != "postgres" {
		return nil
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_subscription_events_changed_fields_jsonb
		ON subscription_events USING gin ((changed_fields::jsonb))`).Error
}
