package subscription

import (
	"errors"
	"math"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
)

func validSubscriptionInput(amount float64) CreateSubscriptionInput {
	monthly := 1
	return CreateSubscriptionInput{
		Name:            "Amount validation plan",
		Amount:          amount,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-01",
	}
}

func TestCreateRejectsInvalidAmountsAtServiceBoundary(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   error
	}{
		{name: "negative", amount: -1, want: ErrAmountMustNotBeNegative},
		{name: "nan", amount: math.NaN(), want: ErrAmountMustBeFinite},
		{name: "negative infinity", amount: math.Inf(-1), want: ErrAmountMustBeFinite},
		{name: "positive infinity", amount: math.Inf(1), want: ErrAmountTooLarge},
		{name: "above maximum", amount: money.MaxAmount + 0.01, want: ErrAmountTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t)
			user := createTestUser(t, db)
			service := NewService(db)

			if _, err := service.Create(user.ID, validSubscriptionInput(tt.amount)); !errors.Is(err, tt.want) {
				t.Fatalf("Create(%v) error = %v, want %v", tt.amount, err, tt.want)
			}

			var count int64
			if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
				t.Fatalf("count subscriptions failed: %v", err)
			}
			if count != 0 {
				t.Fatalf("subscription count = %d, want 0", count)
			}
		})
	}
}

func TestUpdateRejectsInvalidAmountWithoutChangingStoredValue(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	sub, err := service.Create(user.ID, validSubscriptionInput(12.34))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for _, amount := range []float64{-1, math.NaN(), money.MaxAmount + 0.01} {
		if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &amount}); err == nil {
			t.Fatalf("Update(%v) error = nil, want validation error", amount)
		}
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.Amount != 12.34 {
		t.Fatalf("stored amount = %v, want 12.34", stored.Amount)
	}
}

func TestCreateRejectsDerivedMonthlyAmountOutsideCurrencyMinorUnitRange(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(money.MaxAmount)
	input.Name = "Daily CLF maximum plan"
	input.Currency = "CLF"
	input.IntervalUnit = intervalUnitDay
	if _, err := service.Create(user.ID, input); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Create() error = %v, want %v for minor-unit-unsafe monthly amount", err, ErrAmountTooLarge)
	}

	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0", count)
	}
}

func TestCreateRejectsDerivedYearlyAmountOutsideCurrencyMinorUnitRange(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(money.MaxAmount)
	input.Name = "Daily USD maximum plan"
	input.IntervalUnit = intervalUnitDay
	if _, err := service.Create(user.ID, input); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Create() error = %v, want %v for minor-unit-unsafe yearly amount", err, ErrAmountTooLarge)
	}

	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0", count)
	}
}

func TestUpdateRejectsScheduleWithDerivedMonthlyAmountOutsideCurrencyMinorUnitRange(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(money.MaxAmount / 12)
	input.Currency = "CLF"
	sub, err := service.Create(user.ID, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	daily := intervalUnitDay
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		IntervalUnit: &daily,
	}); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Update() error = %v, want %v for minor-unit-unsafe monthly amount", err, ErrAmountTooLarge)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.IntervalUnit != intervalUnitMonth {
		t.Fatalf("stored interval_unit = %q, want %q", stored.IntervalUnit, intervalUnitMonth)
	}
}

func TestUpdateRejectsScheduleWithDerivedYearlyAmountOutsideCurrencyMinorUnitRange(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	input := validSubscriptionInput(money.MaxAmount)
	sub, err := service.Create(user.ID, input)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	daily := intervalUnitDay
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		IntervalUnit: &daily,
	}); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Update() error = %v, want %v for minor-unit-unsafe yearly amount", err, ErrAmountTooLarge)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.IntervalUnit != intervalUnitMonth {
		t.Fatalf("stored interval_unit = %q, want %q", stored.IntervalUnit, intervalUnitMonth)
	}
}

func TestUpdateRejectsCurrencyWithDerivedYearlyAmountOutsideMinorUnitRange(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	const amount = 100_000_000_000.0
	sub, err := service.Create(user.ID, validSubscriptionInput(amount))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	currency := "CLF"
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		Currency: &currency,
	}); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Update() error = %v, want %v for minor-unit-unsafe yearly amount", err, ErrAmountTooLarge)
	}

	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load subscription failed: %v", err)
	}
	if stored.Currency != "USD" {
		t.Fatalf("stored currency = %q, want USD", stored.Currency)
	}
}
