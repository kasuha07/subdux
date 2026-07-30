package subscription

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

type unavailableCurrencyConverter struct{}

func (unavailableCurrencyConverter) Convert(float64, string, string) (float64, bool) {
	return 0, false
}

func TestDashboardFailsClosedWhenExchangeRateIsUnavailable(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(1.234)
	input.Name = "KWD plan"
	input.Currency = "KWD"
	if _, err := service.Create(user.ID, input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := service.GetDashboardSummary(user.ID, "JPY", unavailableCurrencyConverter{})
	if !errors.Is(err, ErrExchangeRateUnavailable) {
		t.Fatalf("GetDashboardSummary() error = %v, want %v", err, ErrExchangeRateUnavailable)
	}
}

func TestAnalyticsReportFailsClosedWhenExchangeRateIsUnavailable(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(1.234)
	input.Name = "KWD plan"
	input.Currency = "KWD"
	if _, err := service.Create(user.ID, input); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err := service.GetAnalyticsReport(user.ID, "JPY", unavailableCurrencyConverter{})
	if !errors.Is(err, ErrExchangeRateUnavailable) {
		t.Fatalf("GetAnalyticsReport() error = %v, want %v", err, ErrExchangeRateUnavailable)
	}
}

func TestAnnualGrowthSkipsIneligibleForeignSubscriptionBeforeConversion(t *testing.T) {
	restoreClock := pkg.SetNowForTest(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	monthly := 1
	nextBillingDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	sub := model.Subscription{
		UserID:          user.ID,
		Name:            "Canceling KWD growth",
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
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}
	previous, current := 1.0, 2.0
	subscriptionID := sub.ID
	if err := db.Create(&model.SubscriptionEvent{
		UserID:                user.ID,
		SubscriptionID:        &subscriptionID,
		SubscriptionName:      sub.Name,
		Type:                  subscriptionEventUpdated,
		PreviousMonthlyAmount: &previous,
		NewMonthlyAmount:      &current,
		PreviousCurrency:      "KWD",
		NewCurrency:           "KWD",
		CreatedAt:             time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create price event failed: %v", err)
	}

	items, err := service.reportAnnualGrowth(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("reportAnnualGrowth() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("annual growth = %+v, want none for cancel-at-period-end subscription", items)
	}
}

func TestAnnualGrowthEligibleForeignSubscriptionFailsClosedWithoutExchangeRate(t *testing.T) {
	restoreClock := pkg.SetNowForTest(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	monthly := 1
	nextBillingDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	sub := model.Subscription{
		UserID:          user.ID,
		Name:            "Active KWD growth",
		Amount:          2,
		Currency:        "KWD",
		Enabled:         true,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}
	previous, current := 1.0, 2.0
	subscriptionID := sub.ID
	if err := db.Create(&model.SubscriptionEvent{
		UserID:                user.ID,
		SubscriptionID:        &subscriptionID,
		SubscriptionName:      sub.Name,
		Type:                  subscriptionEventUpdated,
		PreviousMonthlyAmount: &previous,
		NewMonthlyAmount:      &current,
		PreviousCurrency:      "KWD",
		NewCurrency:           "KWD",
		CreatedAt:             time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create price event failed: %v", err)
	}

	if _, err := service.reportAnnualGrowth(user.ID, "USD", nil); !errors.Is(err, ErrExchangeRateUnavailable) {
		t.Fatalf("reportAnnualGrowth() error = %v, want %v", err, ErrExchangeRateUnavailable)
	}
}

func TestReportPriceIncreasesConsumesNewestCurrencySwitch(t *testing.T) {
	restoreClock := pkg.SetNowForTest(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Migrated plan",
		Amount:          20,
		Currency:        "JPY",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	oldPrevious, oldCurrent := 10.0, 20.0
	newPrevious, newCurrent := 20.0, 1000.0
	oldID, newID := sub.ID, sub.ID
	if err := db.Create(&model.SubscriptionEvent{
		UserID:                user.ID,
		SubscriptionID:        &oldID,
		SubscriptionName:      sub.Name,
		Type:                  subscriptionEventUpdated,
		PreviousMonthlyAmount: &oldPrevious,
		NewMonthlyAmount:      &oldCurrent,
		PreviousCurrency:      "USD",
		NewCurrency:           "USD",
		CreatedAt:             time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create old price event failed: %v", err)
	}
	if err := db.Create(&model.SubscriptionEvent{
		UserID:                user.ID,
		SubscriptionID:        &newID,
		SubscriptionName:      sub.Name,
		Type:                  subscriptionEventUpdated,
		PreviousMonthlyAmount: &newPrevious,
		NewMonthlyAmount:      &newCurrent,
		PreviousCurrency:      "USD",
		NewCurrency:           "JPY",
		CreatedAt:             time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create newest currency-switch event failed: %v", err)
	}

	items, err := service.reportPriceIncreases(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("reportPriceIncreases() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("price increases = %+v, want none after newest currency switch", items)
	}
}
