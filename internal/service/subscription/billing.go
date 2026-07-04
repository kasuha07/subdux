package subscription

import (
	"strings"
	"time"
)

func normalizeBillingDraft(draft billingDraft) (billingDraft, *time.Time, error) {
	draft.BillingType = normalizeBillingType(draft.BillingType)
	if draft.BillingType == "" {
		draft.BillingType = billingTypeRecurring
	}

	if draft.NextBillingDate == nil {
		return draft, nil, ErrNextBillingDateRequiredRecurring
	}

	switch draft.BillingType {
	case billingTypeRecurring:
		draft.RecurrenceType = normalizeRecurrenceType(draft.RecurrenceType)
		if draft.RecurrenceType == "" {
			draft.RecurrenceType = recurrenceTypeInterval
		}

		nextBillingDate := normalizeDateUTC(*draft.NextBillingDate)
		draft.NextBillingDate = &nextBillingDate

		switch draft.RecurrenceType {
		case recurrenceTypeInterval:
			if draft.IntervalCount == nil || *draft.IntervalCount < 1 {
				return draft, nil, ErrIntervalCountTooLow
			}
			intervalCount := *draft.IntervalCount
			draft.IntervalCount = &intervalCount

			draft.IntervalUnit = normalizeIntervalUnit(draft.IntervalUnit)
			if !isValidIntervalUnit(draft.IntervalUnit) {
				return draft, nil, ErrIntervalUnitInvalid
			}

			draft.MonthlyDay = nil
			draft.YearlyMonth = nil
			draft.YearlyDay = nil

			return draft, &nextBillingDate, nil
		case recurrenceTypeMonthlyDate:
			if draft.MonthlyDay == nil || *draft.MonthlyDay < 1 || *draft.MonthlyDay > 31 {
				return draft, nil, ErrMonthlyDayInvalid
			}
			monthlyDay := *draft.MonthlyDay
			draft.MonthlyDay = &monthlyDay
			draft.IntervalCount = nil
			draft.IntervalUnit = ""
			draft.YearlyMonth = nil
			draft.YearlyDay = nil

			return draft, &nextBillingDate, nil
		case recurrenceTypeYearlyDate:
			if draft.YearlyMonth == nil || *draft.YearlyMonth < 1 || *draft.YearlyMonth > 12 {
				return draft, nil, ErrYearlyMonthInvalid
			}
			if draft.YearlyDay == nil || *draft.YearlyDay < 1 || *draft.YearlyDay > 31 {
				return draft, nil, ErrYearlyDayInvalid
			}

			yearlyMonth := *draft.YearlyMonth
			yearlyDay := *draft.YearlyDay
			draft.YearlyMonth = &yearlyMonth
			draft.YearlyDay = &yearlyDay
			draft.IntervalCount = nil
			draft.IntervalUnit = ""
			draft.MonthlyDay = nil

			return draft, &nextBillingDate, nil
		default:
			return draft, nil, ErrRecurrenceTypeInvalid
		}
	default:
		return draft, nil, ErrBillingTypeMustBeRecurring
	}
}

func NormalizeBillingDraft(draft BillingDraft) (BillingDraft, *time.Time, error) {
	return normalizeBillingDraft(draft)
}

func normalizeBillingType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeRecurrenceType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeIntervalUnit(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidIntervalUnit(unit string) bool {
	switch unit {
	case intervalUnitDay, intervalUnitWeek, intervalUnitMonth, intervalUnitYear:
		return true
	default:
		return false
	}
}

func parseOptionalDateString(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, ErrInvalidDateFormat
	}

	normalized := normalizeDateUTC(parsed)
	return &normalized, nil
}

func normalizeDateUTC(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
