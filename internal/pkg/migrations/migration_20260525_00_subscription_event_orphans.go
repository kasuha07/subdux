package migrations

import (
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func cleanupSubscriptionEventOrphans(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.SubscriptionEvent{}) {
		return nil
	}

	if err := db.Exec(`
		DELETE FROM subscription_events
		WHERE user_id NOT IN (SELECT id FROM users)
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		UPDATE subscription_events
		SET subscription_id = NULL
		WHERE subscription_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1
		      FROM subscriptions
		      WHERE subscriptions.id = subscription_events.subscription_id
		  )
	`).Error; err != nil {
		return err
	}

	if err := db.Exec(`
		UPDATE subscription_events
		SET subscription_id = NULL
		WHERE subscription_id IS NOT NULL
		  AND EXISTS (
		      SELECT 1
		      FROM subscriptions
		      WHERE subscriptions.id = subscription_events.subscription_id
		        AND subscriptions.user_id != subscription_events.user_id
		  )
	`).Error; err != nil {
		return err
	}

	return db.AutoMigrate(&model.SubscriptionEvent{})
}
