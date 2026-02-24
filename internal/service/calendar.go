package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type CalendarService struct {
	DB *gorm.DB
}

func NewCalendarService(db *gorm.DB) *CalendarService {
	return &CalendarService{DB: db}
}

func (s *CalendarService) GenerateToken(userID uint, name string) (*model.CalendarToken, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	token := hex.EncodeToString(b)

	ct := model.CalendarToken{
		UserID:    userID,
		Token:     token,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.DB.Create(&ct).Error; err != nil {
		return nil, err
	}
	return &ct, nil
}

func (s *CalendarService) ListTokens(userID uint) ([]model.CalendarToken, error) {
	var tokens []model.CalendarToken
	if err := s.DB.Where("user_id = ?", userID).Order("created_at ASC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *CalendarService) DeleteToken(userID uint, tokenID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", tokenID, userID).Delete(&model.CalendarToken{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("token not found")
	}
	return nil
}

func (s *CalendarService) ValidateToken(token string) (uint, error) {
	var ct model.CalendarToken
	if err := s.DB.Where("token = ?", token).First(&ct).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("invalid token")
		}
		return 0, err
	}
	return ct.UserID, nil
}

func (s *CalendarService) GetSubscriptionsForCalendar(userID uint) ([]model.Subscription, error) {
	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND enabled = ? AND next_billing_date IS NOT NULL", userID, true).
		Order("next_billing_date ASC").
		Find(&subs).Error; err != nil {
		return nil, err
	}
	return subs, nil
}

func (s *CalendarService) GenerateICalFeed(userID uint) (string, error) {
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
		summary := fmt.Sprintf("%s - %.2f %s", sub.Name, sub.Amount, sub.Currency)

		sb.WriteString("BEGIN:VEVENT" + crlf)
		sb.WriteString(icalFold(fmt.Sprintf("UID:subdux-sub-%d@subdux", sub.ID)) + crlf)
		sb.WriteString(icalFold("DTSTART;VALUE=DATE:" + dateStr) + crlf)
		sb.WriteString(icalFold("DTEND;VALUE=DATE:" + dateStr) + crlf)
		sb.WriteString(icalFold("SUMMARY:" + icalEscape(summary)) + crlf)

		if sub.Notes != "" {
			sb.WriteString(icalFold("DESCRIPTION:" + icalEscape(sub.Notes)) + crlf)
		}

		if sub.BillingType == billingTypeRecurring && isRecurringScheduleValid(sub) {
			rrule := buildRRule(sub)
			if rrule != "" {
				sb.WriteString(icalFold("RRULE:" + rrule) + crlf)
			}
		}

		sb.WriteString("END:VEVENT" + crlf)
	}

	sb.WriteString("END:VCALENDAR" + crlf)
	return sb.String(), nil
}

func buildRRule(sub model.Subscription) string {
	switch sub.RecurrenceType {
	case recurrenceTypeInterval:
		if sub.IntervalCount == nil {
			return ""
		}
		count := *sub.IntervalCount
		switch sub.IntervalUnit {
		case intervalUnitDay:
			return fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", count)
		case intervalUnitWeek:
			return fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d", count)
		case intervalUnitMonth:
			return fmt.Sprintf("FREQ=MONTHLY;INTERVAL=%d", count)
		case intervalUnitYear:
			return fmt.Sprintf("FREQ=YEARLY;INTERVAL=%d", count)
		}
	case recurrenceTypeMonthlyDate:
		if sub.MonthlyDay == nil {
			return ""
		}
		return fmt.Sprintf("FREQ=MONTHLY;BYMONTHDAY=%d", *sub.MonthlyDay)
	case recurrenceTypeYearlyDate:
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
			octets = 1 // the leading space counts
		}
		sb.WriteRune(r)
		octets += len(encoded)
	}
	return sb.String()
}
