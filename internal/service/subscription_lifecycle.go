package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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
		} else {
			draft.RenewalMode = renewalModeAutoRenew
		}
	}
	if !isValidRenewalMode(draft.RenewalMode) {
		return draft, errors.New("renewal_mode must be one of: auto_renew, manual_renew, cancel_at_period_end")
	}

	if draft.EndsAt != nil {
		normalized := normalizeDateUTC(*draft.EndsAt)
		draft.EndsAt = &normalized
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

// advanceSubscriptionLifecycle applies, in memory, the lifecycle transition a
// subscription is due as of referenceDate: an auto-renew subscription rolls its
// next billing date forward, while manual-renew and cancel-at-period-end
// subscriptions end once their boundary has passed. It mutates sub and reports
// whether anything changed.
//
// This is the single source of truth for lifecycle progression. Read paths call
// it to present the correct state without writing; the write path and the
// background sweep call it through persistAdvancedSubscriptionLifecycle to
// persist that state. A user therefore sees the same lifecycle whether or not
// the database row has caught up yet.
func advanceSubscriptionLifecycle(sub *model.Subscription, referenceDate time.Time) bool {
	if sub == nil || !subscriptionIsActive(*sub) || sub.BillingType != billingTypeRecurring {
		return false
	}

	today := normalizeDateUTC(referenceDate)
	switch normalizeRenewalMode(sub.RenewalMode) {
	case renewalModeAutoRenew:
		nextBillingDate, changed := nextRecurringBillingDateOnOrAfter(sub, today)
		if !changed || nextBillingDate == nil {
			return false
		}
		sub.NextBillingDate = nextBillingDate
		sub.EndsAt = nil
		sub.Status = subscriptionStatusActive
		sub.Enabled = true
		return true
	case renewalModeManualRenew:
		if sub.NextBillingDate == nil {
			return false
		}
		endedAt := normalizeDateUTC(*sub.NextBillingDate)
		if !endedAt.Before(today) {
			return false
		}
		markSubscriptionEndedAt(sub, endedAt)
		return true
	case renewalModeCancelAtPeriodEnd:
		boundary := cancelAtPeriodEndBoundary(*sub)
		if boundary == nil {
			return false
		}
		endedAt := normalizeDateUTC(*boundary)
		if !endedAt.Before(today) {
			return false
		}
		markSubscriptionEndedAt(sub, endedAt)
		return true
	}
	return false
}

func markSubscriptionEndedAt(sub *model.Subscription, endedAt time.Time) {
	sub.Status = subscriptionStatusEnded
	sub.EndsAt = &endedAt
	sub.Enabled = false
}

// persistAdvancedSubscriptionLifecycle advances sub and, when its state changed,
// writes the updated lifecycle columns. It is used by write paths and the
// background sweep; read paths must never call it.
func persistAdvancedSubscriptionLifecycle(db *gorm.DB, userID uint, sub *model.Subscription, referenceDate time.Time) error {
	if !advanceSubscriptionLifecycle(sub, referenceDate) {
		return nil
	}
	return db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, userID).
		Updates(map[string]interface{}{
			"next_billing_date": sub.NextBillingDate,
			"ends_at":           sub.EndsAt,
			"status":            sub.Status,
			"enabled":           sub.Enabled,
		}).Error
}

// reconcileSubscriptionLifecycleForUser persists any due lifecycle transitions
// for a user's active recurring subscriptions. It is invoked by the background
// sweep and by write paths, not by ordinary read requests.
func reconcileSubscriptionLifecycleForUser(db *gorm.DB, userID uint, referenceDate time.Time) error {
	var subs []model.Subscription
	if err := db.Where("user_id = ? AND status = ? AND billing_type = ?", userID, subscriptionStatusActive, billingTypeRecurring).
		Find(&subs).Error; err != nil {
		return err
	}

	for i := range subs {
		if err := persistAdvancedSubscriptionLifecycle(db, userID, &subs[i], referenceDate); err != nil {
			return err
		}
	}

	return nil
}

// presentSubscriptionForResponse advances a subscription's lifecycle in memory
// and normalizes legacy fields, producing the state a read request should
// return without writing to the database.
func presentSubscriptionForResponse(sub *model.Subscription, referenceDate time.Time) {
	advanceSubscriptionLifecycle(sub, referenceDate)
	normalizeSubscriptionForResponse(sub)
}

// reconcileSubscriptionForWrite persists any due lifecycle transition for a
// single subscription before a write path mutates it. Read paths advance state
// in memory only; a mutation, by contrast, must operate on a database row that
// already reflects the current lifecycle so it does not resurrect or overwrite a
// transition that should have happened. Missing rows are ignored: the caller's
// own load reports not-found.
func reconcileSubscriptionForWrite(db *gorm.DB, userID, id uint, referenceDate time.Time) error {
	var sub model.Subscription
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return persistAdvancedSubscriptionLifecycle(db, userID, &sub, referenceDate)
}

// ReconcileUserLifecycle persists all due lifecycle transitions for a single
// user. It is the explicit, on-demand counterpart to the background sweep —
// callable from an API endpoint or admin action to repair a user's state
// immediately rather than waiting for the next sweep.
func (s *SubscriptionService) ReconcileUserLifecycle(userID uint) error {
	return reconcileSubscriptionLifecycleForUser(s.DB, userID, pkg.NowInSystemTimezone())
}

func deriveLegacyLifecycle(enabled bool, nextBillingDate, endsAt *time.Time, updatedAt time.Time) lifecycleDraft {
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
