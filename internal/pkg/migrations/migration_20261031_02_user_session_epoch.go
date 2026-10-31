package migrations

import "gorm.io/gorm"

// migrateUserSessionEpoch adds users.session_epoch, the per-user counter that
// human access JWTs carry and that global session revocation (logout-all,
// password or email change, account disable) advances. Existing rows start at
// 0, which matches access tokens issued before this migration, so upgrading
// does not sign anyone out. Fixed SQL keeps this migration independent of
// future model changes; a column that already exists (fresh databases created
// from the current model) is left alone.
func migrateUserSessionEpoch(db *gorm.DB) error {
	if !db.Migrator().HasTable("users") || db.Migrator().HasColumn("users", "session_epoch") {
		return nil
	}
	columnType := "integer"
	if db.Dialector.Name() == "postgres" {
		columnType = "bigint"
	}
	return db.Exec("ALTER TABLE users ADD COLUMN session_epoch " + columnType + " NOT NULL DEFAULT 0").Error
}
