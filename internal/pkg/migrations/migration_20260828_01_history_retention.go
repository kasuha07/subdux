package migrations

import (
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

// recentHistoryRetentionLimit is the number of recent records retained for
// each user in the notification and MCP audit histories. Keep this in sync
// with the write-time limits in the corresponding service packages.
const recentHistoryRetentionLimit = 30

// migrateHistoryRetention removes historical rows beyond the bounded recent
// history that the application exposes. The service layer maintains the same
// bound after each new record is written; this migration also cleans existing
// databases so old rows do not remain unbounded after an upgrade.
func migrateHistoryRetention(db *gorm.DB) error {
	if err := pruneNotificationLogHistory(db); err != nil {
		return err
	}
	return pruneAuditEventHistory(db)
}

func pruneNotificationLogHistory(db *gorm.DB) error {
	var userIDs []uint
	if err := db.Model(&model.NotificationLog{}).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		keepIDs := db.Model(&model.NotificationLog{}).
			Select("id").
			Where("user_id = ?", userID).
			Order("sent_at DESC, id DESC").
			Limit(recentHistoryRetentionLimit)
		if err := db.Where("user_id = ? AND id NOT IN (?)", userID, keepIDs).
			Delete(&model.NotificationLog{}).Error; err != nil {
			return err
		}
	}
	return nil
}

func pruneAuditEventHistory(db *gorm.DB) error {
	var userIDs []uint
	if err := db.Model(&model.AuditEvent{}).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		keepIDs := db.Model(&model.AuditEvent{}).
			Select("event_id").
			Where("user_id = ?", userID).
			Order("occurred_at DESC, event_id DESC").
			Limit(recentHistoryRetentionLimit)
		if err := db.Where("user_id = ? AND event_id NOT IN (?)", userID, keepIDs).
			Delete(&model.AuditEvent{}).Error; err != nil {
			return err
		}
	}
	return nil
}
