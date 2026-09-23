package migrations

import (
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func migrateUserJevSettings(db *gorm.DB) error {
	if db.Dialector.Name() == "postgres" {
		return db.AutoMigrate(&model.UserJevSetting{})
	}

	return db.Exec(`CREATE TABLE IF NOT EXISTS user_jev_settings (
		user_id integer PRIMARY KEY,
		revision integer NOT NULL DEFAULT 1,
		enabled numeric NOT NULL DEFAULT false,
		api_key text NOT NULL DEFAULT '',
		updated_at datetime,
		CONSTRAINT fk_user_jev_settings_user FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
	)`).Error
}
