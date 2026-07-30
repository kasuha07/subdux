package subscription

import (
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

// noisyConverter models an exchange-rate conversion that lands a hair off the
// minor-unit grid, which is how sub-cent drift enters converted amounts.
type noisyConverter struct{}

func (noisyConverter) Convert(amount float64, _, _ string) (float64, bool) {
	return amount * 1.0000000000001, true
}

// seedDriftedPriceEvent writes a price-bearing event whose two monthly amounts
// differ only below the currency's minor unit, reproducing the drifted rows
// already stored by earlier versions.
func seedDriftedPriceEvent(t *testing.T, db *gorm.DB, userID uint, sub model.Subscription, previous, next float64) {
	t.Helper()

	subscriptionID := sub.ID
	event := model.SubscriptionEvent{
		UserID:                userID,
		SubscriptionID:        &subscriptionID,
		SubscriptionName:      sub.Name,
		Type:                  subscriptionEventUpdated,
		ChangedFields:         encodeSubscriptionEventFields([]string{"monthly_amount"}),
		PreviousAmount:        float64Ptr(previous),
		NewAmount:             float64Ptr(next),
		PreviousMonthlyAmount: float64Ptr(previous),
		NewMonthlyAmount:      float64Ptr(next),
		PreviousCurrency:      sub.Currency,
		NewCurrency:           sub.Currency,
		CreatedAt:             pkg.NowUTC(),
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create drifted price event failed: %v", err)
	}
}

func newDriftTestSubscription(t *testing.T, service *Service, userID uint) model.Subscription {
	t.Helper()

	monthly := 1
	sub, err := service.Create(userID, CreateSubscriptionInput{
		Name:            "Steady Plan",
		Amount:          9.99,
		Currency:        "EUR",
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
	return *sub
}

func TestAnalyticsReportIgnoresSubMinorUnitPriceDrift(t *testing.T) {
	tests := []struct {
		name      string
		target    string
		converter CurrencyConverter
	}{
		// Drift arriving through a noisy exchange-rate conversion.
		{name: "converted amounts", target: "USD", converter: noisyConverter{}},
		// Drift already stored in the event row, with no conversion to absorb
		// it: only the minor-unit comparison stands between it and a bogus
		// "price increased" report.
		{name: "stored amounts without conversion", target: "EUR", converter: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
			t.Cleanup(restoreClock)

			db := newTestDB(t)
			user := createTestUser(t, db)
			service := NewService(db)

			sub := newDriftTestSubscription(t, service, user.ID)
			seedDriftedPriceEvent(t, db, user.ID, sub, 9.99, 9.990000000000002)

			report, err := service.GetAnalyticsReport(user.ID, tt.target, tt.converter)
			if err != nil {
				t.Fatalf("GetAnalyticsReport() error = %v", err)
			}

			if got := len(report.PriceIncreases); got != 0 {
				t.Fatalf("price_increases length = %d, want 0 (drift below one cent is not a price increase): %+v", got, report.PriceIncreases)
			}
			if got := len(report.AnnualGrowth); got != 0 {
				t.Fatalf("annual_growth length = %d, want 0 (drift below one cent is not growth): %+v", got, report.AnnualGrowth)
			}
		})
	}
}

func TestAnalyticsReportStillReportsRealPriceIncrease(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	sub := newDriftTestSubscription(t, service, user.ID)
	seedDriftedPriceEvent(t, db, user.ID, sub, 9.99, 11.49)

	report, err := service.GetAnalyticsReport(user.ID, "EUR", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	if got := len(report.PriceIncreases); got != 1 {
		t.Fatalf("price_increases length = %d, want 1", got)
	}
	assertExactAmount(t, report.PriceIncreases[0].DeltaMonthlyAmount, 1.5, "price increase delta")
}

func TestActionCenterIgnoresSubMinorUnitPriceDrift(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	sub := newDriftTestSubscription(t, service, user.ID)
	seedDriftedPriceEvent(t, db, user.ID, sub, 9.99, 9.990000000000002)

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	for _, item := range center.Items {
		if item.Type == actionTypePriceIncrease {
			t.Fatalf("price increase action = %+v, want none for sub-cent drift", item)
		}
	}
}

func TestActionCenterStillReportsRealPriceIncrease(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	sub := newDriftTestSubscription(t, service, user.ID)
	seedDriftedPriceEvent(t, db, user.ID, sub, 9.99, 11.49)

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	var found *SubscriptionAction
	for i := range center.Items {
		if center.Items[i].Type == actionTypePriceIncrease {
			found = &center.Items[i]
			break
		}
	}
	if found == nil {
		t.Fatal("price increase action = none, want the 1.50 increase reported")
	}
	if found.DeltaMonthlyAmount == nil || *found.DeltaMonthlyAmount != 1.5 {
		t.Fatalf("delta_monthly_amount = %v, want exactly 1.5", found.DeltaMonthlyAmount)
	}
}

// createTenCentSubscriptions makes three 0.10/month subscriptions, whose naive
// float sum is 0.30000000000000004.
func createTenCentSubscriptions(t *testing.T, service *Service, userID uint) {
	t.Helper()

	monthly := 1
	for _, name := range []string{"Dime A", "Dime B", "Dime C"} {
		if _, err := service.Create(userID, CreateSubscriptionInput{
			Name:            name,
			Amount:          0.1,
			Currency:        "USD",
			Status:          subscriptionStatusActive,
			RenewalMode:     renewalModeAutoRenew,
			BillingType:     billingTypeRecurring,
			RecurrenceType:  recurrenceTypeInterval,
			IntervalCount:   &monthly,
			IntervalUnit:    intervalUnitMonth,
			NextBillingDate: "2026-03-15",
		}); err != nil {
			t.Fatalf("create %q failed: %v", name, err)
		}
	}
}

func TestDashboardSummaryTotalsLandOnMinorUnitGrid(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	createTenCentSubscriptions(t, service, user.ID)

	summary, err := service.GetDashboardSummary(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}

	assertExactAmount(t, summary.TotalMonthly, 0.3, "total_monthly")
	assertExactAmount(t, summary.CommittedMonthly, 0.3, "committed_monthly")
	assertExactAmount(t, summary.DueThisMonth, 0.3, "due_this_month")
	assertExactAmount(t, summary.TotalYearly, 3.6, "total_yearly")
	assertExactAmount(t, summary.CommittedYearly, 3.6, "committed_yearly")
}

func TestAnalyticsReportTotalsLandOnMinorUnitGrid(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	createTenCentSubscriptions(t, service, user.ID)

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}

	assertExactAmount(t, report.KPIs.TotalMonthly, 0.3, "total_monthly")
	assertExactAmount(t, report.KPIs.TotalYearly, 3.6, "total_yearly")
	assertExactAmount(t, report.KPIs.DueThisMonth, 0.3, "due_this_month")
	assertExactAmount(t, report.KPIs.DueNext30Days, 0.3, "due_next_30_days")
	assertExactAmount(t, report.MonthlyForecast[0].AmountDue, 0.3, "march amount_due")

	if got := len(report.RenewalModeBreakdown); got != 1 {
		t.Fatalf("renewal_mode_breakdown length = %d, want 1", got)
	}
	assertExactAmount(t, report.RenewalModeBreakdown[0].MonthlyAmount, 0.3, "breakdown monthly_amount")
	assertExactAmount(t, report.RenewalModeBreakdown[0].YearlyAmount, 3.6, "breakdown yearly_amount")
	assertExactAmount(t, report.RenewalModeBreakdown[0].Percentage, 100, "breakdown percentage")
}

func TestSubscriptionEventPersistsRoundedMonthlyAmount(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	tests := []struct {
		name     string
		amount   float64
		currency string
		want     float64
	}{
		// 100/12 is 8.333333333333334 before quantization.
		{name: "two decimal currency", amount: 100, currency: "USD", want: 8.33},
		// JPY has no minor unit, so 12000/12 stays whole and 1000/12 truncates
		// to a whole yen.
		{name: "zero decimal currency", amount: 1000, currency: "JPY", want: 83},
	}

	yearly := 1
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sub, err := service.Create(user.ID, CreateSubscriptionInput{
				Name:            "Yearly " + tt.currency,
				Amount:          tt.amount,
				Currency:        tt.currency,
				Status:          subscriptionStatusActive,
				RenewalMode:     renewalModeAutoRenew,
				BillingType:     billingTypeRecurring,
				RecurrenceType:  recurrenceTypeInterval,
				IntervalCount:   &yearly,
				IntervalUnit:    intervalUnitYear,
				NextBillingDate: "2026-04-15",
			})
			if err != nil {
				t.Fatalf("create subscription failed: %v", err)
			}

			var event model.SubscriptionEvent
			if err := db.Where("subscription_id = ? AND type = ?", sub.ID, subscriptionEventCreated).
				First(&event).Error; err != nil {
				t.Fatalf("find created event failed: %v", err)
			}
			if event.NewMonthlyAmount == nil {
				t.Fatal("new_monthly_amount = nil, want a recorded amount")
			}
			assertExactAmount(t, *event.NewMonthlyAmount, tt.want, "new_monthly_amount")

			// The user-entered amount is stored verbatim, never rounded.
			assertExactAmount(t, *event.NewAmount, tt.amount, "new_amount")
		})
	}
}

func assertExactAmount(t *testing.T, got, want float64, label string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want exactly %v", label, got, want)
	}
}
