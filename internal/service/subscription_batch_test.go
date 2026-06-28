package service

import (
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

// registerQueryCounter attaches a counter to the database's query callback so a
// test can assert that a code path issues a constant number of SELECTs rather
// than one per row (the N+1 pattern this batch loading replaced).
func registerQueryCounter(t *testing.T, db *gorm.DB, counter *int) {
	t.Helper()
	if err := db.Callback().Query().After("gorm:query").Register("test:count_queries", func(tx *gorm.DB) {
		*counter++
	}); err != nil {
		t.Fatalf("register query counter failed: %v", err)
	}
}

func seedPriceIncreaseFixtures(t *testing.T, service *SubscriptionService, db *gorm.DB, userID uint, count int) {
	t.Helper()
	monthly := 1
	for i := 0; i < count; i++ {
		sub, err := service.Create(userID, CreateSubscriptionInput{
			Name:            "Sub",
			Amount:          10,
			Currency:        "USD",
			Status:          subscriptionStatusActive,
			RenewalMode:     renewalModeAutoRenew,
			BillingType:     billingTypeRecurring,
			RecurrenceType:  recurrenceTypeInterval,
			IntervalCount:   &monthly,
			IntervalUnit:    intervalUnitMonth,
			NextBillingDate: "2026-04-15",
		})
		if err != nil {
			t.Fatalf("create subscription failed: %v", err)
		}
		increased := 20.0
		if _, err := service.Update(userID, sub.ID, UpdateSubscriptionInput{Amount: &increased}); err != nil {
			t.Fatalf("increase amount failed: %v", err)
		}
		failedAt := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
		if err := db.Create(&model.NotificationLog{
			UserID:         userID,
			SubscriptionID: sub.ID,
			ChannelType:    "webhook",
			NotifyDate:     mustDate(t, "2026-03-04"),
			Status:         notificationLogStatusFailed,
			Error:          "timeout",
			SentAt:         failedAt,
		}).Error; err != nil {
			t.Fatalf("create failed log failed: %v", err)
		}
	}
}

// TestActionCenterReadPathsHaveConstantQueryCount proves the price-increase,
// notification-failure, and annual-growth-baseline paths issue the same number
// of queries whether they process two subscriptions or many. A per-row lookup
// would make the larger fixture issue strictly more queries; equal counts is
// the regression guard against reintroducing N+1.
func TestActionCenterReadPathsHaveConstantQueryCount(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-10"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	service := NewSubscriptionService(db)

	small := createTestUser(t, db)
	large := model.User{Username: "large", Email: "large@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&large).Error; err != nil {
		t.Fatalf("create large user failed: %v", err)
	}

	seedPriceIncreaseFixtures(t, service, db, small.ID, 2)
	seedPriceIncreaseFixtures(t, service, db, large.ID, 8)

	today := normalizeDateUTC(pkg.NowInSystemTimezone())

	var counter int
	registerQueryCounter(t, db, &counter)

	measure := func(userID uint, run func() error) int {
		counter = 0
		if err := run(); err != nil {
			t.Fatalf("read path failed: %v", err)
		}
		return counter
	}

	cases := []struct {
		name string
		run  func(userID uint) func() error
	}{
		{
			name: "priceIncreaseActions",
			run: func(userID uint) func() error {
				return func() error { _, err := service.priceIncreaseActions(userID, today); return err }
			},
		},
		{
			name: "notificationFailureActions",
			run: func(userID uint) func() error {
				return func() error { _, err := service.notificationFailureActions(userID, today); return err }
			},
		},
		{
			name: "annualGrowthBaselineMonthlyAmounts",
			run: func(userID uint) func() error {
				return func() error {
					_, err := service.annualGrowthBaselineMonthlyAmounts(userID, "USD", nil, pkg.NowInSystemTimezone())
					return err
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			smallCount := measure(small.ID, tc.run(small.ID))
			largeCount := measure(large.ID, tc.run(large.ID))
			if smallCount != largeCount {
				t.Fatalf("query count grew with row count: 2 subs => %d queries, 8 subs => %d queries (N+1 regression)", smallCount, largeCount)
			}
		})
	}
}
