package service

import (
	"errors"

	"github.com/shiroha/subdux/internal/model"
)

func (s *SubscriptionService) MarkManualRenewed(userID, id uint) (*model.Subscription, error) {
	sub, err := s.GetByID(userID, id)
	if err != nil {
		return nil, err
	}

	if normalizeStatus(sub.Status) != subscriptionStatusActive {
		return nil, errors.New("only active subscriptions can be marked as renewed")
	}
	if normalizeRenewalMode(sub.RenewalMode) != renewalModeManualRenew {
		return nil, errors.New("only manual_renew subscriptions can be marked as renewed")
	}
	if normalizeBillingType(sub.BillingType) != billingTypeRecurring {
		return nil, errors.New("only recurring subscriptions can be marked as renewed")
	}
	if sub.NextBillingDate == nil {
		return nil, errors.New("next_billing_date is required to mark subscription as renewed")
	}
	if !isRecurringScheduleValid(*sub) {
		return nil, errors.New("subscription recurrence settings are invalid")
	}

	nextBillingDate, ok := nextRecurringOccurrenceAfter(*sub, *sub.NextBillingDate)
	if !ok {
		return nil, errors.New("failed to calculate next billing date")
	}

	if err := s.DB.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{
			"next_billing_date": normalizeDateUTC(nextBillingDate),
			"status":            subscriptionStatusActive,
			"renewal_mode":      renewalModeManualRenew,
			"ends_at":           nil,
			"enabled":           true,
		}).Error; err != nil {
		return nil, err
	}

	return s.GetByID(userID, id)
}
