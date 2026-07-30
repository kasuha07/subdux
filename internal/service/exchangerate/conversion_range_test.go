package exchangerate

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

func TestConvertUsesTargetCurrencyMinorUnitRangeInsteadOfSubscriptionMaximum(t *testing.T) {
	db := newTestDB(t)
	if err := db.Create(&[]model.ExchangeRate{
		{TargetCurrency: "JPY", Rate: 2, Source: "test", FetchedAt: time.Now().UTC()},
		{TargetCurrency: "CLF", Rate: 2, Source: "test", FetchedAt: time.Now().UTC()},
		{TargetCurrency: "EUR", Rate: 0.5, Source: "test", FetchedAt: time.Now().UTC()},
	}).Error; err != nil {
		t.Fatalf("seed exchange rates failed: %v", err)
	}
	service := NewService(db)

	got, ok := service.Convert(400_000_000_000, "USD", "JPY")
	if !ok || got != 800_000_000_000 {
		t.Fatalf("Convert(large USD to JPY) = %v, %v; want 800000000000, true", got, ok)
	}

	if got, ok := service.Convert(500_000_000_000, "USD", "CLF"); ok || got != 0 {
		t.Fatalf("Convert(minor-unit-unsafe USD to CLF) = %v, %v; want 0, false", got, ok)
	}

	got, ok = service.Convert(1.005, "USD", "EUR")
	if !ok || got != 0.5 {
		t.Fatalf("Convert(unrounded source amount) = %v, %v; want 0.5, true", got, ok)
	}
}

func TestConvertAcceptsGrandfatheredSameCurrencyAmountWhenMinorUnitsAreSafe(t *testing.T) {
	service := NewService(newTestDB(t))

	const legacyAmount = 700_000_000_000.0
	got, ok := service.Convert(legacyAmount, "USD", "USD")
	if !ok || got != legacyAmount {
		t.Fatalf("Convert(grandfathered USD amount) = %v, %v; want %v, true", got, ok, legacyAmount)
	}
}
