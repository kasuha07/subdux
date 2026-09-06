package subscription

import (
	"sync"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

// TestSweepDoesNotOverwriteConcurrentUserEdit proves the background lifecycle
// sweep cannot clobber a user edit that lands between the sweep's read and its
// write. The GORM query callback freezes the sweep right after it loads the
// user's subscriptions; the user then commits a new next_billing_date; the
// sweep's gated write must match no rows and leave the user's value alone.
func TestSweepDoesNotOverwriteConcurrentUserEdit(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	service := NewService(db)
	user := createSubscriptionRolloverTestUser(t, db)

	intervalCount := 1
	overdue := now.AddDate(0, 0, -9)
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Sweep race",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdue.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	// Freeze the sweep after it loads the subscription list, before it writes.
	sweepLoaded := make(chan struct{})
	gate := make(chan struct{})
	var once sync.Once
	if err := db.Callback().Query().After("gorm:query").Register("test:pause_sweep_after_load", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "subscriptions" {
			return
		}
		if _, isSlice := tx.Statement.Dest.(*[]model.Subscription); !isSlice {
			return
		}
		once.Do(func() {
			close(sweepLoaded)
			<-gate
		})
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}

	var sweepErr error
	sweepDone := make(chan struct{})
	go func() {
		defer close(sweepDone)
		sweepErr = reconcileSubscriptionLifecycleForUser(db, user.ID, now)
	}()

	<-sweepLoaded

	// The user commits a new billing date while the sweep holds the stale snapshot.
	newDate := "2027-03-06"
	updated, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{NextBillingDate: &newDate})
	if err != nil {
		t.Fatalf("user update failed: %v", err)
	}
	if updated.NextBillingDate == nil || updated.NextBillingDate.Format("2006-01-02") != newDate {
		t.Fatalf("user update returned next_billing_date = %v, want %s", updated.NextBillingDate, newDate)
	}

	close(gate)
	<-sweepDone
	if sweepErr != nil {
		t.Fatalf("reconcileSubscriptionLifecycleForUser() error = %v", sweepErr)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("reload subscription failed: %v", err)
	}
	if stored.NextBillingDate == nil || stored.NextBillingDate.Format("2006-01-02") != newDate {
		t.Fatalf("stored next_billing_date = %v after sweep, want user's %s preserved",
			stored.NextBillingDate, newDate)
	}
	if stored.Status != subscriptionStatusActive {
		t.Fatalf("stored status = %q, want %q (sweep must not end an edited row)", stored.Status, subscriptionStatusActive)
	}
}

