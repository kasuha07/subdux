package migrations

import "gorm.io/gorm"

// migrateUserTOTPLastCounter adds users.totp_last_counter, the last TOTP time
// step (Unix time / 30s) the account accepted. Login and step-up verification
// advance it atomically so a code cannot be replayed within its validity
// window. Existing rows start at 0, which accepts any current code once.
// Fixed SQL keeps this migration independent of future model changes; a column
// that already exists (fresh databases created from the current model) is left
// alone.
func migrateUserTOTPLastCounter(db *gorm.DB) error {
	if !db.Migrator().HasTable("users") || db.Migrator().HasColumn("users", "totp_last_counter") {
		return nil
	}
	columnType := "integer"
	if db.Dialector.Name() == "postgres" {
		columnType = "bigint"
	}
	return db.Exec("ALTER TABLE users ADD COLUMN totp_last_counter " + columnType + " NOT NULL DEFAULT 0").Error
}
