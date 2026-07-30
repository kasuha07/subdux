package subscription

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

type reportZeroDeltaConverter struct{}

func (reportZeroDeltaConverter) Convert(amount float64, _, _ string) (float64, bool) {
	return amount * 1.1, true
}

func TestReportPriceIncreasesConsumesLatestConvertedMinorUnitEqualEvent(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	subscriptionID := uint(1)
	oldPrevious, oldCurrent := 5.0, 8.0
	latestPrevious, latestCurrent := 10.0, 10.004
	events := []model.SubscriptionEvent{
		{
			UserID:                user.ID,
			SubscriptionID:        &subscriptionID,
			SubscriptionName:      "Drifted plan",
			Type:                  subscriptionEventUpdated,
			PreviousMonthlyAmount: &oldPrevious,
			NewMonthlyAmount:      &oldCurrent,
			PreviousCurrency:      "EUR",
			NewCurrency:           "EUR",
			CreatedAt:             time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			UserID:                user.ID,
			SubscriptionID:        &subscriptionID,
			SubscriptionName:      "Drifted plan",
			Type:                  subscriptionEventUpdated,
			PreviousMonthlyAmount: &latestPrevious,
			NewMonthlyAmount:      &latestCurrent,
			PreviousCurrency:      "EUR",
			NewCurrency:           "EUR",
			CreatedAt:             time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatalf("create price events failed: %v", err)
	}

	items, err := service.reportPriceIncreases(user.ID, "USD", reportZeroDeltaConverter{})
	if err != nil {
		t.Fatalf("reportPriceIncreases() error = %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("price increases = %+v, want latest converted minor-unit-equal event to suppress older increase", items)
	}
}
