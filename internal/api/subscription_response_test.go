package api

import (
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

func TestMapSubscriptionResponseFormatsDateOnly(t *testing.T) {
	nextBillingDate := time.Date(2026, time.February, 24, 9, 30, 0, 0, time.FixedZone("+08", 8*60*60))
	response := mapSubscriptionResponse(model.Subscription{
		NextBillingDate: &nextBillingDate,
	})

	if response.NextBillingDate == nil {
		t.Fatal("next_billing_date should not be nil")
	}
	if got, want := *response.NextBillingDate, "2026-02-24"; got != want {
		t.Fatalf("next_billing_date = %s, want %s", got, want)
	}
}

func TestMapSubscriptionResponseFormatsNilDate(t *testing.T) {
	response := mapSubscriptionResponse(model.Subscription{})
	if response.NextBillingDate != nil {
		t.Fatalf("next_billing_date = %v, want nil", *response.NextBillingDate)
	}
}
