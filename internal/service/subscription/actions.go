package subscription

import (
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

func (s *Service) MarkManualRenewed(userID, id uint) (*model.Subscription, error) {
	// Persist any due lifecycle transition first so the active/renewal checks
	// below run against the subscription's true current state rather than a row
	// the background sweep has not yet caught up on.
	if err := reconcileSubscriptionForWrite(s.DB, userID, id, pkg.NowInSystemTimezone()); err != nil {
		return nil, err
	}

	sub, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}

	if normalizeStatus(sub.Status) != subscriptionStatusActive {
		return nil, ErrOnlyActiveCanRenew
	}
	if normalizeRenewalMode(sub.RenewalMode) != renewalModeManualRenew {
		return nil, ErrOnlyManualRenewCanRenew
	}
	if normalizeBillingType(sub.BillingType) != billingTypeRecurring {
		return nil, ErrOnlyRecurringCanRenew
	}
	if sub.NextBillingDate == nil {
		return nil, ErrNextBillingDateRequiredMark
	}
	if !isRecurringScheduleValid(*sub) {
		return nil, ErrRecurrenceSettingsInvalid
	}
	before := *sub

	nextBillingDate, ok := nextRecurringOccurrenceAfter(*sub, *sub.NextBillingDate)
	if !ok {
		return nil, ErrNextBillingDateCalcFailed
	}

	var updated model.Subscription
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Subscription{}).
			Where("id = ? AND user_id = ?", id, userID).
			Updates(map[string]interface{}{
				"next_billing_date": normalizeDateUTC(nextBillingDate),
				"status":            subscriptionStatusActive,
				"renewal_mode":      renewalModeManualRenew,
				"ends_at":           nil,
				"enabled":           true,
			}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&updated).Error; err != nil {
			return err
		}
		normalizeSubscriptionForResponse(&updated)
		return (&Service{DB: tx}).recordSubscriptionChanged(userID, before, updated, subscriptionEventManualRenewed)
	}); err != nil {
		return nil, err
	}

	return &updated, nil
}
