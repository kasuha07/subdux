package service

import (
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

const (
	subscriptionLifecycleSweepTaskKey  = "subscription_lifecycle_sweep"
	subscriptionLifecycleSweepLeaseTTL = 30 * time.Minute
)

// ReconcileDueLifecycles advances subscription lifecycle for every user that
// owns an active recurring subscription. It is the primary driver of lifecycle
// transitions: by rolling renewals and ending overdue subscriptions on a timer,
// the common boundary-crossing case is handled in the background so read
// requests issue no writes in steady state. The read path keeps its own
// reconcile as a correctness backstop for the window between sweeps.
//
// The work is guarded by a background-task lease so only one instance runs it
// when several share a database. ownerID identifies this process for the lease.
func (s *SubscriptionService) ReconcileDueLifecycles(ownerID string) error {
	return withBackgroundTaskLease(s.DB, ownerID, subscriptionLifecycleSweepTaskKey, subscriptionLifecycleSweepLeaseTTL, func() error {
		return s.reconcileDueLifecycles(pkg.NowInSystemTimezone())
	})
}

func (s *SubscriptionService) reconcileDueLifecycles(now time.Time) error {
	var userIDs []uint
	if err := s.DB.Model(&model.Subscription{}).
		Where("status = ? AND billing_type = ?", subscriptionStatusActive, billingTypeRecurring).
		Distinct("user_id").
		Pluck("user_id", &userIDs).Error; err != nil {
		return err
	}

	for _, userID := range userIDs {
		if err := reconcileSubscriptionLifecycleForUser(s.DB, userID, now); err != nil {
			return err
		}
	}
	return nil
}
