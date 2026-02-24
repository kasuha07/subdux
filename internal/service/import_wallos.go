package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type ImportService struct {
	DB *gorm.DB
}

func NewImportService(db *gorm.DB) *ImportService {
	return &ImportService{DB: db}
}

type WallosSubscription struct {
	Name             string `json:"Name"`
	PaymentCycle     string `json:"Payment Cycle"`
	NextPayment      string `json:"Next Payment"`
	Renewal          string `json:"Renewal"`
	Category         string `json:"Category"`
	PaymentMethod    string `json:"Payment Method"`
	PaidBy           string `json:"Paid By"`
	Price            string `json:"Price"`
	Notes            string `json:"Notes"`
	URL              string `json:"URL"`
	State            string `json:"State"`
	Notifications    string `json:"Notifications"`
	CancellationDate string `json:"Cancellation Date"`
	Active           string `json:"Active"`
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

var currencySymbols = map[string]string{
	"$": "USD", "€": "EUR", "£": "GBP", "¥": "JPY", "￥": "CNY",
	"₩": "KRW", "₹": "INR", "₽": "RUB", "₺": "TRY", "₴": "UAH",
	"₫": "VND", "₱": "PHP", "₿": "BTC", "฿": "THB", "₪": "ILS",
	"R$": "BRL", "zł": "PLN", "Kč": "CZK", "kr": "SEK", "Fr": "CHF",
	"RM": "MYR", "Rp": "IDR", "Rs": "INR", "DH": "MAD", "DA": "DZD",
	"DT": "TND", "LD": "LYD", "S$": "SGD", "A$": "AUD", "C$": "CAD",
	"NZ$": "NZD", "HK$": "HKD", "NT$": "TWD", "MX$": "MXN",
}

var knownCurrencies = []string{
	"USD", "EUR", "GBP", "JPY", "CNY", "AUD", "CAD", "CHF", "HKD", "SGD",
	"SEK", "NOK", "DKK", "NZD", "MXN", "BRL", "INR", "RUB", "ZAR", "KRW",
	"TRY", "PLN", "THB", "IDR", "MYR", "PHP", "CZK", "HUF", "RON", "BGN",
	"HRK", "ISK", "ILS", "AED", "SAR", "QAR", "KWD", "BHD", "OMR", "JOD",
	"EGP", "NGN", "KES", "GHS", "MAD", "TND", "DZD", "LYD", "PKR", "BDT",
	"LKR", "NPR", "MMK", "VND", "TWD", "CLP", "COP", "PEN", "ARS", "UYU",
}

var currencyRe = regexp.MustCompile(`[A-Z]{3}`)

func extractCurrencyAndAmount(price string) (float64, string) {
	currency := "USD"
	found := false

	// Try symbol matching first (longer symbols first to match "NZ$" before "$")
	for sym, code := range currencySymbols {
		if strings.Contains(price, sym) {
			currency = code
			found = true
			break
		}
	}

	// Then try 3-letter currency codes
	if !found {
		upper := strings.ToUpper(price)
		for _, code := range knownCurrencies {
			if strings.Contains(upper, code) {
				currency = code
				found = true
				break
			}
		}
		if !found {
			if m := currencyRe.FindString(upper); m != "" {
				currency = m
			}
		}
	}

	// Strip everything except digits, dots, commas, minus
	cleaned := regexp.MustCompile(`[^\d.,-]`).ReplaceAllString(price, "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	amount, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, currency
	}
	return amount, currency
}

var everyNRe = regexp.MustCompile(`(?i)^every\s+(\d+)\s+(day|week|month|year)s?$`)

func mapPaymentCycle(cycle string) (billingType, recurrenceType, intervalUnit string, intervalCount int) {
	lower := strings.ToLower(strings.TrimSpace(cycle))
	switch lower {
	case "monthly":
		return "recurring", "interval", "month", 1
	case "yearly", "annual":
		return "recurring", "interval", "year", 1
	case "weekly":
		return "recurring", "interval", "week", 1
	case "daily":
		return "recurring", "interval", "day", 1
	case "biweekly", "bi-weekly":
		return "recurring", "interval", "week", 2
	case "quarterly":
		return "recurring", "interval", "month", 3
	case "semiannual", "semi-annual":
		return "recurring", "interval", "month", 6
	}

	// Handle "Every N <unit>" format (e.g. "Every 3 Months")
	if m := everyNRe.FindStringSubmatch(cycle); m != nil {
		n, _ := strconv.Atoi(m[1])
		unit := strings.ToLower(m[2])
		if n > 0 {
			return "recurring", "interval", unit, n
		}
	}

	return "one_time", "", "", 0
}

var dateFormats = []string{
	"2006-01-02",
	"01/02/2006",
	"2006/01/02",
	"02-01-2006",
}

func parseDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	for _, layout := range dateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func parseEnabled(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower == "enabled" || lower == "yes" || lower == "1" || lower == "true"
}

func (s *ImportService) ImportFromWallos(userID uint, data []WallosSubscription) (*ImportResult, error) {
	result := &ImportResult{Errors: []string{}}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range data {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				result.Errors = append(result.Errors, "skipped item with empty name")
				result.Skipped++
				continue
			}

			amount, currency := extractCurrencyAndAmount(item.Price)
			billingType, recurrenceType, intervalUnit, intervalCount := mapPaymentCycle(item.PaymentCycle)
			nextBilling := parseDate(item.NextPayment)
			enabled := parseEnabled(item.Active)

			// Deduplicate by name + amount + currency + billing_type + next_billing_date
			query := tx.Model(&model.Subscription{}).
				Where("user_id = ? AND name = ? AND amount = ? AND currency = ? AND billing_type = ?",
					userID, name, amount, currency, billingType)
			if nextBilling != nil {
				query = query.Where("next_billing_date = ?", *nextBilling)
			} else {
				query = query.Where("next_billing_date IS NULL")
			}
			var count int64
			if err := query.Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				result.Skipped++
				continue
			}

			notifyEnabled := parseEnabled(item.Notifications)
			sub := model.Subscription{
				UserID:        userID,
				Name:          name,
				Amount:        amount,
				Currency:      currency,
				Enabled:       enabled,
				BillingType:   billingType,
				Category:      item.Category,
				URL:           item.URL,
				Notes:         item.Notes,
				NotifyEnabled: &notifyEnabled,
			}

			if billingType == "recurring" {
				sub.RecurrenceType = recurrenceType
				sub.IntervalUnit = intervalUnit
				sub.IntervalCount = &intervalCount
			}

			if nextBilling != nil {
				sub.NextBillingDate = nextBilling
			}

			if err := tx.Create(&sub).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to import %q: %v", name, err))
				continue
			}

			// Force update enabled field to handle GORM default:true zero-value issue
			if !enabled {
				if err := tx.Model(&sub).Update("enabled", false).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to update enabled for %q: %v", name, err))
					continue
				}
			}

			result.Imported++
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
