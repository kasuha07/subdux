package service

import (
	"math"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestGetAnalyticsReportAggregatesSubscriptionSpend(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	categoryService := NewCategoryService(db)
	videoCategory, err := categoryService.Create(user.ID, CreateCategoryInput{Name: "Video"})
	if err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	paymentMethodService := NewPaymentMethodService(db)
	cardMethod, err := paymentMethodService.Create(user.ID, CreatePaymentMethodInput{Name: "Card"})
	if err != nil {
		t.Fatalf("create payment method failed: %v", err)
	}

	monthly := 1
	yearly := 1
	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          12,
		Icon:            "custom:netflix",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
		CategoryID:      &videoCategory.ID,
		PaymentMethodID: &cardMethod.ID,
	}); err != nil {
		t.Fatalf("create monthly subscription failed: %v", err)
	}

	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Cloud Annual",
		Amount:          120,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &yearly,
		IntervalUnit:    intervalUnitYear,
		NextBillingDate: "2026-04-10",
		Category:        "Cloud",
	}); err != nil {
		t.Fatalf("create yearly subscription failed: %v", err)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if got, want := report.KPIs.ActiveCount, int64(2); got != want {
		t.Fatalf("active_count = %d, want %d", got, want)
	}
	assertFloatEqual(t, report.KPIs.TotalMonthly, 22, "total_monthly")
	assertFloatEqual(t, report.KPIs.CommittedMonthly, 12, "committed_monthly")
	assertFloatEqual(t, report.KPIs.DueThisMonth, 12, "due_this_month")
	assertFloatEqual(t, report.KPIs.DueNext30Days, 12, "due_next_30_days")

	if got, want := len(report.MonthlyForecast), 12; got != want {
		t.Fatalf("monthly_forecast length = %d, want %d", got, want)
	}
	if got, want := report.MonthlyForecast[0].Month, "2026-03"; got != want {
		t.Fatalf("first forecast month = %q, want %q", got, want)
	}
	assertFloatEqual(t, report.MonthlyForecast[0].AmountDue, 12, "march amount_due")
	assertFloatEqual(t, report.MonthlyForecast[1].AmountDue, 132, "april amount_due")
	assertFloatEqual(t, report.MonthlyForecast[2].AmountDue, 12, "may amount_due")

	if got, want := report.CategoryBreakdown[0].Label, "Video"; got != want {
		t.Fatalf("top category label = %q, want %q", got, want)
	}
	assertFloatEqual(t, report.CategoryBreakdown[0].MonthlyAmount, 12, "top category monthly_amount")

	if got, want := report.PaymentMethodBreakdown[0].Label, "Card"; got != want {
		t.Fatalf("top payment method label = %q, want %q", got, want)
	}

	if got, want := report.TopSubscriptions[0].Name, "Video Pro"; got != want {
		t.Fatalf("top subscription = %q, want %q", got, want)
	}
	if got, want := report.TopSubscriptions[0].Icon, "custom:netflix"; got != want {
		t.Fatalf("top subscription icon = %q, want %q", got, want)
	}
	if got, want := report.UpcomingRenewals[0].Name, "Video Pro"; got != want {
		t.Fatalf("upcoming renewal = %q, want %q", got, want)
	}
	if got, want := report.UpcomingRenewals[0].Icon, "custom:netflix"; got != want {
		t.Fatalf("upcoming renewal icon = %q, want %q", got, want)
	}
}

func TestGetAnalyticsReportExcludesCancelAtPeriodEndFromSpendAndForecast(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Active Video",
		Amount:          12,
		Icon:            "custom:netflix",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
		Category:        "Video",
	}); err != nil {
		t.Fatalf("create active subscription failed: %v", err)
	}

	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Ending Video",
		Amount:          99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-20",
		Category:        "Ending",
	}); err != nil {
		t.Fatalf("create ending subscription failed: %v", err)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if got, want := report.KPIs.ActiveCount, int64(2); got != want {
		t.Fatalf("active_count = %d, want %d", got, want)
	}
	if got, want := report.KPIs.CancelingCount, int64(1); got != want {
		t.Fatalf("canceling_count = %d, want %d", got, want)
	}
	assertFloatEqual(t, report.KPIs.TotalMonthly, 12, "total_monthly")
	assertFloatEqual(t, report.KPIs.TotalYearly, 144, "total_yearly")
	assertFloatEqual(t, report.KPIs.DueThisMonth, 12, "due_this_month")
	assertFloatEqual(t, report.KPIs.DueNext30Days, 12, "due_next_30_days")

	if got, want := len(report.UpcomingRenewals), 1; got != want {
		t.Fatalf("upcoming_renewals length = %d, want %d", got, want)
	}
	if got, want := report.UpcomingRenewals[0].Name, "Active Video"; got != want {
		t.Fatalf("upcoming renewal = %q, want %q", got, want)
	}
	assertFloatEqual(t, report.MonthlyForecast[0].AmountDue, 12, "march amount_due")

	if got, want := len(report.TopSubscriptions), 1; got != want {
		t.Fatalf("top_subscriptions length = %d, want %d", got, want)
	}
	if got, want := report.TopSubscriptions[0].Name, "Active Video"; got != want {
		t.Fatalf("top subscription = %q, want %q", got, want)
	}
	for _, item := range report.CategoryBreakdown {
		if item.Label == "Ending" {
			t.Fatal("cancel-at-period-end subscription should not appear in category breakdown")
		}
	}
	for _, item := range report.RenewalModeBreakdown {
		if item.Key == renewalModeCancelAtPeriodEnd {
			t.Fatal("cancel-at-period-end subscription should not appear in spend breakdown")
		}
	}
}

