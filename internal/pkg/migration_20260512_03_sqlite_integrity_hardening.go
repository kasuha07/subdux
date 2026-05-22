package pkg

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const maxNotifyDaysBeforeSQLite = 10

func migrateSQLiteIntegrityHardening(db *gorm.DB) error {
	if err := normalizeDataForSQLiteConstraints(db); err != nil {
		return err
	}

	modelsToRebuild := []interface{}{
		&model.User{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.EmailVerificationCode{},
		&model.UserBackupCode{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationTemplate{},
		&model.APIKey{},
		&model.RefreshToken{},
		&model.CalendarToken{},
		&model.NotificationLog{},
	}

	return withSQLiteForeignKeysDisabled(db, func(tx *gorm.DB) error {
		for _, value := range modelsToRebuild {
			if err := rebuildSQLiteTable(tx, value); err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizeDataForSQLiteConstraints(db *gorm.DB) error {
	if err := normalizeUsersForSQLiteConstraints(db); err != nil {
		return err
	}
	if err := cleanupUserScopedOrphans(db); err != nil {
		return err
	}
	if err := normalizeSubscriptionsForSQLiteConstraints(db); err != nil {
		return err
	}
	if err := normalizeNotificationPoliciesForSQLiteConstraints(db); err != nil {
		return err
	}
	return cleanupNotificationLogOrphans(db)
}

func normalizeUsersForSQLiteConstraints(db *gorm.DB) error {
	var users []model.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}

	for i := range users {
		user := users[i]
		updates := map[string]interface{}{}

		role := strings.ToLower(strings.TrimSpace(user.Role))
		switch role {
		case "admin", "user":
		default:
			role = "user"
		}
		if role != user.Role {
			updates["role"] = role
		}

		status := strings.ToLower(strings.TrimSpace(user.Status))
		switch status {
		case "active", "disabled":
		default:
			status = "active"
		}
		if status != user.Status {
			updates["status"] = status
		}

		if len(updates) == 0 {
			continue
		}
		if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func cleanupUserScopedOrphans(db *gorm.DB) error {
	userIDs := db.Model(&model.User{}).Select("id")

	for _, value := range []interface{}{
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.UserBackupCode{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationTemplate{},
		&model.APIKey{},
		&model.RefreshToken{},
		&model.CalendarToken{},
	} {
		if err := db.Where("user_id NOT IN (?)", userIDs).Delete(value).Error; err != nil {
			return err
		}
	}

	if err := db.Model(&model.EmailVerificationCode{}).
		Where("user_id IS NOT NULL AND user_id NOT IN (?)", userIDs).
		Update("user_id", nil).Error; err != nil {
		return err
	}

	return nil
}

func normalizeSubscriptionsForSQLiteConstraints(db *gorm.DB) error {
	var subscriptions []model.Subscription
	if err := db.Find(&subscriptions).Error; err != nil {
		return err
	}

	for i := range subscriptions {
		sub := subscriptions[i]
		updates := map[string]interface{}{}

		if sub.Amount < 0 {
			updates["amount"] = 0
		}

		status := strings.ToLower(strings.TrimSpace(sub.Status))
		if status != subscriptionStatusActive && status != subscriptionStatusEnded {
			if sub.Enabled {
				status = subscriptionStatusActive
			} else {
				status = subscriptionStatusEnded
			}
		}
		if status != sub.Status {
			updates["status"] = status
		}

		renewalMode := strings.ToLower(strings.TrimSpace(sub.RenewalMode))
		switch renewalMode {
		case subscriptionRenewalModeAutoRenew, subscriptionRenewalModeManualRenew, subscriptionRenewalModeCancelEnd:
		default:
			if status == subscriptionStatusEnded {
				renewalMode = subscriptionRenewalModeCancelEnd
			} else {
				renewalMode = subscriptionRenewalModeAutoRenew
			}
		}
		if renewalMode != sub.RenewalMode {
			updates["renewal_mode"] = renewalMode
		}

		if status == subscriptionStatusEnded && sub.EndsAt == nil {
			endedAt := normalizeSubscriptionDate(sub.UpdatedAt)
			if sub.NextBillingDate != nil {
				endedAt = normalizeSubscriptionDate(*sub.NextBillingDate)
			} else if endedAt.IsZero() {
				endedAt = normalizeSubscriptionDate(time.Now().UTC())
			}
			updates["ends_at"] = endedAt
		}

		expectedEnabled := status == subscriptionStatusActive
		if sub.Enabled != expectedEnabled {
			updates["enabled"] = expectedEnabled
		}

		if sub.NotifyDaysBefore != nil {
			days := *sub.NotifyDaysBefore
			if days < 0 || days > maxNotifyDaysBeforeSQLite {
				updates["notify_days_before"] = nil
			}
		}

		if sub.CategoryID != nil {
			var category model.Category
			err := db.Select("id", "user_id").First(&category, *sub.CategoryID).Error
			if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && category.UserID != sub.UserID) {
				updates["category_id"] = nil
			} else if err != nil {
				return err
			}
		}

		if sub.PaymentMethodID != nil {
			var method model.PaymentMethod
			err := db.Select("id", "user_id").First(&method, *sub.PaymentMethodID).Error
			if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && method.UserID != sub.UserID) {
				updates["payment_method_id"] = nil
			} else if err != nil {
				return err
			}
		}

		if len(updates) == 0 {
			continue
		}
		if err := db.Model(&model.Subscription{}).Where("id = ?", sub.ID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func normalizeNotificationPoliciesForSQLiteConstraints(db *gorm.DB) error {
	var policies []model.NotificationPolicy
	if err := db.Find(&policies).Error; err != nil {
		return err
	}

	for i := range policies {
		policy := policies[i]
		days := policy.DaysBefore
		if days >= 0 && days <= maxNotifyDaysBeforeSQLite {
			continue
		}
		if err := db.Model(&model.NotificationPolicy{}).
			Where("id = ?", policy.ID).
			Update("days_before", 3).Error; err != nil {
			return err
		}
	}

	return nil
}

func cleanupNotificationLogOrphans(db *gorm.DB) error {
	var logs []model.NotificationLog
	if err := db.Find(&logs).Error; err != nil {
		return err
	}

	for i := range logs {
		entry := logs[i]

		var user model.User
		if err := db.Select("id").First(&user, entry.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Delete(&model.NotificationLog{}, entry.ID).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}

		var sub model.Subscription
		if err := db.Select("id", "user_id").First(&sub, entry.SubscriptionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := db.Delete(&model.NotificationLog{}, entry.ID).Error; err != nil {
					return err
				}
				continue
			}
			return err
		}
		if sub.UserID != entry.UserID {
			if err := db.Delete(&model.NotificationLog{}, entry.ID).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
