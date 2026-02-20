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
	anchor := mustDate(t, "2025-01-31")
	count := 1
	draft := billingDraft{
		BillingType:       billingTypeRecurring,
		RecurrenceType:    recurrenceTypeInterval,
		IntervalCount:     &count,
		IntervalUnit:      intervalUnitMonth,
		BillingAnchorDate: &anchor,
	}

	_, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-02-01"))
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
	anchor := mustDate(t, "2024-02-29")
	count := 1
	draft := billingDraft{
		BillingType:       billingTypeRecurring,
		RecurrenceType:    recurrenceTypeInterval,
		IntervalCount:     &count,
		IntervalUnit:      intervalUnitYear,
		BillingAnchorDate: &anchor,
	}

	_, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-03-01"))
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

func TestNormalizeBillingDraft_MonthlySpecificDayClamp(t *testing.T) {
	anchor := mustDate(t, "2025-03-01")
	day := 31
	draft := billingDraft{
		BillingType:       billingTypeRecurring,
		RecurrenceType:    recurrenceTypeMonthlyDate,
		BillingAnchorDate: &anchor,
		MonthlyDay:        &day,
	}

	_, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-04-01"))
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2025-04-30"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
}

func TestNormalizeBillingDraft_YearlySpecificLeapDayClamp(t *testing.T) {
	anchor := mustDate(t, "2025-01-01")
	month := 2
	day := 29
	draft := billingDraft{
		BillingType:       billingTypeRecurring,
		RecurrenceType:    recurrenceTypeYearlyDate,
		BillingAnchorDate: &anchor,
		YearlyMonth:       &month,
		YearlyDay:         &day,
	}

	_, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-02-01"))
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

func TestNormalizeBillingDraft_TrialThenRecurring(t *testing.T) {
	anchor := mustDate(t, "2025-02-01")
	trialStart := mustDate(t, "2025-02-01")
	trialEnd := mustDate(t, "2025-02-15")
	count := 1
	draft := billingDraft{
		BillingType:       billingTypeRecurring,
		RecurrenceType:    recurrenceTypeInterval,
		IntervalCount:     &count,
		IntervalUnit:      intervalUnitMonth,
		BillingAnchorDate: &anchor,
		TrialEnabled:      true,
		TrialStartDate:    &trialStart,
		TrialEndDate:      &trialEnd,
	}

	normalized, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-02-10"))
	if err != nil {
		t.Fatalf("normalizeBillingDraft() error = %v", err)
	}
	if !normalized.TrialEnabled {
		t.Fatal("trial should remain enabled")
	}
	if next == nil {
		t.Fatal("next billing date should not be nil")
	}
	if got, want := next.Format("2006-01-02"), "2025-03-01"; got != want {
		t.Fatalf("next billing date = %s, want %s", got, want)
	}
}

func TestNormalizeBillingDraft_OneTimeAndLifetimeBehaveSame(t *testing.T) {
	purchaseDate := mustDate(t, "2025-02-20")

	testCases := []string{billingTypeOneTime, billingTypeLifetime}
	for _, billingType := range testCases {
		billingType := billingType
		t.Run(billingType, func(t *testing.T) {
			draft := billingDraft{
				BillingType:       billingType,
				BillingAnchorDate: &purchaseDate,
			}

			normalized, next, err := normalizeBillingDraft(draft, mustDate(t, "2025-02-20"))
			if err != nil {
				t.Fatalf("normalizeBillingDraft() error = %v", err)
			}
			if normalized.BillingAnchorDate == nil {
				t.Fatal("billing anchor date should be preserved")
			}
			if normalized.BillingType != billingTypeOneTime {
				t.Fatalf("billing type = %s, want %s", normalized.BillingType, billingTypeOneTime)
			}
			if got, want := normalized.BillingAnchorDate.Format("2006-01-02"), "2025-02-20"; got != want {
				t.Fatalf("billing anchor date = %s, want %s", got, want)
			}
			if next != nil {
				t.Fatalf("next billing date should be nil for %s", billingType)
			}
		})
	}
}
