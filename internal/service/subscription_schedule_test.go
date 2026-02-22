package service

import (
	"testing"
	"time"
)

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse date %q: %v", value, err)
	}
	return normalizeDateUTC(parsed)
}

func TestNormalizeBillingDraft_MonthEndInterval(t *testing.T) {
	nextBilling := mustDate(t, "2025-02-28")
	count := 1
	draft := billingDraft{
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &count,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBilling,
	}

	_, next, err := normalizeBillingDraft(draft)
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2025-02-28"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
}

func TestNormalizeBillingDraft_LeapYearInterval(t *testing.T) {
	nextBilling := mustDate(t, "2026-02-28")
	count := 1
	draft := billingDraft{
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &count,
		IntervalUnit:    intervalUnitYear,
		NextBillingDate: &nextBilling,
	}

	_, next, err := normalizeBillingDraft(draft)
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2026-02-28"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
}

func TestNormalizeBillingDraft_MonthlySpecificDayValidation(t *testing.T) {
	nextBilling := mustDate(t, "2025-04-30")
	day := 31
	draft := billingDraft{
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeMonthlyDate,
		NextBillingDate: &nextBilling,
		MonthlyDay:      &day,
	}

	normalized, next, err := normalizeBillingDraft(draft)
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2025-04-30"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
	if normalized.IntervalCount != nil || normalized.IntervalUnit != "" {
		t.Fatal("interval fields should be cleared for monthly recurrence")
	}
}

func TestNormalizeBillingDraft_YearlySpecificValidation(t *testing.T) {
	nextBilling := mustDate(t, "2025-02-28")
	month := 2
	day := 29
	draft := billingDraft{
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeYearlyDate,
		NextBillingDate: &nextBilling,
		YearlyMonth:     &month,
		YearlyDay:       &day,
	}

	normalized, next, err := normalizeBillingDraft(draft)
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2025-02-28"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
	if normalized.IntervalCount != nil || normalized.IntervalUnit != "" || normalized.MonthlyDay != nil {
		t.Fatal("non-yearly recurrence fields should be cleared for yearly recurrence")
	}
}

func TestNormalizeBillingDraft_RequiresNextBillingDate(t *testing.T) {
	count := 1
	recurringDraft := billingDraft{
		BillingType:    billingTypeRecurring,
		RecurrenceType: recurrenceTypeInterval,
		IntervalCount:  &count,
		IntervalUnit:   intervalUnitMonth,
	}
	if _, _, err := normalizeBillingDraft(recurringDraft); err == nil {
		t.Fatal("normalizeBillingDraft() expected missing recurring next_billing_date error")
	}

	oneTimeDraft := billingDraft{
		BillingType: billingTypeOneTime,
	}
	if _, _, err := normalizeBillingDraft(oneTimeDraft); err == nil {
		t.Fatal("normalizeBillingDraft() expected missing one-time next_billing_date error")
	}
}

func TestNormalizeBillingDraft_OneTimeAndLifetimeBehaveSame(t *testing.T) {
	nextBillingDate := mustDate(t, "2025-02-20")

	testCases := []string{billingTypeOneTime, billingTypeLifetime}
	for _, billingType := range testCases {
		billingType := billingType
		t.Run(billingType, func(t *testing.T) {
			draft := billingDraft{
				BillingType:     billingType,
				NextBillingDate: &nextBillingDate,
			}

			normalized, next, err := normalizeBillingDraft(draft)
			if err != nil {
				t.Fatalf("normalizeBillingDraft() error = %v", err)
			}
			if normalized.BillingType != billingTypeOneTime {
				t.Fatalf("billing type = %s, want %s", normalized.BillingType, billingTypeOneTime)
			}
			if next == nil {
				t.Fatal("next billing date should not be nil")
			}
			if got, want := next.Format("2006-01-02"), "2025-02-20"; got != want {
				t.Fatalf("next billing date = %s, want %s", got, want)
			}
		})
	}
}

func TestNormalizeBillingDraft_IntervalValidationStillApplies(t *testing.T) {
	nextBillingDate := mustDate(t, "2025-02-20")
	draft := billingDraft{
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		NextBillingDate: &nextBillingDate,
	}

	_, _, err := normalizeBillingDraft(draft)
	if err != nil {
		if got, want := err.Error(), "interval_count must be at least 1 for interval recurrence"; got != want {
			t.Fatalf("normalizeBillingDraft() error = %q, want %q", got, want)
		}
		return
	}
	t.Fatal("normalizeBillingDraft() expected interval validation error")
}
