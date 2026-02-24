package service

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
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

type WallosImportRequest struct {
	Data    []WallosSubscription `json:"data"`
	Confirm bool                 `json:"confirm"`
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

type PreviewCurrencyChange struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
	IsNew  bool   `json:"is_new"`
}

type PreviewPaymentMethodChange struct {
	Name    string `json:"name"`
	IsNew   bool   `json:"is_new"`
	Matched string `json:"matched,omitempty"`
}

type PreviewCategoryChange struct {
	Name  string `json:"name"`
	IsNew bool   `json:"is_new"`
}

type PreviewSubscriptionChange struct {
	Name        string  `json:"name"`
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`
	BillingType string  `json:"billing_type"`
	Category    string  `json:"category,omitempty"`
	Skipped     bool    `json:"skipped"`
	SkipReason  string  `json:"skip_reason,omitempty"`
}

type ImportPreview struct {
	Currencies     []PreviewCurrencyChange      `json:"currencies"`
	PaymentMethods []PreviewPaymentMethodChange `json:"payment_methods"`
	Categories     []PreviewCategoryChange      `json:"categories"`
	Subscriptions  []PreviewSubscriptionChange  `json:"subscriptions"`
}

// WallosImportResponse is returned for both preview and confirm modes.
type WallosImportResponse struct {
	// Preview fields (returned when confirm=false)
	Preview *ImportPreview `json:"preview,omitempty"`
	// Result fields (returned when confirm=true)
	Result *ImportResult `json:"result,omitempty"`
}

const maxWallosImportItems = 5000

var ErrWallosImportTooLarge = errors.New("wallos import file is too large")

// currencySymbols maps currency symbols to candidate currency codes.
// Ambiguous symbols (e.g. ¥ for JPY/CNY, $ for USD/CAD/AUD) list multiple candidates;
// the user's preferred currency is used to disambiguate.
var currencySymbols = map[string][]string{
	"€": {"EUR"}, "£": {"GBP"}, "₩": {"KRW"}, "₹": {"INR"}, "₽": {"RUB"},
	"₺": {"TRY"}, "₴": {"UAH"}, "₫": {"VND"}, "₱": {"PHP"}, "₿": {"BTC"},
	"฿": {"THB"}, "₪": {"ILS"}, "zł": {"PLN"}, "Kč": {"CZK"},
	"¥": {"CNY", "JPY"}, "￥": {"CNY", "JPY"},
	"$":  {"USD", "CAD", "AUD", "NZD", "HKD", "SGD", "TWD", "MXN"},
	"kr": {"SEK", "NOK", "DKK", "ISK"},
	"Fr": {"CHF"},
	"RM": {"MYR"}, "Rp": {"IDR"}, "Rs": {"INR", "PKR", "LKR", "NPR"},
	"DH": {"MAD"}, "DA": {"DZD"}, "DT": {"TND"}, "LD": {"LYD"},
	"R$": {"BRL"}, "S$": {"SGD"}, "A$": {"AUD"}, "C$": {"CAD"},
	"NZ$": {"NZD"}, "HK$": {"HKD"}, "NT$": {"TWD"}, "MX$": {"MXN"},
}

var wallosPaymentMethods = map[string]bool{
	"PayPal": true, "Credit Card": true, "Bank Transfer": true, "Direct Debit": true,
	"Money": true, "Google Pay": true, "Samsung Pay": true, "Apple Pay": true,
	"Crypto": true, "Klarna": true, "Amazon Pay": true, "SEPA": true,
	"Skrill": true, "Sofort": true, "Stripe": true, "Affirm": true,
	"AliPay": true, "Elo": true, "Facebook Pay": true, "GiroPay": true,
	"iDeal": true, "Union Pay": true, "Interac": true, "WeChat": true,
	"Paysafe": true, "Poli": true, "Qiwi": true, "ShopPay": true,
	"Venmo": true, "VeriFone": true, "WebMoney": true,
}

// wallosToSystemKey maps Wallos payment method names to subdux system keys.
var wallosToSystemKey = map[string]string{
	"PayPal": "paypal",
	"AliPay": "alipay",
	"WeChat": "wechatpay",
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

func extractCurrencyAndAmount(price string, preferredCurrency string) (float64, string) {
	currency := preferredCurrency
	if currency == "" {
		currency = "USD"
	}
	found := false

	// Try symbol matching, longest symbols first to match "NZ$" before "$"
	type symEntry struct {
		sym   string
		codes []string
	}
	sorted := make([]symEntry, 0, len(currencySymbols))
	for sym, codes := range currencySymbols {
		sorted = append(sorted, symEntry{sym, codes})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].sym) > len(sorted[j].sym)
	})

	for _, entry := range sorted {
		if strings.Contains(price, entry.sym) {
			currency = resolveAmbiguous(entry.codes, preferredCurrency)
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

// resolveAmbiguous picks the best currency code from candidates.
// If the user's preferred currency is among the candidates, use it; otherwise use the first candidate.
func resolveAmbiguous(candidates []string, preferredCurrency string) string {
	if len(candidates) == 1 {
		return candidates[0]
	}
	for _, c := range candidates {
		if c == preferredCurrency {
			return c
		}
	}
	return candidates[0]
}

// symbolForCode returns the currency symbol for a given code, or empty string if unknown.
func symbolForCode(code string) string {
	for sym, codes := range currencySymbols {
		for _, c := range codes {
			if c == code {
				return sym
			}
		}
	}
	return ""
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

// errPreviewRollback is a sentinel error used to trigger transaction rollback in preview mode.
var errPreviewRollback = fmt.Errorf("preview rollback")

func (s *ImportService) ImportFromWallos(userID uint, data []WallosSubscription, confirm bool) (*WallosImportResponse, error) {
	if len(data) > maxWallosImportItems {
		return nil, ErrWallosImportTooLarge
	}

	result := &ImportResult{Errors: []string{}}
	preview := &ImportPreview{
		Currencies:     []PreviewCurrencyChange{},
		PaymentMethods: []PreviewPaymentMethodChange{},
		Categories:     []PreviewCategoryChange{},
		Subscriptions:  []PreviewSubscriptionChange{},
	}

	seenCurrencies := map[string]bool{}
	seenPaymentMethods := map[string]bool{}
	seenCategories := map[string]bool{}
	// Track subscriptions within this import batch to catch duplicates in the input data itself.
	// Key: "name|amount|currency|billingType"
	seenSubscriptions := map[string]bool{}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		// Read user's preferred currency to disambiguate symbols like ¥ and $
		var pref model.UserPreference
		preferredCurrency := "USD"
		if err := tx.Where("user_id = ?", userID).First(&pref).Error; err == nil {
			preferredCurrency = pref.PreferredCurrency
		}

		for _, item := range data {
			name := strings.TrimSpace(item.Name)
			if name == "" {
				if confirm {
					result.Errors = append(result.Errors, "skipped item with empty name")
					result.Skipped++
				}
				continue
			}

			amount, currency := extractCurrencyAndAmount(item.Price, preferredCurrency)
			billingType, recurrenceType, intervalUnit, intervalCount := mapPaymentCycle(item.PaymentCycle)
			nextBilling := parseDate(item.NextPayment)
			enabled := parseEnabled(item.Active)

			// Deduplicate by name + amount + currency + billing_type (without next_billing_date,
			// because the app may advance billing dates after import, causing re-imports).
			dedupKey := fmt.Sprintf("%s|%v|%s|%s", name, amount, currency, billingType)

			// Check within the current import batch first
			isDuplicate := seenSubscriptions[dedupKey]

			if !isDuplicate {
				// Check against existing DB records
				var count int64
				if err := tx.Model(&model.Subscription{}).
					Where("user_id = ? AND name = ? AND amount = ? AND currency = ? AND billing_type = ?",
						userID, name, amount, currency, billingType).
					Count(&count).Error; err != nil {
					return err
				}
				isDuplicate = count > 0
			}

			seenSubscriptions[dedupKey] = true

			// Collect preview info
			if currency != "" && !seenCurrencies[currency] {
				seenCurrencies[currency] = true
				var uc model.UserCurrency
				ucErr := tx.Where("user_id = ? AND code = ?", userID, currency).First(&uc).Error
				isNew := ucErr == gorm.ErrRecordNotFound
				preview.Currencies = append(preview.Currencies, PreviewCurrencyChange{
					Code:   currency,
					Symbol: symbolForCode(currency),
					IsNew:  isNew,
				})
			}

			pmName := strings.TrimSpace(item.PaymentMethod)
			if pmName != "" && wallosPaymentMethods[pmName] && !seenPaymentMethods[pmName] {
				seenPaymentMethods[pmName] = true
				var pm model.PaymentMethod
				found := false
				matched := ""

				if sysKey, ok := wallosToSystemKey[pmName]; ok {
					if err := tx.Where("user_id = ? AND system_key = ?", userID, sysKey).First(&pm).Error; err == nil {
						found = true
						matched = pm.Name
					}
				}
				if !found {
					if err := tx.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(pmName)).First(&pm).Error; err == nil {
						found = true
						matched = pm.Name
					}
				}

				preview.PaymentMethods = append(preview.PaymentMethods, PreviewPaymentMethodChange{
					Name:    pmName,
					IsNew:   !found,
					Matched: matched,
				})
			}

			categoryName := strings.TrimSpace(item.Category)
			if strings.EqualFold(categoryName, "No category") {
				categoryName = ""
			}
			if categoryName != "" && !seenCategories[categoryName] {
				seenCategories[categoryName] = true
				var cat model.Category
				catErr := tx.Where("user_id = ? AND name = ?", userID, categoryName).First(&cat).Error
				isNew := catErr == gorm.ErrRecordNotFound
				preview.Categories = append(preview.Categories, PreviewCategoryChange{
					Name:  categoryName,
					IsNew: isNew,
				})
			}

			sub := PreviewSubscriptionChange{
				Name:        name,
				Amount:      amount,
				Currency:    currency,
				BillingType: billingType,
				Category:    categoryName,
			}
			if isDuplicate {
				sub.Skipped = true
				sub.SkipReason = "duplicate"
			}
			preview.Subscriptions = append(preview.Subscriptions, sub)

			// Skip actual import logic in preview mode or for duplicates
			if !confirm || isDuplicate {
				if confirm {
					result.Skipped++
				}
				continue
			}

			// --- Actual import (confirm=true only) ---

			// Resolve category
			var categoryID *uint
			if categoryName != "" {
				var cat model.Category
				err := tx.Where("user_id = ? AND name = ?", userID, categoryName).First(&cat).Error
				if err == gorm.ErrRecordNotFound {
					cat = model.Category{UserID: userID, Name: categoryName}
					if err := tx.Create(&cat).Error; err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("failed to create category %q: %v", categoryName, err))
					} else {
						categoryID = &cat.ID
					}
				} else if err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to lookup category %q: %v", categoryName, err))
				} else {
					categoryID = &cat.ID
				}
			}

			// Ensure user currency exists
			if currency != "" {
				var uc model.UserCurrency
				err := tx.Where("user_id = ? AND code = ?", userID, currency).First(&uc).Error
				if err == gorm.ErrRecordNotFound {
					uc = model.UserCurrency{UserID: userID, Code: currency, Symbol: symbolForCode(currency)}
					if err := tx.Create(&uc).Error; err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("failed to create currency %q: %v", currency, err))
					}
				}
			}

			// Resolve payment method
			var paymentMethodID *uint
			if pmName != "" && wallosPaymentMethods[pmName] {
				var pm model.PaymentMethod
				found := false

				if sysKey, ok := wallosToSystemKey[pmName]; ok {
					if err := tx.Where("user_id = ? AND system_key = ?", userID, sysKey).First(&pm).Error; err == nil {
						found = true
					}
				}
				if !found {
					if err := tx.Where("user_id = ? AND LOWER(name) = ?", userID, strings.ToLower(pmName)).First(&pm).Error; err == nil {
						found = true
					}
				}
				if !found {
					pm = model.PaymentMethod{UserID: userID, Name: pmName}
					if err := tx.Create(&pm).Error; err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("failed to create payment method %q: %v", pmName, err))
					}
				}
				if pm.ID != 0 {
					paymentMethodID = &pm.ID
				}
			}

			notifyEnabled := parseEnabled(item.Notifications)
			subscription := model.Subscription{
				UserID:          userID,
				Name:            name,
				Amount:          amount,
				Currency:        currency,
				Enabled:         enabled,
				BillingType:     billingType,
				Category:        categoryName,
				CategoryID:      categoryID,
				PaymentMethodID: paymentMethodID,
				URL:             item.URL,
				Notes:           item.Notes,
				NotifyEnabled:   &notifyEnabled,
			}

			if billingType == "recurring" {
				subscription.RecurrenceType = recurrenceType
				subscription.IntervalUnit = intervalUnit
				subscription.IntervalCount = &intervalCount
			}

			if nextBilling != nil {
				subscription.NextBillingDate = nextBilling
			}

			if err := tx.Create(&subscription).Error; err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to import %q: %v", name, err))
				continue
			}

			// Force update enabled field to handle GORM default:true zero-value issue
			if !enabled {
				if err := tx.Model(&subscription).Update("enabled", false).Error; err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("failed to update enabled for %q: %v", name, err))
					continue
				}
			}

			result.Imported++
		}

		if !confirm {
			return errPreviewRollback
		}
		return nil
	})

	if err != nil && err != errPreviewRollback {
		return nil, err
	}

	if confirm {
		return &WallosImportResponse{Result: result}, nil
	}
	return &WallosImportResponse{Preview: preview}, nil
}
