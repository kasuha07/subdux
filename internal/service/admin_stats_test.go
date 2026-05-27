package service

import "testing"

func TestAdminStatsExcludesCancelAtPeriodEndFromMonthlySpend(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	subscriptionService := NewSubscriptionService(db)
	adminService := NewAdminService(db)

	intervalCount := 1
	if _, err := subscriptionService.Create(user.ID, CreateSubscriptionInput{
		Name:            "Active monthly",
		Amount:          10,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-20",
	}); err != nil {
		t.Fatalf("create active subscription failed: %v", err)
	}

	if _, err := subscriptionService.Create(user.ID, CreateSubscriptionInput{
		Name:            "Ending monthly",
		Amount:          99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-22",
	}); err != nil {
		t.Fatalf("create ending subscription failed: %v", err)
	}

	stats, err := adminService.GetStats()
	if err != nil {
		t.Fatalf("GetStats() error = %v", err)
	}

	if got, want := stats.TotalSubscriptions, int64(2); got != want {
		t.Fatalf("total_subscriptions = %d, want %d", got, want)
	}
	assertFloatEqual(t, stats.TotalMonthlySpend, 10, "total_monthly_spend")
}
