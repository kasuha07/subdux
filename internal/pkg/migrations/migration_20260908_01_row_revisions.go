package migrations

import "gorm.io/gorm"

// Fixed SQL keeps this migration independent of future model changes. Existing
// rows and new rows start at revision 1. backup_destinations already has it.
func migrateRowRevisions(db *gorm.DB) error {
	for _, table := range []string{"subscriptions", "notification_policies", "notification_channels", "notification_templates", "categories", "payment_methods", "user_currencies", "system_settings"} {
		if !db.Migrator().HasColumn(table, "revision") {
			if err := db.Exec("ALTER TABLE " + table + " ADD COLUMN revision INTEGER NOT NULL DEFAULT 1").Error; err != nil {
				return err
			}
		}
	}
	return nil
}