func TestSubscriptionEventsTrackChangesForAnalytics(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Music",
		Amount:          10,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	var createdEvent model.SubscriptionEvent
	if err := db.Where("subscription_id = ? AND type = ?", sub.ID, subscriptionEventCreated).First(&createdEvent).Error; err != nil {
		t.Fatalf("find created event failed: %v", err)
	}
	if createdEvent.NewAmount == nil || *createdEvent.NewAmount != 10 {
		t.Fatalf("created event new amount = %v, want 10", createdEvent.NewAmount)
	}
	if got, want := decodeSubscriptionEventFields(createdEvent.ChangedFields), []string{"created"}; !stringSlicesEqual(got, want) {
		t.Fatalf("created event changed fields = %v, want %v", got, want)
	}

	updatedAmount := 15.0
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		Amount: &updatedAmount,
	}); err != nil {
		t.Fatalf("update subscription failed: %v", err)
	}

	var updatedEvent model.SubscriptionEvent
	if err := db.Where("subscription_id = ? AND type = ?", sub.ID, subscriptionEventUpdated).
		Order("created_at DESC").
		First(&updatedEvent).Error; err != nil {
		t.Fatalf("find updated event failed: %v", err)
	}
	if got, want := decodeSubscriptionEventFields(updatedEvent.ChangedFields), []string{"amount"}; !stringSlicesEqual(got, want) {
		t.Fatalf("updated event changed fields = %v, want %v", got, want)
	}
	if updatedEvent.PreviousMonthlyAmount == nil || *updatedEvent.PreviousMonthlyAmount != 10 {
		t.Fatalf("previous monthly amount = %v, want 10", updatedEvent.PreviousMonthlyAmount)
	}
	if updatedEvent.NewMonthlyAmount == nil || *updatedEvent.NewMonthlyAmount != 15 {
		t.Fatalf("new monthly amount = %v, want 15", updatedEvent.NewMonthlyAmount)
	}
}

func TestGetAnalyticsReportIncludesHistoryInsights(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-20"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Storage",
		Amount:          8,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	restoreClock()
	restoreClock = pkg.SetNowForTest(mustDate(t, "2026-03-22"))
	t.Cleanup(restoreClock)

	updatedAmount := 12.0
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		Amount: &updatedAmount,
	}); err != nil {
		t.Fatalf("update subscription failed: %v", err)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if got, want := len(report.PriceIncreases), 1; got != want {
		t.Fatalf("price_increases length = %d, want %d", got, want)
	}
	assertFloatEqual(t, report.PriceIncreases[0].DeltaMonthlyAmount, 4, "price increase delta")
	assertFloatEqual(t, report.PriceIncreases[0].DeltaPercentage, 50, "price increase percentage")

	if got, want := len(report.AnnualGrowth), 1; got != want {
		t.Fatalf("annual_growth length = %d, want %d", got, want)
	}
	if got, want := report.AnnualGrowth[0].Name, "Storage"; got != want {
		t.Fatalf("annual growth name = %q, want %q", got, want)
	}
	assertFloatEqual(t, report.AnnualGrowth[0].BaselineMonthlyAmount, 8, "annual growth baseline")
	assertFloatEqual(t, report.AnnualGrowth[0].CurrentMonthlyAmount, 12, "annual growth current")

	if got, want := len(report.RecentChanges), 2; got != want {
		t.Fatalf("recent_changes length = %d, want %d", got, want)
	}
	if got, want := report.RecentChanges[0].Type, subscriptionEventUpdated; got != want {
		t.Fatalf("recent change type = %q, want %q", got, want)
	}
}

func TestGetAnalyticsReportAnnualGrowthIgnoresCreationAndOldEvents(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-20"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	recentSub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Recent Storage",
		Amount:          8,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatalf("create recent subscription failed: %v", err)
	}

	oldSub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Old Storage",
		Amount:          8,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatalf("create old subscription failed: %v", err)
	}

	recentAmount := 12.0
	if _, err := service.Update(user.ID, recentSub.ID, UpdateSubscriptionInput{
		Amount: &recentAmount,
	}); err != nil {
		t.Fatalf("update recent subscription failed: %v", err)
	}

	oldEventTime := mustDate(t, "2025-01-01")
	if err := db.Model(&model.SubscriptionEvent{}).
		Where("subscription_id = ? AND type = ?", oldSub.ID, subscriptionEventUpdated).
		Update("created_at", oldEventTime).Error; err != nil {
		t.Fatalf("age old subscription event failed: %v", err)
	}
	oldAmount := 12.0
	if _, err := service.Update(user.ID, oldSub.ID, UpdateSubscriptionInput{
		Amount: &oldAmount,
	}); err != nil {
		t.Fatalf("update old subscription failed: %v", err)
	}
	if err := db.Model(&model.SubscriptionEvent{}).
		Where("subscription_id = ? AND type = ?", oldSub.ID, subscriptionEventUpdated).
		Update("created_at", oldEventTime).Error; err != nil {
		t.Fatalf("age latest old subscription event failed: %v", err)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if got, want := len(report.AnnualGrowth), 1; got != want {
		t.Fatalf("annual_growth length = %d, want %d", got, want)
	}
	if got, want := report.AnnualGrowth[0].Name, "Recent Storage"; got != want {
		t.Fatalf("annual growth item = %q, want %q", got, want)
	}
	assertFloatEqual(t, report.AnnualGrowth[0].BaselineMonthlyAmount, 8, "annual growth baseline")
	assertFloatEqual(t, report.AnnualGrowth[0].CurrentMonthlyAmount, 12, "annual growth current")
}

func assertFloatEqual(t *testing.T, got, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
