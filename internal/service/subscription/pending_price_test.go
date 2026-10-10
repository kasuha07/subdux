package subscription

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

func pendingPriceTestSubscription(t *testing.T, renewalMode string) model.Subscription {
	t.Helper()
	intervalCount := 1
	nextBilling := mustDate(t, "2026-04-01")
	pendingAmount := 30.0
	pendingFrom := mustDate(t, "2026-06-01")
	return model.Subscription{
		ID:              1,
		UserID:          1,
		Name:            "Intro",
		Amount:          1,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalMode,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBilling,
		PendingAmount:   &pendingAmount,
		PendingFrom:     &pendingFrom,
	}
}

func TestSubscriptionChargeAmountOnHonorsPendingPrice(t *testing.T) {
	sub := pendingPriceTestSubscription(t, renewalModeAutoRenew)

	for _, tc := range []struct {
		date string
		want float64
	}{
		{"2026-04-01", 1},
		{"2026-05-01", 1},
		{"2026-05-31", 1},
		{"2026-06-01", 30},
		{"2026-07-01", 30},
	} {
		if got := subscriptionChargeAmountOn(sub, mustDate(t, tc.date)); got != tc.want {
			t.Fatalf("charge on %s = %v, want %v", tc.date, got, tc.want)
		}
	}

	sub.PendingFrom = nil
	if got := subscriptionChargeAmountOn(sub, mustDate(t, "2026-07-01")); got != 1 {
		t.Fatalf("charge without complete pending price = %v, want current amount", got)
	}
}

func TestApplyDuePendingPrice(t *testing.T) {
	t.Run("before effective date keeps current price", func(t *testing.T) {
		sub := pendingPriceTestSubscription(t, renewalModeAutoRenew)
		if applyDuePendingPrice(&sub, mustDate(t, "2026-05-31")) {
			t.Fatal("pending price applied before its effective date")
		}
		if sub.Amount != 1 || sub.PendingAmount == nil {
			t.Fatalf("subscription changed before effective date: %+v", sub)
		}
	})

	t.Run("effective date switches price and clears schedule", func(t *testing.T) {
		sub := pendingPriceTestSubscription(t, renewalModeManualRenew)
		if !applyDuePendingPrice(&sub, mustDate(t, "2026-06-01")) {
			t.Fatal("pending price was not applied on its effective date")
		}
		if sub.Amount != 30 || sub.PendingAmount != nil || sub.PendingFrom != nil {
			t.Fatalf("subscription after apply = amount %v pending %v/%v", sub.Amount, sub.PendingAmount, sub.PendingFrom)
		}
	})

	t.Run("canceling subscription never reaches scheduled price", func(t *testing.T) {
		sub := pendingPriceTestSubscription(t, renewalModeCancelAtPeriodEnd)
		if applyDuePendingPrice(&sub, mustDate(t, "2026-07-01")) {
			t.Fatal("canceling subscription switched to its scheduled price")
		}
	})
}

func TestNormalizePendingPrice(t *testing.T) {
	now := mustDate(t, "2026-03-15")
	schedule := billingDraft{BillingType: billingTypeRecurring, RecurrenceType: recurrenceTypeInterval, IntervalCount: func() *int { v := 1; return &v }(), IntervalUnit: intervalUnitMonth}
	amount := 30.0
	tomorrow := mustDate(t, "2026-03-16")
	today := mustDate(t, "2026-03-15")
	negative := -1.0

	if gotAmount, gotFrom, err := normalizePendingPrice(nil, nil, "USD", schedule, now); err != nil || gotAmount != nil || gotFrom != nil {
		t.Fatalf("empty pending price = %v %v %v, want nil", gotAmount, gotFrom, err)
	}
	if _, _, err := normalizePendingPrice(&amount, nil, "USD", schedule, now); !errors.Is(err, ErrPendingPriceIncomplete) {
		t.Fatalf("amount without date error = %v", err)
	}
	if _, _, err := normalizePendingPrice(nil, &tomorrow, "USD", schedule, now); !errors.Is(err, ErrPendingPriceIncomplete) {
		t.Fatalf("date without amount error = %v", err)
	}
	if _, _, err := normalizePendingPrice(&amount, &today, "USD", schedule, now); !errors.Is(err, ErrPendingFromMustBeFuture) {
		t.Fatalf("effective today error = %v", err)
	}
	if _, _, err := normalizePendingPrice(&negative, &tomorrow, "USD", schedule, now); !errors.Is(err, ErrAmountMustNotBeNegative) {
		t.Fatalf("negative pending amount error = %v", err)
	}
	gotAmount, gotFrom, err := normalizePendingPrice(&amount, &tomorrow, "USD", schedule, now)
	if err != nil || gotAmount == nil || *gotAmount != 30 || gotFrom == nil || !gotFrom.Equal(tomorrow) {
		t.Fatalf("valid pending price = %v %v %v", gotAmount, gotFrom, err)
	}
}

