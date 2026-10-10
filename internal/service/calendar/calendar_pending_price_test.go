package calendar

import (
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/pkg"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func createPendingPriceCalendarSubscription(t *testing.T, service *subscriptionservice.Service, userID uint, name string, amount float64, nextBilling, pendingFrom string) {
	t.Helper()
	intervalCount := 1
	pendingAmount := 12.0
	if _, err := service.Create(userID, subscriptionservice.CreateSubscriptionInput{
		Name:            name,
		Amount:          amount,
		Currency:        "USD",
		BillingType:     subscriptionservice.BillingTypeRecurring,
		RecurrenceType:  subscriptionservice.RecurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    subscriptionservice.IntervalUnitMonth,
		NextBillingDate: nextBilling,
		PendingAmount:   &pendingAmount,
		PendingFrom:     pendingFrom,
	}); err != nil {
		t.Fatalf("create %q failed: %v", name, err)
	}
}

func TestGenerateICalFeedSplitsSeriesAtScheduledPrice(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := subscriptionservice.NewService(db)
	createPendingPriceCalendarSubscription(t, service, user.ID, "Intro Plan", 1, "2026-03-10", "2026-06-10")

	feed, err := NewService(db).GenerateICalFeed(user.ID)
	if err != nil {
		t.Fatalf("GenerateICalFeed() error = %v", err)
	}

	for _, want := range []string{
		"UID:subdux-sub-1@subdux",
		"DTSTART;VALUE=DATE:20260310",
		"SUMMARY:Intro Plan - 1.00 USD",
		"RRULE:FREQ=MONTHLY;INTERVAL=1;UNTIL=20260609",
		"UID:subdux-sub-1-pending@subdux",
		"DTSTART;VALUE=DATE:20260610",
		"SUMMARY:Intro Plan - 12.00 USD",
	} {
		if !strings.Contains(feed, want) {
			t.Fatalf("iCal feed missing %q; feed = %q", want, feed)
		}
	}
}

func TestGenerateICalFeedTrialStartsAtScheduledPrice(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := subscriptionservice.NewService(db)
	// A trial bills nothing before its first charge, which is already at the
	// regular price: one series, no split.
	createPendingPriceCalendarSubscription(t, service, user.ID, "Trial Plan", 0, "2026-03-08", "2026-03-08")

	feed, err := NewService(db).GenerateICalFeed(user.ID)
	if err != nil {
		t.Fatalf("GenerateICalFeed() error = %v", err)
	}
	if !strings.Contains(feed, "SUMMARY:Trial Plan - 12.00 USD") {
		t.Fatalf("trial feed should show the regular price; feed = %q", feed)
	}
	if strings.Contains(feed, "-pending@subdux") || strings.Contains(feed, "UNTIL=") {
		t.Fatalf("trial feed should not split the series; feed = %q", feed)
	}
}
