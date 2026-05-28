package pkg

import (
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func backfillSubscriptionLifecycleFields(db *gorm.DB) error {
	var subs []model.Subscription
	if err := db.Find(&subs).Error; err != nil {
		return err
	}

	for i := range subs {
		sub := subs[i]
		updates := map[string]interface{}{}

		status := normalizeSubscriptionLifecycleValue(sub.Status)
		if status == "" {
			if sub.Enabled {
				status = subscriptionStatusActive
			} else {
				status = subscriptionStatusEnded
			}
			updates["status"] = status
		}

		renewalMode := normalizeSubscriptionLifecycleValue(sub.RenewalMode)
		if renewalMode == "" {
			if status == subscriptionStatusEnded {
				renewalMode = subscriptionRenewalModeCancelEnd
			} else {
				renewalMode = subscriptionRenewalModeAutoRenew
			}
			updates["renewal_mode"] = renewalMode
		}

		if status == subscriptionStatusEnded && sub.EndsAt == nil {
			if sub.NextBillingDate != nil {
				endedAt := normalizeSubscriptionDate(*sub.NextBillingDate)
				updates["ends_at"] = endedAt
			} else {
				endedAt := normalizeSubscriptionDate(sub.UpdatedAt)
				if endedAt.IsZero() {
					endedAt = normalizeSubscriptionDate(time.Now().UTC())
				}
				updates["ends_at"] = endedAt
			}
		}

		expectedEnabled := status == subscriptionStatusActive
		if sub.Enabled != expectedEnabled {
			updates["enabled"] = expectedEnabled
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

func normalizeSubscriptionLifecycleValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeSubscriptionDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
