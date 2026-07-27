package subscription

import (
	"testing"

	"github.com/kasuha07/subdux/internal/pkg"
)

// newCurrencySwitchSubscription creates a 7.00 USD/month subscription, the
// "before" side of a currency switch.
func newCurrencySwitchSubscription(t *testing.T, service *Service, userID uint) uint {
	t.Helper()

	monthly := 1
	sub, err := service.Create(userID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          7,
		Currency:        "USD",
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
	return sub.ID
}

// switchToYen re-denominates the subscription at 1000 JPY/month, which is worth
// roughly the same as 7.00 USD/month. The event records 7.00 in USD and 1000 in
// JPY, so a raw numeric comparison would read as a ~14000% price increase.
func switchToYen(t *testing.T, service *Service, userID, subscriptionID uint) {
	t.Helper()

	amount := 1000.0
	currency := "JPY"
	if _, err := service.Update(userID, subscriptionID, UpdateSubscriptionInput{
		Amount:   &amount,
		Currency: &currency,
	}); err != nil {
		t.Fatalf("switch subscription currency failed: %v", err)
	}
}

func TestActionCenterIgnoresCurrencySwitchAsPriceChange(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	subscriptionID := newCurrencySwitchSubscription(t, service, user.ID)
	switchToYen(t, service, user.ID, subscriptionID)

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	for _, item := range center.Items {
		if item.Type == actionTypePriceIncrease {
			t.Fatalf("price increase action = %+v, want none: 7.00 USD and 1000 JPY are not comparable without conversion", item)
		}
	}
}

func TestActionCenterCurrencySwitchSuppressesOlderPriceIncrease(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)

	subscriptionID := newCurrencySwitchSubscription(t, service, user.ID)

	increasedAmount := 9.0
	if _, err := service.Update(user.ID, subscriptionID, UpdateSubscriptionInput{Amount: &increasedAmount}); err != nil {
		t.Fatalf("increase subscription amount failed: %v", err)
	}
	switchToYen(t, service, user.ID, subscriptionID)

	center, err := service.GetActionCenter(user.ID)
	if err != nil {
		t.Fatalf("GetActionCenter() error = %v", err)
	}

	for _, item := range center.Items {
		if item.Type == actionTypePriceIncrease {
			t.Fatalf("price increase action = %+v, want the newer currency switch to suppress the stale USD increase", item)
		}
	}
}
