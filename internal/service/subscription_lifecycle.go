package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type lifecycleDraft struct {
	Status      string
	RenewalMode string
	EndsAt      *time.Time
}

func normalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeRenewalMode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidSubscriptionStatus(value string) bool {
	switch value {
	case subscriptionStatusActive, subscriptionStatusEnded:
		return true
	default:
		return false
	}
}

func isValidRenewalMode(value string) bool {
	switch value {
	case renewalModeAutoRenew, renewalModeManualRenew, renewalModeCancelAtPeriodEnd:
		return true
	default:
		return false
	}
}

func normalizeLifecycleDraft(
	draft lifecycleDraft,
	billingType string,
	nextBillingDate *time.Time,
	now time.Time,
) (lifecycleDraft, error) {
	draft.Status = normalizeStatus(draft.Status)
	if draft.Status == "" {
		draft.Status = subscriptionStatusActive
	}
	if !isValidSubscriptionStatus(draft.Status) {
		return draft, errors.New("status must be one of: active, ended")
	}

	draft.RenewalMode = normalizeRenewalMode(draft.RenewalMode)
	if draft.RenewalMode == "" {
		if draft.Status == subscriptionStatusEnded {
			draft.RenewalMode = renewalModeCancelAtPeriodEnd
		} else if normalizeBillingType(billingType) == billingTypeRecurring {
			draft.RenewalMode = renewalModeAutoRenew
		} else {
			draft.RenewalMode = renewalModeManualRenew
		}
	}
	if !isValidRenewalMode(draft.RenewalMode) {
		return draft, errors.New("renewal_mode must be one of: auto_renew, manual_renew, cancel_at_period_end")
	}

	if draft.EndsAt != nil {
		normalized := normalizeDateUTC(*draft.EndsAt)
		draft.EndsAt = &normalized
	}

	if normalizeBillingType(billingType) != billingTypeRecurring {
		draft.Status = subscriptionStatusEnded
		draft.RenewalMode = renewalModeCancelAtPeriodEnd
		if draft.EndsAt == nil {
			endedAt := normalizeDateUTC(now)
			if nextBillingDate != nil {
				endedAt = normalizeDateUTC(*nextBillingDate)
			}
			draft.EndsAt = &endedAt
		}
		return draft, nil
	}

	if draft.Status == subscriptionStatusEnded {
		if draft.EndsAt == nil {
			endedAt := normalizeDateUTC(now)
			if nextBillingDate != nil {
				endedAt = normalizeDateUTC(*nextBillingDate)
			}
			draft.EndsAt = &endedAt
		}
		return draft, nil
	}

	switch draft.RenewalMode {
	case renewalModeAutoRenew, renewalModeManualRenew:
		draft.EndsAt = nil
	case renewalModeCancelAtPeriodEnd:
		if nextBillingDate == nil {
			return draft, errors.New("next_billing_date is required for cancel_at_period_end subscriptions")
		}
		endsAt := normalizeDateUTC(*nextBillingDate)
		draft.EndsAt = &endsAt
	}

	return draft, nil
}

func subscriptionIsActive(sub model.Subscription) bool {
	return normalizeStatus(sub.Status) == subscriptionStatusActive
}

func syncLegacyEnabledForLifecycle(sub *model.Subscription) {
	if sub == nil {
		return
	}
	sub.Enabled = subscriptionIsActive(*sub)
}

func reconcileSubscriptionLifecycleForUser(db *gorm.DB, userID uint, referenceDate time.Time) error {
	today := normalizeDateUTC(referenceDate)

	var subs []model.Subscription
	if err := db.Where("user_id = ? AND status = ? AND billing_type = ?", userID, subscriptionStatusActive, billingTypeRecurring).
		Find(&subs).Error; err != nil {
		return err
	}

	for i := range subs {
		sub := &subs[i]
		renewalMode := normalizeRenewalMode(sub.RenewalMode)

		switch renewalMode {
		case renewalModeAutoRenew:
			nextBillingDate, changed := nextRecurringBillingDateOnOrAfter(sub, today)
			if !changed || nextBillingDate == nil {
				continue
			}
			if err := db.Model(&model.Subscription{}).
				Where("id = ? AND user_id = ?", sub.ID, userID).
				Updates(map[string]interface{}{
					"next_billing_date": *nextBillingDate,
					"ends_at":           nil,
					"status":            subscriptionStatusActive,
					"enabled":           true,
				}).Error; err != nil {
				return err
			}
		case renewalModeManualRenew:
			if sub.NextBillingDate == nil {
				continue
			}
			endedAt := normalizeDateUTC(*sub.NextBillingDate)
			if !endedAt.Before(today) {
				continue
			}
			if err := db.Model(&model.Subscription{}).
				Where("id = ? AND user_id = ?", sub.ID, userID).
				Updates(map[string]interface{}{
					"status":  subscriptionStatusEnded,
					"ends_at": endedAt,
					"enabled": false,
				}).Error; err != nil {
				return err
			}
		case renewalModeCancelAtPeriodEnd:
			boundary := sub.EndsAt
			if boundary == nil {
				boundary = sub.NextBillingDate
			}
			if boundary == nil {
				continue
			}
			endedAt := normalizeDateUTC(*boundary)
			if !endedAt.Before(today) {
				continue
			}
			if err := db.Model(&model.Subscription{}).
				Where("id = ? AND user_id = ?", sub.ID, userID).
				Updates(map[string]interface{}{
					"status":  subscriptionStatusEnded,
					"ends_at": endedAt,
					"enabled": false,
				}).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func deriveLegacyLifecycle(enabled bool, billingType string, nextBillingDate, endsAt *time.Time, updatedAt time.Time) lifecycleDraft {
	if normalizeBillingType(billingType) != billingTypeRecurring {
		draft := lifecycleDraft{
			Status:      subscriptionStatusEnded,
			RenewalMode: renewalModeCancelAtPeriodEnd,
			EndsAt:      copyTimePointer(endsAt),
		}
		if draft.EndsAt == nil {
			if nextBillingDate != nil {
				draft.EndsAt = copyTimePointer(nextBillingDate)
			} else {
				endedAt := normalizeDateUTC(updatedAt)
				draft.EndsAt = &endedAt
			}
		}
		return draft
	}

	status := subscriptionStatusActive
	if !enabled {
		status = subscriptionStatusEnded
	}

	renewalMode := renewalModeAutoRenew
	if status == subscriptionStatusEnded {
		renewalMode = renewalModeCancelAtPeriodEnd
	}

	draft := lifecycleDraft{
		Status:      status,
		RenewalMode: renewalMode,
		EndsAt:      copyTimePointer(endsAt),
	}
	if draft.EndsAt == nil && status == subscriptionStatusEnded {
		if nextBillingDate != nil {
			draft.EndsAt = copyTimePointer(nextBillingDate)
		} else {
			endedAt := normalizeDateUTC(updatedAt)
			draft.EndsAt = &endedAt
		}
	}
	return draft
}

func markSubscriptionEndedNow(sub *model.Subscription, now time.Time) {
	if sub == nil {
		return
	}
	endedAt := normalizeDateUTC(now)
	sub.Status = subscriptionStatusEnded
	sub.EndsAt = &endedAt
	syncLegacyEnabledForLifecycle(sub)
}
