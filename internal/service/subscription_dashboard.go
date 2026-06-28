package service

import (
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func (s *SubscriptionService) GetDashboardSummary(userID uint, targetCurrency string, converter CurrencyConverter) (*DashboardSummary, error) {
	now := pkg.NowInSystemTimezone()

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ?", userID, subscriptionStatusActive).Find(&subs).Error; err != nil {
		return nil, err
	}

	return computeDashboardSummary(presentActiveSubscriptions(subs, now), targetCurrency, converter, now), nil
}

// SubscriptionsWithSummary returns a user's full subscription list (ordered as
// the list endpoint expects) together with the dashboard summary, reading the
// subscriptions table once. It backs the dashboard bootstrap endpoint. Lifecycle
// is advanced in memory for presentation — reads never write — and the summary
// is derived from the active subset after that advance, so an overdue
// subscription the background sweep has not yet ended is excluded just as it
// would be once persisted.
func (s *SubscriptionService) SubscriptionsWithSummary(
	userID uint,
	targetCurrency string,
	converter CurrencyConverter,
) ([]model.Subscription, *DashboardSummary, error) {
	now := pkg.NowInSystemTimezone()

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ?", userID).
		Order("next_billing_date IS NULL ASC").
		Order("next_billing_date ASC").
		Order("id ASC").
		Find(&subs).Error; err != nil {
		return nil, nil, err
	}
	for i := range subs {
		presentSubscriptionForResponse(&subs[i], now)
	}

	activeSubs := make([]model.Subscription, 0, len(subs))
	for _, sub := range subs {
		if subscriptionIsActive(sub) {
			activeSubs = append(activeSubs, sub)
		}
	}

	return subs, computeDashboardSummary(activeSubs, targetCurrency, converter, now), nil
}

// presentActiveSubscriptions advances each subscription's lifecycle in memory
// and returns only those still active as of now. It lets read paths that query
// WHERE status = active reproduce the post-reconcile active set without writing:
// an overdue manual-renew or cancel-at-period-end subscription is dropped, and
// an auto-renew subscription is returned with its billing date rolled forward.
func presentActiveSubscriptions(subs []model.Subscription, now time.Time) []model.Subscription {
	active := make([]model.Subscription, 0, len(subs))
	for i := range subs {
		presentSubscriptionForResponse(&subs[i], now)
		if subscriptionIsActive(subs[i]) {
			active = append(active, subs[i])
		}
	}
	return active
}

// computeDashboardSummary aggregates spend metrics from a set of active
// subscriptions. It performs no I/O so it can be reused by any caller that has
// already loaded the active subscriptions.
func computeDashboardSummary(subs []model.Subscription, targetCurrency string, converter CurrencyConverter, now time.Time) *DashboardSummary {
	if targetCurrency == "" {
		targetCurrency = "USD"
	}

	today := normalizeDateUTC(now)
	startOfThisMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	startOfNextMonth := startOfThisMonth.AddDate(0, 1, 0)

	var totalMonthly float64
	var committedMonthly float64
	var dueThisMonth float64
	for _, sub := range subs {
		amount := sub.Amount
		if converter != nil && sub.Currency != targetCurrency {
			amount = converter.Convert(amount, sub.Currency, targetCurrency)
		}

		factor := subscriptionMonthlyFactor(sub)
		if factor > 0 && subscriptionContributesToOngoingSpend(sub) {
			totalMonthly += amount * factor
			if normalizeRenewalMode(sub.RenewalMode) == renewalModeAutoRenew {
				committedMonthly += amount * factor
			}
		}

		occurrences := len(subscriptionChargeDatesInRange(sub, today, startOfNextMonth))
		if occurrences > 0 {
			dueThisMonth += amount * float64(occurrences)
		}
	}

	sevenDays := today.AddDate(0, 0, 7)
	var upcomingRenewalCount int64
	for _, sub := range subs {
		if !subscriptionHasFutureCharge(sub) || sub.NextBillingDate == nil {
			continue
		}

		nextBillingDate := normalizeDateUTC(*sub.NextBillingDate)
		if nextBillingDate.Before(today) || nextBillingDate.After(sevenDays) {
			continue
		}
		upcomingRenewalCount++
	}

	return &DashboardSummary{
		TotalMonthly:         totalMonthly,
		TotalYearly:          totalMonthly * 12,
		CommittedMonthly:     committedMonthly,
		CommittedYearly:      committedMonthly * 12,
		DueThisMonth:         dueThisMonth,
		ActiveCount:          int64(len(subs)),
		UpcomingRenewalCount: upcomingRenewalCount,
		Currency:             targetCurrency,
	}
}

func subscriptionContributesToOngoingSpend(sub model.Subscription) bool {
	return normalizeStatus(sub.Status) == subscriptionStatusActive &&
		sub.BillingType == billingTypeRecurring &&
		normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd
}

func subscriptionHasFutureCharge(sub model.Subscription) bool {
	return normalizeStatus(sub.Status) == subscriptionStatusActive &&
		sub.BillingType == billingTypeRecurring &&
		normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd
}

func subscriptionChargeDatesInRange(sub model.Subscription, startInclusive, endExclusive time.Time) []time.Time {
	if !subscriptionHasFutureCharge(sub) {
		return nil
	}

	startInclusive = normalizeDateUTC(startInclusive)
	endExclusive = normalizeDateUTC(endExclusive)
	if !startInclusive.Before(endExclusive) {
		return nil
	}
	if sub.NextBillingDate == nil {
		return nil
	}

	current := normalizeDateUTC(*sub.NextBillingDate)
	if normalizeRenewalMode(sub.RenewalMode) != renewalModeAutoRenew {
		if current.Before(startInclusive) || !current.Before(endExclusive) {
			return nil
		}
		return []time.Time{current}
	}
	if !isRecurringScheduleValid(sub) {
		return nil
	}

	if current.Before(startInclusive) {
		next, ok := nextRecurringOccurrenceOnOrAfter(sub, current, startInclusive)
		if !ok {
			return nil
		}
		current = next
	}

	if !current.Before(endExclusive) {
		return nil
	}

	var dates []time.Time
	for current.Before(endExclusive) {
		if !current.Before(startInclusive) {
			dates = append(dates, current)
		}

		next, ok := nextRecurringOccurrenceAfter(sub, current)
		if !ok || !next.After(current) {
			break
		}
		current = next
	}

	return dates
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
