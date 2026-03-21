package pkg

import (
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	subscriptionStatusActive           = "active"
	subscriptionStatusEnded            = "ended"
	subscriptionRenewalModeAutoRenew   = "auto_renew"
	subscriptionRenewalModeManualRenew = "manual_renew"
	subscriptionRenewalModeCancelEnd   = "cancel_at_period_end"
	subscriptionBillingTypeRecurring   = "recurring"
)

func backfillSubscriptionLifecycleFields(db *gorm.DB) error {
	var subs []model.Subscription
	if err := db.Find(&subs).Error; err != nil {
		return err
	}

	for i := range subs {
		sub := subs[i]
		updates := map[string]interface{}{}
		billingType := strings.ToLower(strings.TrimSpace(sub.BillingType))
		migratingLegacyBuyout := billingType != subscriptionBillingTypeRecurring

		status := strings.ToLower(strings.TrimSpace(sub.Status))
		if migratingLegacyBuyout {
			status = subscriptionStatusEnded
			updates["status"] = status
		} else if status == "" {
			if sub.Enabled {
				status = subscriptionStatusActive
			} else {
				status = subscriptionStatusEnded
			}
			updates["status"] = status
		}

		renewalMode := strings.ToLower(strings.TrimSpace(sub.RenewalMode))
		if migratingLegacyBuyout {
			renewalMode = subscriptionRenewalModeCancelEnd
			updates["renewal_mode"] = renewalMode
		} else if renewalMode == "" {
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

		if migratingLegacyBuyout {
			updates["billing_type"] = subscriptionBillingTypeRecurring
			updates["recurrence_type"] = ""
			updates["interval_count"] = nil
			updates["interval_unit"] = ""
			updates["monthly_day"] = nil
			updates["yearly_month"] = nil
			updates["yearly_day"] = nil
			billingType = subscriptionBillingTypeRecurring
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

func normalizeSubscriptionDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
