package subscription

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func autoAdvanceRecurringNextBillingDatesForUser(db *gorm.DB, userID uint, referenceDate time.Time) error {
	today := normalizeDateUTC(referenceDate)

	var subs []model.Subscription
	if err := db.Where(
		"user_id = ? AND status = ? AND renewal_mode = ? AND billing_type = ? AND next_billing_date IS NOT NULL AND next_billing_date < ?",
		userID,
		subscriptionStatusActive,
		renewalModeAutoRenew,
		billingTypeRecurring,
		today,
	).Find(&subs).Error; err != nil {
		return err
	}

	for i := range subs {
		sub := &subs[i]
		nextBillingDate, changed := nextRecurringBillingDateOnOrAfter(sub, today)
		if !changed || nextBillingDate == nil {
			continue
		}
		if err := db.Model(&model.Subscription{}).
			Where("id = ? AND user_id = ?", sub.ID, userID).
			Update("next_billing_date", *nextBillingDate).Error; err != nil {
			return err
		}
	}

	return nil
}

func nextRecurringBillingDateOnOrAfter(sub *model.Subscription, referenceDate time.Time) (*time.Time, bool) {
	if sub == nil || sub.BillingType != billingTypeRecurring || sub.NextBillingDate == nil {
		return nil, false
	}

	today := normalizeDateUTC(referenceDate)
	current := normalizeDateUTC(*sub.NextBillingDate)
	if !current.Before(today) {
		return nil, false
	}

	var next time.Time
	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		if sub.IntervalCount == nil || *sub.IntervalCount < 1 || !isValidIntervalUnit(sub.IntervalUnit) {
			return nil, false
		}
		next = nextIntervalOccurrence(current, today, *sub.IntervalCount, sub.IntervalUnit)
	case recurrenceTypeMonthlyDate:
		if sub.MonthlyDay == nil || *sub.MonthlyDay < 1 || *sub.MonthlyDay > 31 {
			return nil, false
		}
		next = nextMonthlyDayOccurrence(today, *sub.MonthlyDay)
	case recurrenceTypeYearlyDate:
		if sub.YearlyMonth == nil || *sub.YearlyMonth < 1 || *sub.YearlyMonth > 12 || sub.YearlyDay == nil || *sub.YearlyDay < 1 || *sub.YearlyDay > 31 {
			return nil, false
		}
		next = nextYearlyDateOccurrence(today, *sub.YearlyMonth, *sub.YearlyDay)
	default:
		return nil, false
	}

	next = normalizeDateUTC(next)
	if !next.After(current) {
		return nil, false
	}

	return &next, true
}

func nextIntervalOccurrence(anchor, from time.Time, intervalCount int, intervalUnit string) time.Time {
	if intervalCount < 1 || intervalCount > MaxIntervalCount || !isValidIntervalUnit(intervalUnit) || anchor.Year() < 1 || anchor.Year() > 9999 || from.Year() < 1 || from.Year() > 9999 {
		return time.Time{}
	}
	anchor = normalizeDateUTC(anchor)
	from = normalizeDateUTC(from)
	if !from.After(anchor) {
		return anchor
	}

	current := anchor
	// Jump close to the target before taking at most one final step. This
	// avoids work proportional to the age of a daily subscription.
	switch intervalUnit {
	case intervalUnitDay, intervalUnitWeek:
		days := intervalCount
		if intervalUnit == intervalUnitWeek {
			days *= 7
		}
		steps := (from.Unix() - anchor.Unix()) / (86400 * int64(days))
		current = anchor.AddDate(0, 0, int(steps)*days)
	case intervalUnitMonth:
		months := (from.Year()-anchor.Year())*12 + int(from.Month()-anchor.Month())
		current = addMonthsPreservePreferredDay(anchor, months/intervalCount*intervalCount, anchor.Day())
	case intervalUnitYear:
		years := (from.Year() - anchor.Year()) / intervalCount * intervalCount
		current = addYearsPreservePreferredDate(anchor, years, anchor.Month(), anchor.Day())
	}
	previous := current
	switch intervalUnit {
	case intervalUnitDay:
		if current.Before(from) {
			current = current.AddDate(0, 0, intervalCount)
		}
	case intervalUnitWeek:
		if current.Before(from) {
			current = current.AddDate(0, 0, intervalCount*7)
		}
	case intervalUnitMonth:
		preferredDay := anchor.Day()
		if current.Before(from) {
			current = addMonthsPreservePreferredDay(current, intervalCount, preferredDay)
		}
	case intervalUnitYear:
		preferredDay := anchor.Day()
		preferredMonth := anchor.Month()
		if current.Before(from) {
			current = addYearsPreservePreferredDate(current, intervalCount, preferredMonth, preferredDay)
		}
	}
	if current.Before(from) || current.Before(previous) || current.Year() < 1 || current.Year() > 9999 {
		return time.Time{}
	}
	return current
}

func nextMonthlyDayOccurrence(from time.Time, day int) time.Time {
	from = normalizeDateUTC(from)
	year, month, _ := from.Date()

	candidate := buildDate(year, month, day)
	if candidate.Before(from) {
		candidate = addMonthsPreservePreferredDay(candidate, 1, day)
	}

	return candidate
}

func nextYearlyDateOccurrence(from time.Time, month int, day int) time.Time {
	from = normalizeDateUTC(from)
	year := from.Year()

	candidate := buildDate(year, time.Month(month), day)
	if candidate.Before(from) {
		candidate = buildDate(year+1, time.Month(month), day)
	}

	return candidate
}

func addMonthsPreservePreferredDay(base time.Time, months int, preferredDay int) time.Time {
	base = normalizeDateUTC(base)
	year, month, _ := base.Date()
	targetMonthIndex := int(month) - 1 + months
	targetYear := year + targetMonthIndex/12
	targetMonth := targetMonthIndex % 12
	if targetMonth < 0 {
		targetMonth += 12
		targetYear--
	}

	return buildDate(targetYear, time.Month(targetMonth+1), preferredDay)
}

func addYearsPreservePreferredDate(base time.Time, years int, preferredMonth time.Month, preferredDay int) time.Time {
	base = normalizeDateUTC(base)
	targetYear := base.Year() + years
	return buildDate(targetYear, preferredMonth, preferredDay)
}

func buildDate(year int, month time.Month, preferredDay int) time.Time {
	day := clampDay(year, month, preferredDay)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func clampDay(year int, month time.Month, day int) int {
	if day < 1 {
		return 1
	}
	maxDay := daysInMonth(year, month)
	if day > maxDay {
		return maxDay
	}
	return day
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}
