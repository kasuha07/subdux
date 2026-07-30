package subscription

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

type successfulDashboardReportConverter struct{}

func (successfulDashboardReportConverter) Convert(amount float64, _, _ string) (float64, bool) {
	return amount * 0.5025, true
}

type failedDashboardReportConverter struct{}

func (failedDashboardReportConverter) Convert(float64, string, string) (float64, bool) {
	return 0, false
}

func TestDashboardAndReportShareSubscriptionAmountConversion(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		currency       string
		targetCurrency string
		converter      CurrencyConverter
		want           float64
	}{
		{
			name:           "same currency quantization",
			amount:         1.005,
			currency:       "USD",
			targetCurrency: "USD",
			want:           1.01,
		},
		{
			name:           "successful cross currency conversion",
			amount:         2,
			currency:       "KWD",
			targetCurrency: "USD",
			converter:      successfulDashboardReportConverter{},
			want:           1.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
			t.Cleanup(restoreClock)

			db := newTestDB(t)
			user := createTestUser(t, db)
			service := NewService(db)
			seedDashboardReportConversionSubscription(t, service, user.ID, tt.amount, tt.currency)

			summary, err := service.GetDashboardSummary(user.ID, tt.targetCurrency, tt.converter)
			if err != nil {
				t.Fatalf("GetDashboardSummary() error = %v", err)
			}
			report, err := service.GetAnalyticsReport(user.ID, tt.targetCurrency, tt.converter)
			if err != nil {
				t.Fatalf("GetAnalyticsReport() error = %v", err)
			}

			if got := summary.TotalMonthly; got != tt.want {
				t.Fatalf("dashboard total_monthly = %v, want %v", got, tt.want)
			}
			if got := report.KPIs.TotalMonthly; got != tt.want {
				t.Fatalf("report total_monthly = %v, want %v", got, tt.want)
			}
			if summary.TotalMonthly != report.KPIs.TotalMonthly {
				t.Fatalf(
					"dashboard total_monthly = %v, report total_monthly = %v",
					summary.TotalMonthly,
					report.KPIs.TotalMonthly,
				)
			}
		})
	}
}

func TestDashboardAndReportFailClosedOnUnavailableConversion(t *testing.T) {
	tests := []struct {
		name      string
		converter CurrencyConverter
	}{
		{name: "missing converter"},
		{name: "failed converter", converter: failedDashboardReportConverter{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			user := createTestUser(t, db)
			service := NewService(db)
			seedDashboardReportConversionSubscription(t, service, user.ID, 2, "KWD")

			if _, err := service.GetDashboardSummary(user.ID, "USD", tt.converter); !errors.Is(err, ErrExchangeRateUnavailable) {
				t.Fatalf("GetDashboardSummary() error = %v, want %v", err, ErrExchangeRateUnavailable)
			}
			if _, err := service.GetAnalyticsReport(user.ID, "USD", tt.converter); !errors.Is(err, ErrExchangeRateUnavailable) {
				t.Fatalf("GetAnalyticsReport() error = %v, want %v", err, ErrExchangeRateUnavailable)
			}
		})
	}
}

func TestDashboardAndReportSkipConversionWithoutMonetaryContribution(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	monthly := 1
	nextBillingDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&model.Subscription{
		UserID:          user.ID,
		Name:            "Canceling foreign plan",
		Amount:          2,
		Currency:        "KWD",
		Enabled:         true,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBillingDate,
	}).Error; err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	summary, err := service.GetDashboardSummary(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}
	if summary.TotalMonthly != 0 || summary.DueThisMonth != 0 {
		t.Fatalf("dashboard monetary totals = monthly %v, due %v, want zero", summary.TotalMonthly, summary.DueThisMonth)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}
	if report.KPIs.TotalMonthly != 0 || report.KPIs.DueThisMonth != 0 || report.KPIs.DueNext30Days != 0 {
		t.Fatalf(
			"report monetary totals = monthly %v, due month %v, due 30 days %v, want zero",
			report.KPIs.TotalMonthly,
			report.KPIs.DueThisMonth,
			report.KPIs.DueNext30Days,
		)
	}
}

func seedDashboardReportConversionSubscription(
	t *testing.T,
	service *Service,
	userID uint,
	amount float64,
	currency string,
) {
	t.Helper()

	monthly := 1
	if _, err := service.Create(userID, CreateSubscriptionInput{
		Name:            "Conversion plan",
		Amount:          amount,
		Currency:        currency,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}
