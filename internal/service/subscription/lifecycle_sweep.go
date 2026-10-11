package subscription

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	subscriptionLifecycleSweepTaskKey  = "subscription_lifecycle_sweep"
	subscriptionLifecycleSweepLeaseTTL = 30 * time.Minute
	postgresLifecycleSweepBatchSize    = 100
)

// ReconcileDueLifecycles advances subscription lifecycle for every user that
// owns an active recurring subscription. It is the primary driver of lifecycle
// transitions: by rolling renewals and ending overdue subscriptions on a timer,
// the common boundary-crossing case is handled in the background so read
// requests issue no writes in steady state. The read path keeps its own
// reconcile as a correctness backstop for the window between sweeps.
//
// SQLite uses a background-task lease keyed by ownerID. PostgreSQL claims
// disjoint due rows with SKIP LOCKED so several instances can sweep at once.
func (s *Service) ReconcileDueLifecycles(ownerID string) error {
	if pkg.IsPostgres(s.DB) {
		return s.reconcileDueLifecyclesPostgres(pkg.NowInSystemTimezone())
	}
	return serviceutil.WithBackgroundTaskLease(s.DB, ownerID, subscriptionLifecycleSweepTaskKey, subscriptionLifecycleSweepLeaseTTL, func() error {
		return s.reconcileDueLifecycles(pkg.NowInSystemTimezone())
	})
}

// PostgreSQL workers claim disjoint batches of due rows. Keep the transaction
// short and retain the revision check in persistAdvancedSubscriptionLifecycle:
// other write paths can still change a subscription between sweep passes.
func (s *Service) reconcileDueLifecyclesPostgres(now time.Time) error {
	today := normalizeDateUTC(now)
	var lastID uint
	for {
		var subs []model.Subscription
		err := s.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("id > ? AND status = ? AND billing_type = ? AND (next_billing_date < ? OR ends_at < ? OR pending_from <= ?)",
					lastID, subscriptionStatusActive, billingTypeRecurring, today, today, today).
				Order("id ASC").Limit(postgresLifecycleSweepBatchSize).Find(&subs).Error; err != nil {
				return err
			}
			for i := range subs {
				if err := persistAdvancedSubscriptionLifecycle(tx, subs[i].UserID, &subs[i], now); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if len(subs) == 0 {
			return nil
		}
		lastID = subs[len(subs)-1].ID
	}
}

func (s *Service) reconcileDueLifecycles(now time.Time) error {
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
