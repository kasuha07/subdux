package service

import (
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

// countSubscriptionTableWrites registers callbacks that increment counter for
// every insert/update against the subscriptions table, so a test can assert a
// code path is write-free.
func countSubscriptionTableWrites(t *testing.T, db *gorm.DB, counter *int) {
	t.Helper()
	increment := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "subscriptions" {
			*counter++
		}
	}
	if err := db.Callback().Update().After("gorm:update").Register("test:readpath_update", increment); err != nil {
		t.Fatalf("register update counter failed: %v", err)
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:readpath_create", increment); err != nil {
		t.Fatalf("register create counter failed: %v", err)
	}
}

// TestReadPathsDoNotWriteSubscriptions verifies the core invariant of this
// round: List, the dashboard summary, the action center, and the analytics
// report all advance lifecycle in memory and never write the subscriptions
// table, even when subscriptions are overdue.
func TestReadPathsDoNotWriteSubscriptions(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	if err := db.AutoMigrate(&model.SubscriptionActionSnooze{}); err != nil {
		t.Fatalf("migrate action snoozes failed: %v", err)
	}
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

	today := setSubscriptionRolloverTestNow(t)
	intervalCount := 1

	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Overdue auto",
		Amount:          10,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: today.AddDate(0, 0, -5).Format("2006-01-02"),
	}); err != nil {
		t.Fatalf("create auto subscription failed: %v", err)
	}
	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Overdue manual",
		Amount:          12,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: today.AddDate(0, 0, -2).Format("2006-01-02"),
	}); err != nil {
		t.Fatalf("create manual subscription failed: %v", err)
	}

	var writes int
	countSubscriptionTableWrites(t, db, &writes)

	if _, err := service.List(user.ID); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.GetDashboardSummary(user.ID, "USD", nil); err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}
	if _, err := service.GetActionCenter(user.ID); err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}
	if _, err := service.GetAnalyticsReport(user.ID, "USD", nil); err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if writes != 0 {
		t.Fatalf("read paths issued %d subscription writes, want 0", writes)
	}
}

// TestUpdatePersistsDueLifecycleBeforeMutating confirms write-path maintenance:
// updating an overdue auto-renew subscription (even an unrelated field) first
// persists the rolled-forward billing date, so the mutation operates on current
// state rather than resurrecting a stale schedule.
func TestUpdatePersistsDueLifecycleBeforeMutating(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

	today := setSubscriptionRolloverTestNow(t)
	overdue := today.AddDate(0, 0, -10)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Overdue auto",
		Amount:          10,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitWeek,
		NextBillingDate: overdue.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	newName := "Renamed"
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{Name: &newName}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.Name != newName {
		t.Fatalf("name = %q, want %q", stored.Name, newName)
	}
	expected := nextIntervalOccurrence(overdue, today, intervalCount, intervalUnitWeek)
	if stored.NextBillingDate == nil || stored.NextBillingDate.Format("2006-01-02") != expected.Format("2006-01-02") {
		t.Fatalf("stored next_billing_date = %v, want rolled forward to %s", stored.NextBillingDate, expected.Format("2006-01-02"))
	}
}

// TestReconcileUserLifecyclePersists confirms the explicit manual repair entry
// writes due transitions immediately rather than waiting for the sweep.
func TestReconcileUserLifecyclePersists(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

	today := setSubscriptionRolloverTestNow(t)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Overdue manual",
		Amount:          12,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: today.AddDate(0, 0, -2).Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	if err := service.ReconcileUserLifecycle(user.ID); err != nil {
		t.Fatalf("ReconcileUserLifecycle() error = %v", err)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.Status != subscriptionStatusEnded {
		t.Fatalf("stored status = %q, want %q after manual reconcile", stored.Status, subscriptionStatusEnded)
	}
}
