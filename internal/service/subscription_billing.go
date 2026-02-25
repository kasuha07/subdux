package service

import (
	"errors"
	"strings"
	"time"
)

func normalizeBillingDraft(draft billingDraft) (billingDraft, *time.Time, error) {
	draft.BillingType = normalizeBillingType(draft.BillingType)
	if draft.BillingType == "" {
		draft.BillingType = billingTypeRecurring
	}

	if draft.NextBillingDate == nil {
		switch draft.BillingType {
		case billingTypeRecurring:
			return draft, nil, errors.New("next_billing_date is required for recurring subscriptions")
		case billingTypeOneTime:
			return draft, nil, errors.New("next_billing_date is required for one-time subscriptions")
		}
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
				return draft, nil, errors.New("interval_count must be at least 1 for interval recurrence")
			}
			intervalCount := *draft.IntervalCount
			draft.IntervalCount = &intervalCount

			draft.IntervalUnit = normalizeIntervalUnit(draft.IntervalUnit)
			if !isValidIntervalUnit(draft.IntervalUnit) {
				return draft, nil, errors.New("interval_unit must be one of: day, week, month, year")
			}

			draft.MonthlyDay = nil
			draft.YearlyMonth = nil
			draft.YearlyDay = nil

			return draft, &nextBillingDate, nil
		case recurrenceTypeMonthlyDate:
			if draft.MonthlyDay == nil || *draft.MonthlyDay < 1 || *draft.MonthlyDay > 31 {
				return draft, nil, errors.New("monthly_day must be between 1 and 31 for monthly date recurrence")
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
				return draft, nil, errors.New("yearly_month must be between 1 and 12 for yearly date recurrence")
			}
			if draft.YearlyDay == nil || *draft.YearlyDay < 1 || *draft.YearlyDay > 31 {
				return draft, nil, errors.New("yearly_day must be between 1 and 31 for yearly date recurrence")
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
			return draft, nil, errors.New("recurrence_type must be one of: interval, monthly_date, yearly_date")
		}
	case billingTypeOneTime:
		nextBillingDate := normalizeDateUTC(*draft.NextBillingDate)
		draft.NextBillingDate = &nextBillingDate
		draft.RecurrenceType = ""
		draft.IntervalCount = nil
		draft.IntervalUnit = ""
		draft.MonthlyDay = nil
		draft.YearlyMonth = nil
		draft.YearlyDay = nil
		return draft, &nextBillingDate, nil
	default:
		return draft, nil, errors.New("billing_type must be one of: recurring, one_time")
	}
}

func normalizeBillingType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == billingTypeLifetime {
		return billingTypeOneTime
	}
	return normalized
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
		return nil, errors.New("invalid date format, expected YYYY-MM-DD")
	}

	normalized := normalizeDateUTC(parsed)
	return &normalized, nil
}

func normalizeDateUTC(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
