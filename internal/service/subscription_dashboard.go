package service

import (
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func (s *SubscriptionService) GetDashboardSummary(userID uint, targetCurrency string, converter CurrencyConverter) (*DashboardSummary, error) {
	now := time.Now().In(pkg.GetSystemTimezone())
	if err := autoAdvanceRecurringNextBillingDatesForUser(s.DB, userID, now); err != nil {
		return nil, err
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&subs).Error; err != nil {
		return nil, err
	}

	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	today := normalizeDateUTC(now)
	startOfThisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfThisMonth.AddDate(0, 1, 0)

	var totalMonthly float64
	var dueThisMonth float64
	for _, sub := range subs {
		amount := sub.Amount
		if converter != nil && sub.Currency != targetCurrency {
			amount = converter.Convert(amount, sub.Currency, targetCurrency)
		}

		factor := subscriptionMonthlyFactor(sub)
		if factor <= 0 {
			occurrences := countSubscriptionOccurrencesInRange(sub, today, startOfNextMonth)
			if occurrences > 0 {
				dueThisMonth += amount * float64(occurrences)
			}
			continue
		}

		totalMonthly += amount * factor

		occurrences := countSubscriptionOccurrencesInRange(sub, today, startOfNextMonth)
		if occurrences > 0 {
			dueThisMonth += amount * float64(occurrences)
		}
	}

	sevenDays := today.AddDate(0, 0, 7)
	var upcomingRenewalCount int64
	if err := s.DB.Model(&model.Subscription{}).Where(
		"user_id = ? AND enabled = ? AND billing_type = ? AND next_billing_date IS NOT NULL AND next_billing_date >= ? AND next_billing_date <= ?",
		userID,
		true,
		billingTypeRecurring,
		today,
		sevenDays,
	).Count(&upcomingRenewalCount).Error; err != nil {
		return nil, err
	}

	return &DashboardSummary{
		TotalMonthly:         totalMonthly,
		TotalYearly:          totalMonthly * 12,
		DueThisMonth:         dueThisMonth,
		EnabledCount:         int64(len(subs)),
		UpcomingRenewalCount: upcomingRenewalCount,
		Currency:             targetCurrency,
	}, nil
}

func countSubscriptionOccurrencesInRange(sub model.Subscription, startInclusive, endExclusive time.Time) int {
	startInclusive = normalizeDateUTC(startInclusive)
	endExclusive = normalizeDateUTC(endExclusive)
	if !startInclusive.Before(endExclusive) {
		return 0
	}
	if sub.NextBillingDate == nil {
		return 0
	}

	current := normalizeDateUTC(*sub.NextBillingDate)
	if sub.BillingType != billingTypeRecurring {
		if current.Before(startInclusive) || !current.Before(endExclusive) {
			return 0
		}
		return 1
	}
	if !isRecurringScheduleValid(sub) {
		return 0
	}

	if current.Before(startInclusive) {
		next, ok := nextRecurringOccurrenceOnOrAfter(sub, current, startInclusive)
		if !ok {
			return 0
		}
		current = next
	}

	if !current.Before(endExclusive) {
		return 0
	}

	occurrences := 0
	for current.Before(endExclusive) {
		if !current.Before(startInclusive) {
			occurrences++
		}

		next, ok := nextRecurringOccurrenceAfter(sub, current)
		if !ok || !next.After(current) {
			break
		}
		current = next
	}

	return occurrences
}

func nextRecurringOccurrenceOnOrAfter(sub model.Subscription, anchor, from time.Time) (time.Time, bool) {
	anchor = normalizeDateUTC(anchor)
	from = normalizeDateUTC(from)

	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		return nextIntervalOccurrence(anchor, from, *sub.IntervalCount, sub.IntervalUnit), true
	case recurrenceTypeMonthlyDate:
		return nextMonthlyDayOccurrence(from, *sub.MonthlyDay), true
	case recurrenceTypeYearlyDate:
		return nextYearlyDateOccurrence(from, *sub.YearlyMonth, *sub.YearlyDay), true
	default:
		return time.Time{}, false
	}
}

func isRecurringScheduleValid(sub model.Subscription) bool {
	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		return sub.IntervalCount != nil && *sub.IntervalCount >= 1 && isValidIntervalUnit(sub.IntervalUnit)
	case recurrenceTypeMonthlyDate:
		return sub.MonthlyDay != nil && *sub.MonthlyDay >= 1 && *sub.MonthlyDay <= 31
	case recurrenceTypeYearlyDate:
		return sub.YearlyMonth != nil && *sub.YearlyMonth >= 1 && *sub.YearlyMonth <= 12 &&
			sub.YearlyDay != nil && *sub.YearlyDay >= 1 && *sub.YearlyDay <= 31
	default:
		return false
	}
}

func nextRecurringOccurrenceAfter(sub model.Subscription, current time.Time) (time.Time, bool) {
	nextFrom := normalizeDateUTC(current).AddDate(0, 0, 1)
	return nextRecurringOccurrenceOnOrAfter(sub, current, nextFrom)
}

func subscriptionMonthlyFactor(sub model.Subscription) float64 {
	if sub.BillingType != billingTypeRecurring {
		return 0
	}

	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		if sub.IntervalCount == nil || *sub.IntervalCount <= 0 {
			return 0
		}
		count := float64(*sub.IntervalCount)
		switch sub.IntervalUnit {
		case intervalUnitDay:
			return 30.436875 / count
		case intervalUnitWeek:
			return 4.348125 / count
		case intervalUnitMonth:
			return 1 / count
		case intervalUnitYear:
			return 1 / (12 * count)
		default:
			return 0
		}
	case recurrenceTypeMonthlyDate:
		return 1
	case recurrenceTypeYearlyDate:
		return 1.0 / 12.0
	default:
		return 0
	}
}
