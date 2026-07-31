package calendar

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/money"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestGenerateICalFeedFormatsAmountWithCurrencyMinorUnit(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := subscriptionservice.NewService(db)
	calendarService := NewService(db)

	intervalCount := 1
	subscriptions := []struct {
		name     string
		amount   float64
		currency string
		want     string
	}{
		{name: "Yen Plan", amount: 1200, currency: "JPY", want: "SUMMARY:Yen Plan - 1200 JPY"},
		{name: "Dinar Plan", amount: 1.2, currency: "KWD", want: "SUMMARY:Dinar Plan - 1.200 KWD"},
		{name: "Dollar Plan", amount: 7.5, currency: "USD", want: "SUMMARY:Dollar Plan - 7.50 USD"},
	}

	for _, tt := range subscriptions {
		if _, err := service.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
			Name:            tt.name,
			Amount:          tt.amount,
			Currency:        tt.currency,
			Status:          subscriptionservice.StatusActive,
			RenewalMode:     subscriptionservice.RenewalModeAutoRenew,
			BillingType:     subscriptionservice.BillingTypeRecurring,
			RecurrenceType:  subscriptionservice.RecurrenceTypeInterval,
			IntervalCount:   &intervalCount,
			IntervalUnit:    subscriptionservice.IntervalUnitMonth,
			NextBillingDate: "2026-03-10",
		}); err != nil {
			t.Fatalf("create %q failed: %v", tt.name, err)
		}
	}

	feed, err := calendarService.GenerateICalFeed(user.ID)
	if err != nil {
		t.Fatalf("GenerateICalFeed() error = %v", err)
	}

	for _, tt := range subscriptions {
		if !strings.Contains(feed, tt.want) {
			t.Fatalf("iCal feed missing %q; feed = %q", tt.want, feed)
		}
	}
}

func TestGenerateICalFeedRejectsUnsafeAmount(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	nextBillingDate := mustDate(t, "2026-03-10")
	sub := model.Subscription{
		UserID:          user.ID,
		Name:            "Corrupt amount",
		Amount:          math.MaxFloat64,
		Currency:        "USD",
		Status:          subscriptionservice.StatusActive,
		RenewalMode:     subscriptionservice.RenewalModeAutoRenew,
		BillingType:     subscriptionservice.BillingTypeRecurring,
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("create corrupt subscription: %v", err)
	}

	feed, err := NewService(db).GenerateICalFeed(user.ID)
	if !errors.Is(err, money.ErrUnsafeFormat) || feed != "" {
		t.Fatalf("GenerateICalFeed() = %q, %v; want empty ErrUnsafeFormat", feed, err)
	}
}
