package service

import (
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestGetActionCenterAggregatesRenewalRepairAndHistoryItems(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.NotificationLog{}); err != nil {
		t.Fatalf("failed to migrate notification logs: %v", err)
	}
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	autoRenew, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          12,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-05",
	})
	if err != nil {
		t.Fatalf("create auto-renew subscription failed: %v", err)
	}

	manualRenew, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Domain",
		Amount:          20,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitYear,
		NextBillingDate: "2026-03-03",
	})
	if err != nil {
		t.Fatalf("create manual-renew subscription failed: %v", err)
	}

	ending, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Old Tool",
		Amount:          9,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-20",
	})
	if err != nil {
		t.Fatalf("create ending subscription failed: %v", err)
	}

	missingDate := model.Subscription{
		UserID:          user.ID,
		Name:            "Broken Plan",
		Amount:          4,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: nil,
	}
	if err := db.Create(&missingDate).Error; err != nil {
		t.Fatalf("create missing-date subscription failed: %v", err)
	}

	if err := db.Create(&model.NotificationLog{
		UserID:         user.ID,
		SubscriptionID: autoRenew.ID,
		ChannelType:    "webhook",
		NotifyDate:     mustDate(t, "2026-03-04"),
		Status:         "failed",
		Error:          "timeout",
		SentAt:         time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create failed notification log failed: %v", err)
	}

	updatedAmount := 18.0
	if _, err := service.Update(user.ID, autoRenew.ID, UpdateSubscriptionInput{Amount: &updatedAmount}); err != nil {
		t.Fatalf("update subscription amount failed: %v", err)
	}

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	if got, want := center.WindowDays, actionCenterUpcomingDays; got != want {
		t.Fatalf("window_days = %d, want %d", got, want)
	}
	if got, want := center.Counts.Total, 6; got != want {
		t.Fatalf("total action count = %d, want %d", got, want)
	}
	if center.Counts.NeedsDecision < 3 {
		t.Fatalf("needs_decision = %d, want at least 3", center.Counts.NeedsDecision)
	}
	if center.Counts.NeedsRepair < 2 {
		t.Fatalf("needs_repair = %d, want at least 2", center.Counts.NeedsRepair)
	}

	byType := map[string]SubscriptionAction{}
	for _, item := range center.Items {
		byType[item.Type] = item
	}

	if got := byType[actionTypeManualRenewalDue]; got.SubscriptionID != manualRenew.ID || got.DaysUntil == nil || *got.DaysUntil != 2 {
		t.Fatalf("manual renewal action = %+v, want subscription %d due in 2 days", got, manualRenew.ID)
	}
	if got := byType[actionTypeUpcomingRenewal]; got.SubscriptionID != autoRenew.ID || got.DueDate == nil || *got.DueDate != "2026-03-05" {
		t.Fatalf("upcoming renewal action = %+v, want Video Pro due 2026-03-05", got)
	}
	if got := byType[actionTypeEndingSoon]; got.SubscriptionID != ending.ID || got.DueDate == nil || *got.DueDate != "2026-03-20" {
		t.Fatalf("ending action = %+v, want Old Tool ending 2026-03-20", got)
	}
	if got := byType[actionTypeMissingNextBilling]; got.SubscriptionID != missingDate.ID || !got.NeedsRepair {
		t.Fatalf("missing date action = %+v, want repair item for Broken Plan", got)
	}
	if got := byType[actionTypeNotificationFailed]; got.SubscriptionID != autoRenew.ID || got.NotificationChannel != "webhook" || got.NotificationError != "timeout" {
		t.Fatalf("notification failure action = %+v, want webhook timeout for Video Pro", got)
	}
	if got := byType[actionTypePriceIncrease]; got.SubscriptionID != autoRenew.ID || got.DeltaMonthlyAmount == nil {
		t.Fatalf("price increase action = %+v, want delta for Video Pro", got)
	}
}

func TestSnoozeActionHidesMatchingItem(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          12,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-05",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	key := subscriptionActionKey(sub.ID, actionTypeUpcomingRenewal, "2026-03-05")
	if _, err := service.SnoozeAction(user.ID, SnoozeSubscriptionActionInput{Key: key, Days: 7}); err != nil {
		t.Fatalf("SnoozeAction() error = %v", err)
	}

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}
	if got, want := center.Counts.Total, 0; got != want {
		t.Fatalf("visible action count = %d, want %d", got, want)
	}
	if got, want := center.Counts.Snoozed, 1; got != want {
		t.Fatalf("snoozed action count = %d, want %d", got, want)
	}
}

func TestActionCenterSuppressesRecoveredNotificationFailures(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          12,
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

	if err := db.Create(&model.NotificationLog{
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelType:    "webhook",
		NotifyDate:     mustDate(t, "2026-03-04"),
		Status:         notificationLogStatusFailed,
		Error:          "timeout",
		SentAt:         time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create failed notification log failed: %v", err)
	}
	if err := db.Create(&model.NotificationLog{
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelType:    "webhook",
		NotifyDate:     mustDate(t, "2026-03-04"),
		Status:         notificationLogStatusSent,
		SentAt:         time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create recovered notification log failed: %v", err)
	}

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	for _, item := range center.Items {
		if item.Type == actionTypeNotificationFailed {
			t.Fatalf("notification failure action = %+v, want recovered failure suppressed", item)
		}
	}
}

func TestActionCenterSuppressesOlderPriceIncreasesAfterReduction(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          12,
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

	increasedAmount := 18.0
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &increasedAmount}); err != nil {
		t.Fatalf("increase subscription amount failed: %v", err)
	}
	reducedAmount := 10.0
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &reducedAmount}); err != nil {
		t.Fatalf("reduce subscription amount failed: %v", err)
	}

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	for _, item := range center.Items {
		if item.Type == actionTypePriceIncrease {
			t.Fatalf("price increase action = %+v, want latest reduction to suppress older increase", item)
		}
	}
}