// TestSweepDoesNotOverwriteConcurrentBillingRuleEdit proves the gate covers
// the billing-rule inputs of the advance, not just the lifecycle columns: the
// user switches the recurrence from monthly to yearly while the sweep holds a
// stale snapshot with an identical next_billing_date, and the sweep's write
// must still lose the race instead of rolling the date by the old monthly rule.
func TestSweepDoesNotOverwriteConcurrentBillingRuleEdit(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	service := NewService(db)
	user := createSubscriptionRolloverTestUser(t, db)

	intervalCount := 1
	// Anchor the overdue date on a day-of-month that exists in every month so
	// the monthly and yearly rules both keep the day stable.
	base := time.Date(now.Year(), now.Month(), 5, 0, 0, 0, 0, time.UTC)
	for !base.Before(now) {
		base = base.AddDate(0, -1, 0)
	}
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Rule race",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: base.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	// Freeze the sweep after it loads the subscription list, before it writes.
	sweepLoaded := make(chan struct{})
	gate := make(chan struct{})
	var once sync.Once
	if err := db.Callback().Query().After("gorm:query").Register("test:pause_rule_race", func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "subscriptions" {
			return
		}
		if _, isSlice := tx.Statement.Dest.(*[]model.Subscription); !isSlice {
			return
		}
		once.Do(func() {
			close(sweepLoaded)
			<-gate
		})
	}); err != nil {
		t.Fatalf("register query callback failed: %v", err)
	}

	var sweepErr error
	sweepDone := make(chan struct{})
	go func() {
		defer close(sweepDone)
		sweepErr = reconcileSubscriptionLifecycleForUser(db, user.ID, now)
	}()

	<-sweepLoaded

	// The user switches monthly -> yearly and re-submits the same overdue
	// next_billing_date. Update's own reconcile first rolls the date forward
	// under the old monthly rule, then the user's explicit values win: the row
	// ends up with the yearly rule, the same overdue date — and the gate
	// columns (status, renewal_mode, next_billing_date, ends_at) all identical
	// to the sweep's snapshot. Only the computation inputs (the billing rule
	// fields) changed.
	newUnit := intervalUnitYear
	sameDate := base.Format("2006-01-02")
	updated, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		IntervalUnit:    &newUnit,
		NextBillingDate: &sameDate,
	})
	if err != nil {
		t.Fatalf("user update failed: %v", err)
	}
	if updated.IntervalUnit != intervalUnitYear {
		t.Fatalf("user update interval_unit = %q, want %q", updated.IntervalUnit, intervalUnitYear)
	}
	if updated.NextBillingDate == nil || updated.NextBillingDate.Format("2006-01-02") != sameDate {
		t.Fatalf("user update next_billing_date = %v, want user-submitted %s", updated.NextBillingDate, sameDate)
	}

	close(gate)
	<-sweepDone
	if sweepErr != nil {
		t.Fatalf("reconcileSubscriptionLifecycleForUser() error = %v", sweepErr)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("reload subscription failed: %v", err)
	}
	// The stale snapshot would roll the date by the old monthly rule
	// (base + 1 month); the user's yearly rule and explicit date must survive.
	// A gate comparing only the lifecycle columns would match here (they are
	// all unchanged by the edit) and let the stale write through; only a
	// row-version gate rejects it.
	if stored.IntervalUnit != intervalUnitYear {
		t.Fatalf("stored interval_unit = %q, want %q (sweep clobbered the rule edit)", stored.IntervalUnit, intervalUnitYear)
	}
	if stored.NextBillingDate == nil || stored.NextBillingDate.Format("2006-01-02") != sameDate {
		t.Fatalf("stored next_billing_date = %v, want user's %s preserved under the yearly rule",
			stored.NextBillingDate, sameDate)
	}
}

// TestSweepStillAdvancesUnchangedRows proves the snapshot gate does not break
// the sweep's normal job: with no concurrent edit, an overdue auto-renew
// subscription still rolls forward and an overdue manual-renew still ends.
func TestSweepStillAdvancesUnchangedRows(t *testing.T) {
	now := setSubscriptionRolloverTestNow(t)

	db := newSubscriptionRolloverTestDB(t)
	service := NewService(db)
	user := createSubscriptionRolloverTestUser(t, db)

	intervalCount := 1
	autoSub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Auto overdue",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: now.AddDate(0, 0, -3).Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create auto subscription failed: %v", err)
	}
	manualSub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual overdue",
		Amount:          12,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: now.AddDate(0, 0, -2).Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("create manual subscription failed: %v", err)
	}

	if err := reconcileSubscriptionLifecycleForUser(db, user.ID, now); err != nil {
		t.Fatalf("reconcileSubscriptionLifecycleForUser() error = %v", err)
	}

	var advanced model.Subscription
	if err := db.First(&advanced, autoSub.ID).Error; err != nil {
		t.Fatalf("reload auto subscription failed: %v", err)
	}
	expected := now.AddDate(0, 1, -3).Format("2006-01-02") // rolled forward to the first occurrence on or after today
	if advanced.NextBillingDate == nil || advanced.NextBillingDate.Format("2006-01-02") != expected {
		t.Fatalf("auto next_billing_date = %v, want %s", advanced.NextBillingDate, expected)
	}

	var ended model.Subscription
	if err := db.First(&ended, manualSub.ID).Error; err != nil {
		t.Fatalf("reload manual subscription failed: %v", err)
	}
	if ended.Status != subscriptionStatusEnded {
		t.Fatalf("manual status = %q, want %q", ended.Status, subscriptionStatusEnded)
	}
}
