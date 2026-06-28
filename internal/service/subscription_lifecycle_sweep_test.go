package service

import (
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

func createNamedRolloverUser(t *testing.T, db *gorm.DB, username, email string) model.User {
	t.Helper()
	user := model.User{Username: username, Email: email, Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user %s failed: %v", username, err)
	}
	return user
}

func countSubscriptionWrites(t *testing.T, db *gorm.DB, counter *int) {
	t.Helper()
	increment := func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == "subscriptions" {
			*counter++
		}
	}
	if err := db.Callback().Update().After("gorm:update").Register("test:count_sub_updates", increment); err != nil {
		t.Fatalf("register update counter failed: %v", err)
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:count_sub_creates", increment); err != nil {
		t.Fatalf("register create counter failed: %v", err)
	}
}

// TestReconcileDueLifecyclesAdvancesAndEndsAcrossUsers verifies the sweep drives
// lifecycle for every user: it rolls an overdue auto-renew forward and ends an
// overdue manual-renew, each owned by a different user.
func TestReconcileDueLifecyclesAdvancesAndEndsAcrossUsers(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	service := NewSubscriptionService(db)

	autoUser := createNamedRolloverUser(t, db, "auto-user", "auto@example.com")
	manualUser := createNamedRolloverUser(t, db, "manual-user", "manual@example.com")

	intervalCount := 1
	overdueAuto := now.AddDate(0, 0, -3)
	autoSub, err := service.Create(autoUser.ID, CreateSubscriptionInput{
		Name:            "Auto overdue",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdueAuto.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create auto-renew subscription failed: %v", err)
	}

	overdueManual := now.AddDate(0, 0, -2)
	manualSub, err := service.Create(manualUser.ID, CreateSubscriptionInput{
		Name:            "Manual overdue",
		Amount:          12,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdueManual.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create manual-renew subscription failed: %v", err)
	}

	if err := service.reconcileDueLifecycles(now); err != nil {
		t.Fatalf("reconcileDueLifecycles() error = %v", err)
	}

	var advanced model.Subscription
	if err := db.First(&advanced, autoSub.ID).Error; err != nil {
		t.Fatalf("reload auto-renew subscription failed: %v", err)
	}
	if advanced.Status != subscriptionStatusActive {
		t.Fatalf("auto-renew status = %q, want %q", advanced.Status, subscriptionStatusActive)
	}
	if advanced.NextBillingDate == nil || !advanced.NextBillingDate.After(normalizeDateUTC(overdueAuto)) {
		t.Fatalf("auto-renew next_billing_date = %v, want rolled forward past %s", advanced.NextBillingDate, overdueAuto.Format("2006-01-02"))
	}

	var ended model.Subscription
	if err := db.First(&ended, manualSub.ID).Error; err != nil {
		t.Fatalf("reload manual-renew subscription failed: %v", err)
	}
	if ended.Status != subscriptionStatusEnded {
		t.Fatalf("manual-renew status = %q, want %q", ended.Status, subscriptionStatusEnded)
	}
}

// TestReconcileDueLifecyclesIsIdempotent proves a second sweep over an
// already-reconciled state issues no further subscription writes — the property
// that lets the read path stop writing once the sweep has run.
func TestReconcileDueLifecyclesIsIdempotent(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	service := NewSubscriptionService(db)
	user := createSubscriptionRolloverTestUser(t, db)

	intervalCount := 1
	overdue := now.AddDate(0, 0, -3)
	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Auto overdue",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdue.Format("2006-01-02"),
	}); err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	if err := service.reconcileDueLifecycles(now); err != nil {
		t.Fatalf("first reconcileDueLifecycles() error = %v", err)
	}

	var writes int
	countSubscriptionWrites(t, db, &writes)

	if err := service.reconcileDueLifecycles(now); err != nil {
		t.Fatalf("second reconcileDueLifecycles() error = %v", err)
	}

	if writes != 0 {
		t.Fatalf("second sweep issued %d subscription writes, want 0 (not idempotent)", writes)
	}
}

// TestReconcileDueLifecyclesRespectsLease confirms the sweep is gated by the
// background-task lease: an instance that cannot claim the lease performs no
// work, while the lease owner processes due subscriptions.
func TestReconcileDueLifecyclesRespectsLease(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	if err := db.AutoMigrate(&model.BackgroundTaskLease{}); err != nil {
		t.Fatalf("migrate background task leases failed: %v", err)
	}
	service := NewSubscriptionService(db)
	user := createSubscriptionRolloverTestUser(t, db)

	// A live lease held by another owner.
	if err := db.Create(&model.BackgroundTaskLease{
		TaskKey:     subscriptionLifecycleSweepTaskKey,
		OwnerID:     "other-instance",
		LeaseUntil:  pkg.NowUTC().Add(time.Hour),
		HeartbeatAt: pkg.NowUTC(),
	}).Error; err != nil {
		t.Fatalf("seed lease failed: %v", err)
	}

	intervalCount := 1
	overdue := now.AddDate(0, 0, -2)
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual overdue",
		Amount:          12,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdue.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	// A different instance cannot claim the lease, so it must not reconcile.
	if err := service.ReconcileDueLifecycles("this-instance"); err != nil {
		t.Fatalf("ReconcileDueLifecycles(blocked) error = %v", err)
	}
	var untouched model.Subscription
	if err := db.First(&untouched, sub.ID).Error; err != nil {
		t.Fatalf("reload subscription failed: %v", err)
	}
	if untouched.Status != subscriptionStatusActive {
		t.Fatalf("status = %q, want %q while lease held elsewhere", untouched.Status, subscriptionStatusActive)
	}

	// The lease owner re-running it does the work.
	if err := service.ReconcileDueLifecycles("other-instance"); err != nil {
		t.Fatalf("ReconcileDueLifecycles(owner) error = %v", err)
	}
	var ended model.Subscription
	if err := db.First(&ended, sub.ID).Error; err != nil {
		t.Fatalf("reload subscription failed: %v", err)
	}
	if ended.Status != subscriptionStatusEnded {
		t.Fatalf("status = %q, want %q after lease owner ran", ended.Status, subscriptionStatusEnded)
	}
}
