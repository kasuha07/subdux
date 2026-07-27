package calendar

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/money"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/kasuha07/subdux/internal/service/userstatus"
	"gorm.io/gorm"
)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.DB != nil {
		clone.DB = s.DB.WithContext(ctx)
	}
	return &clone
}

func (s *Service) GenerateToken(userID uint, name string) (*model.CalendarToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(b)
	tokenHash := hashCalendarToken(token)

	ct := model.CalendarToken{
		UserID:    userID,
		Token:     tokenHash,
		Name:      name,
		CreatedAt: pkg.NowUTC(),
	}
	if err := s.DB.Create(&ct).Error; err != nil {
		return nil, err
	}
	ct.Token = token
	return &ct, nil
}

func (s *Service) ListTokens(userID uint) ([]model.CalendarToken, error) {
	var tokens []model.CalendarToken
	if err := s.DB.Where("user_id = ?", userID).Order("created_at ASC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	for i := range tokens {
		tokens[i].Token = ""
	}
	return tokens, nil
}

func (s *Service) DeleteToken(userID uint, tokenID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", tokenID, userID).Delete(&model.CalendarToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("token not found")
	}
	return nil
}

func (s *Service) ValidateToken(token string) (uint, error) {
	tokenHash := hashCalendarToken(token)

	var ct model.CalendarToken
	if err := s.DB.Where("token = ?", tokenHash).First(&ct).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, err
		}

		if err := s.DB.Where("token = ?", token).First(&ct).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, errors.New("invalid token")
			}
			return 0, err
		}

		if migrateErr := s.DB.Model(&ct).Update("token", tokenHash).Error; migrateErr != nil {
			return 0, migrateErr
		}
	}

	if err := userstatus.EnsureActive(s.DB, ct.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, userstatus.ErrUserNotActive) {
			return 0, errors.New("invalid token")
		}
		return 0, err
	}

	return ct.UserID, nil
}

func hashCalendarToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) GetSubscriptionsForCalendar(userID uint) ([]model.Subscription, error) {
	now := pkg.NowInSystemTimezone()

	var subs []model.Subscription
	if err := s.DB.Where(
		"user_id = ? AND status = ? AND renewal_mode != ? AND next_billing_date IS NOT NULL",
		userID,
		subscriptionservice.StatusActive,
		subscriptionservice.RenewalModeCancelAtPeriodEnd,
	).
		Order("next_billing_date ASC").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	// Present lifecycle in memory (no writes): auto-renew dates roll forward and
	// overdue manual-renew subscriptions drop out, matching what the calendar
	// would show once the background sweep persists those transitions.
	return subscriptionservice.PresentActiveSubscriptions(subs, now), nil
}

func (s *Service) GenerateICalFeed(userID uint) (string, error) {
	subs, err := s.GetSubscriptionsForCalendar(userID)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	crlf := "\r\n"

	sb.WriteString("BEGIN:VCALENDAR" + crlf)
	sb.WriteString("VERSION:2.0" + crlf)
	sb.WriteString("PRODID:-//Subdux//Calendar//EN" + crlf)
	sb.WriteString(icalFold("X-WR-CALNAME:Subdux Subscriptions") + crlf)
	sb.WriteString("CALSCALE:GREGORIAN" + crlf)
	sb.WriteString("METHOD:PUBLISH" + crlf)

	for _, sub := range subs {
		if sub.NextBillingDate == nil {
			continue
		}

		dateStr := sub.NextBillingDate.UTC().Format("20060102")
		// Format with the currency's own minor unit rather than a hardcoded two
		// decimals: JPY reads "1235", KWD reads "1.200".
		summary := fmt.Sprintf("%s - %s %s", sub.Name, money.Format(sub.Amount, sub.Currency), sub.Currency)

		sb.WriteString("BEGIN:VEVENT" + crlf)
		sb.WriteString(icalFold(fmt.Sprintf("UID:subdux-sub-%d@subdux", sub.ID)) + crlf)
		sb.WriteString(icalFold("DTSTART;VALUE=DATE:"+dateStr) + crlf)
		sb.WriteString(icalFold("DTEND;VALUE=DATE:"+dateStr) + crlf)
		sb.WriteString(icalFold("SUMMARY:"+icalEscape(summary)) + crlf)

		if sub.Notes != "" {
			sb.WriteString(icalFold("DESCRIPTION:"+icalEscape(sub.Notes)) + crlf)
		}

		if sub.BillingType == subscriptionservice.BillingTypeRecurring &&
			subscriptionservice.NormalizeRenewalMode(sub.RenewalMode) == subscriptionservice.RenewalModeAutoRenew &&
			subscriptionservice.IsRecurringScheduleValid(sub) {
			rrule := buildRRule(sub)
			if rrule != "" {
				sb.WriteString(icalFold("RRULE:"+rrule) + crlf)
			}
		}

		sb.WriteString("END:VEVENT" + crlf)
	}

	sb.WriteString("END:VCALENDAR" + crlf)
	return sb.String(), nil
}

func buildRRule(sub model.Subscription) string {
	switch sub.RecurrenceType {
	case subscriptionservice.RecurrenceTypeInterval:
		if sub.IntervalCount == nil {
			return ""
		}
		count := *sub.IntervalCount
		switch sub.IntervalUnit {
		case subscriptionservice.IntervalUnitDay:
			return fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", count)
		case subscriptionservice.IntervalUnitWeek:
			return fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d", count)
		case subscriptionservice.IntervalUnitMonth:
			return fmt.Sprintf("FREQ=MONTHLY;INTERVAL=%d", count)
		case subscriptionservice.IntervalUnitYear:
			return fmt.Sprintf("FREQ=YEARLY;INTERVAL=%d", count)
		}
	case subscriptionservice.RecurrenceTypeMonthlyDate:
		if sub.MonthlyDay == nil {
			return ""
		}
		return fmt.Sprintf("FREQ=MONTHLY;BYMONTHDAY=%d", *sub.MonthlyDay)
	case subscriptionservice.RecurrenceTypeYearlyDate:
		if sub.YearlyMonth == nil || sub.YearlyDay == nil {
			return ""
		}
		return fmt.Sprintf("FREQ=YEARLY;BYMONTH=%d;BYMONTHDAY=%d", *sub.YearlyMonth, *sub.YearlyDay)
	}
	return ""
}

// icalEscape escapes special characters in iCalendar text values.
func icalEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ";", `\;`)
	s = strings.ReplaceAll(s, ",", `\,`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// icalFold folds a content line per RFC 5545: lines longer than 75 octets
// are folded by inserting CRLF followed by a single space.
func icalFold(line string) string {
	const maxOctets = 75
	if len(line) <= maxOctets {
		return line
	}

	var sb strings.Builder
	octets := 0
	for _, r := range line {
		encoded := []byte(string(r))
		if octets+len(encoded) > maxOctets {
			sb.WriteString("\r\n ")
			octets = 1
		}
		sb.WriteRune(r)
		octets += len(encoded)
	}
	return sb.String()
}
