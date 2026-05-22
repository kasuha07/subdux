package service

import (
	"math"
	"testing"

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

func assertFloatEqual(t *testing.T, got, want float64, label string) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
