package service

import (
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/pkg"
)

func TestGenerateICalFeedOmitsRRuleForNonAutoRenewRecurring(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)
	calendarService := NewCalendarService(db)

	intervalCount := 1
	if _, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual recurring",
		Amount:          7.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-10",
	}); err != nil {
		t.Fatalf("create manual recurring subscription failed: %v", err)
	}

	feed, err := calendarService.GenerateICalFeed(user.ID)
	if err != nil {
		t.Fatalf("GenerateICalFeed() error = %v", err)
	}

	if strings.Contains(feed, "RRULE:") {
		t.Fatal("manual_renew subscription should not emit RRULE in iCal feed")
	}
}
