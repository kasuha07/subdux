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

func TestMapSubscriptionResponseIncludesTimestamps(t *testing.T) {
	createdAt := time.Date(2026, time.February, 23, 9, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.February, 24, 10, 45, 0, 0, time.UTC)

	response := mapSubscriptionResponse(model.Subscription{
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})

	if !response.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %s, want %s", response.CreatedAt, createdAt)
	}
	if !response.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("updated_at = %s, want %s", response.UpdatedAt, updatedAt)
	}
}