func TestPendingPriceTrialLifecycleEndToEnd(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	svc := NewService(db)
	setSubscriptionRolloverTestNow(t) // 2026-03-15

	intervalCount := 1
	pendingAmount := 30.0
	created, err := svc.Create(user.ID, CreateSubscriptionInput{
		Name:            "Trial",
		Amount:          0,
		Currency:        "USD",
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-22",
		PendingAmount:   &pendingAmount,
		PendingFrom:     "2026-03-22",
	})
	if err != nil {
		t.Fatalf("create trial: %v", err)
	}
	if created.PendingAmount == nil || *created.PendingAmount != 30 {
		t.Fatalf("created pending amount = %v", created.PendingAmount)
	}

	// The upcoming charge is already billed at the scheduled price.
	charges := subscriptionDetailUpcomingCharges(*created, 2, mustDate(t, "2026-03-15"))
	if len(charges) != 2 || charges[0].Amount != 30 || charges[1].Amount != 30 {
		t.Fatalf("upcoming charges = %+v, want both at the scheduled price", charges)
	}
	actions := subscriptionScheduleActions(*created, mustDate(t, "2026-03-15"), mustDate(t, "2026-04-14"))
	if len(actions) != 1 || actions[0].Type != actionTypePendingPrice || actions[0].Amount != 30 ||
		actions[0].PreviousAmount == nil || *actions[0].PreviousAmount != 0 || !actions[0].NeedsDecision {
		t.Fatalf("actions = %+v, want one pending-price decision 0 -> 30", actions)
	}

	// Crossing the effective date persists the switch with a system event.
	restore := pkg.SetNowForTest(mustDate(t, "2026-03-22"))
	defer restore()
	if err := svc.ReconcileUserLifecycle(user.ID); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var stored model.Subscription
	if err := db.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Amount != 30 || stored.PendingAmount != nil || stored.PendingFrom != nil {
		t.Fatalf("stored after effective date = amount %v pending %v/%v", stored.Amount, stored.PendingAmount, stored.PendingFrom)
	}
	var events []model.SubscriptionEvent
	if err := db.Where("subscription_id = ? AND type = ?", created.ID, subscriptionEventPendingPriceApplied).Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ActorUserID != nil ||
		events[0].PreviousAmount == nil || *events[0].PreviousAmount != 0 ||
		events[0].NewAmount == nil || *events[0].NewAmount != 30 {
		t.Fatalf("pending price events = %+v, want one system event 0 -> 30", events)
	}

	// The expected switch is not reported again as a price increase.
	increases, err := svc.priceIncreaseActions(user.ID, mustDate(t, "2026-03-22"))
	if err != nil {
		t.Fatal(err)
	}
	if len(increases) != 0 {
		t.Fatalf("price increase actions = %+v, want none for a scheduled price", increases)
	}

	// Reconciling again is a no-op.
	if err := svc.ReconcileUserLifecycle(user.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.SubscriptionEvent{}).Where("subscription_id = ? AND type = ?", created.ID, subscriptionEventPendingPriceApplied).Count(&count)
	if count != 1 {
		t.Fatalf("pending price events after second reconcile = %d, want 1", count)
	}
}

func TestPendingPriceSplitsForecastAndDueThisMonth(t *testing.T) {
	sub := pendingPriceTestSubscription(t, renewalModeAutoRenew)
	amounts, err := convertSubscriptionChargeAmounts(sub, "USD", nil)
	if err != nil {
		t.Fatal(err)
	}
	dates := subscriptionChargeDatesInRange(sub, mustDate(t, "2026-04-01"), mustDate(t, "2026-08-01"))
	if len(dates) != 4 {
		t.Fatalf("charge dates = %v, want 4", dates)
	}
	total, err := amounts.total(dates, "USD")
	if err != nil {
		t.Fatal(err)
	}
	// April and May at the introductory price, June and July at the regular price.
	if total != 62 {
		t.Fatalf("total = %v, want 62", total)
	}

	summary, err := computeDashboardSummary([]model.Subscription{sub}, "USD", nil, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if summary.DueThisMonth != 30 {
		t.Fatalf("due this month = %v, want the scheduled price", summary.DueThisMonth)
	}
}

func TestUpdateSetsAndClearsPendingPrice(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	svc := NewService(db)
	setSubscriptionRolloverTestNow(t) // 2026-03-15

	intervalCount := 1
	created, err := svc.Create(user.ID, CreateSubscriptionInput{
		Name:            "Promo",
		Amount:          10,
		Currency:        "USD",
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	amount := 1.0
	regular := 10.0
	pendingFrom := "2026-07-01"
	updated, err := svc.Update(user.ID, created.ID, UpdateSubscriptionInput{
		Revision:        created.Revision,
		Amount:          &amount,
		PendingAmount:   &regular,
		PendingFrom:     &pendingFrom,
		PendingPriceSet: true,
	})
	if err != nil {
		t.Fatalf("set pending price: %v", err)
	}
	if updated.Amount != 1 || updated.PendingAmount == nil || *updated.PendingAmount != 10 || updated.PendingFrom == nil {
		t.Fatalf("updated = amount %v pending %v/%v", updated.Amount, updated.PendingAmount, updated.PendingFrom)
	}

	// Omitting the pending fields keeps the schedule.
	name := "Promo renamed"
	updated, err = svc.Update(user.ID, created.ID, UpdateSubscriptionInput{Revision: updated.Revision, Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if updated.PendingAmount == nil {
		t.Fatal("unrelated update cleared the pending price")
	}

	// Ending the promotion early: regular price now, schedule cleared.
	updated, err = svc.Update(user.ID, created.ID, UpdateSubscriptionInput{
		Revision:        updated.Revision,
		Amount:          &regular,
		PendingPriceSet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Amount != 10 || updated.PendingAmount != nil || updated.PendingFrom != nil {
		t.Fatalf("after clearing = amount %v pending %v/%v", updated.Amount, updated.PendingAmount, updated.PendingFrom)
	}

	past := "2026-03-15"
	if _, err := svc.Update(user.ID, created.ID, UpdateSubscriptionInput{
		Revision:        updated.Revision,
		PendingAmount:   &regular,
		PendingFrom:     &past,
		PendingPriceSet: true,
	}); !errors.Is(err, ErrPendingFromMustBeFuture) {
		t.Fatalf("pending from today error = %v", err)
	}
}
