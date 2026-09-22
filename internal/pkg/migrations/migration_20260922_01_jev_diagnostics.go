package migrations

import "gorm.io/gorm"

func migrateJevDiagnostics(db *gorm.DB) error {
	for _, statement := range []string{
		`ALTER TABLE user_jev_settings ADD COLUMN connection_status text NOT NULL DEFAULT 'not_tested'`,
		`ALTER TABLE user_jev_settings ADD COLUMN last_checked_at datetime`,
		`ALTER TABLE user_jev_settings ADD COLUMN last_success_at datetime`,
		`ALTER TABLE user_jev_settings ADD COLUMN classification_requests integer NOT NULL DEFAULT 0`,
		`ALTER TABLE user_jev_settings ADD COLUMN classification_suggestions integer NOT NULL DEFAULT 0`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
