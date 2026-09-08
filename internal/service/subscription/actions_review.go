package subscription

import (
	"fmt"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm/clause"
)

// Reviews reuse the existing user/subscription-scoped suppression storage, but
// have separate keys from temporary snoozes and do not count as snoozed tasks.
// Schedule keys describe one decision and one billing period, never the whole
// subscription. Price keys identify an event so a later increase remains visible.
func scheduleReviewKey(sub model.Subscription) string {
	date := ""
	if sub.NextBillingDate != nil {
		date = normalizeDateUTC(*sub.NextBillingDate).Format("2006-01-02")
	}
	end := ""
	if sub.EndsAt != nil {
		end = normalizeDateUTC(*sub.EndsAt).Format("2006-01-02")
	}
	return subscriptionActionKey(sub.ID, "reviewed_schedule", sub.RenewalMode+":"+date+":"+end)
}

func priceReviewKey(subscriptionID, eventID uint) string {
	return subscriptionActionKey(subscriptionID, "reviewed_price", fmt.Sprint(eventID))
}

// Called inside the subscription update transaction: a successful decision and
// its review markers commit together, including when the caller uses batch/MCP.
func (s *Service) recordActionReview(userID uint, before, after model.Subscription) error {
	lifecycleChanged := before.Status != after.Status || before.RenewalMode != after.RenewalMode ||
		!datePtrEqual(before.NextBillingDate, after.NextBillingDate) || !datePtrEqual(before.EndsAt, after.EndsAt)
	if !lifecycleChanged {
		return nil
	}
	// Invalidate old decisions even if a later edit returns to the same dates.
	if err := s.DB.Where("user_id = ? AND subscription_id = ? AND action_key LIKE ?",
		userID, after.ID, subscriptionActionKey(after.ID, "reviewed_schedule", "")+":%").
		Delete(&model.SubscriptionActionSnooze{}).Error; err != nil {
		return err
	}
	if before.RenewalMode == after.RenewalMode || after.Status != subscriptionStatusActive ||
		(after.RenewalMode != renewalModeAutoRenew && after.RenewalMode != renewalModeCancelAtPeriodEnd) ||
		after.NextBillingDate == nil {
		return nil
	}
	end := after.NextBillingDate
	if after.RenewalMode == renewalModeCancelAtPeriodEnd && after.EndsAt != nil {
		end = after.EndsAt
	}
	until := normalizeDateUTC(*end).AddDate(0, 0, 1)
	if err := s.saveActionReview(userID, after.ID, scheduleReviewKey(after), until); err != nil {
		return err
	}

	// Choosing whether to keep the subscription also resolves its existing price
	// decision. Notification failures and future price events are independent.
	items, err := s.priceIncreaseActionsForSubscription(userID, normalizeDateUTC(pkg.NowInSystemTimezone()), after.ID)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.SubscriptionID == after.ID {
			until := normalizeDateUTC(pkg.NowInSystemTimezone()).AddDate(0, 0, actionCenterRecentChangeDays+1)
			if err := s.saveActionReview(userID, after.ID, priceReviewKey(after.ID, item.priceEventID), until); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) saveActionReview(userID, subscriptionID uint, key string, until time.Time) error {
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "subscription_id"}, {Name: "action_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"snoozed_until", "updated_at"}),
	}).Create(&model.SubscriptionActionSnooze{
		UserID: userID, SubscriptionID: subscriptionID, ActionKey: key, SnoozedUntil: until,
	}).Error
}
