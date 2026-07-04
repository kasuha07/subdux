package mcp

import (
	"net/url"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/kasuha07/subdux/internal/model"
)

type subscriptionResponse struct {
	ID               uint      `json:"id"`
	Name             string    `json:"name"`
	Amount           float64   `json:"amount"`
	Currency         string    `json:"currency"`
	Status           string    `json:"status"`
	RenewalMode      string    `json:"renewal_mode"`
	EndsAt           *string   `json:"ends_at"`
	BillingType      string    `json:"billing_type"`
	RecurrenceType   string    `json:"recurrence_type"`
	IntervalCount    *int      `json:"interval_count"`
	IntervalUnit     string    `json:"interval_unit"`
	MonthlyDay       *int      `json:"monthly_day"`
	YearlyMonth      *int      `json:"yearly_month"`
	YearlyDay        *int      `json:"yearly_day"`
	NextBillingDate  *string   `json:"next_billing_date"`
	Category         string    `json:"category"`
	CategoryID       *uint     `json:"category_id"`
	PaymentMethodID  *uint     `json:"payment_method_id"`
	NotifyEnabled    *bool     `json:"notify_enabled"`
	NotifyDaysBefore *int      `json:"notify_days_before"`
	Icon             string    `json:"icon"`
	URL              string    `json:"url"`
	Notes            string    `json:"notes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type categoryResponse struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	SystemKey      *string `json:"system_key"`
	NameCustomized bool    `json:"name_customized"`
	DisplayOrder   int     `json:"display_order"`
}

type paymentMethodResponse struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	SystemKey      *string `json:"system_key"`
	NameCustomized bool    `json:"name_customized"`
	Icon           string  `json:"icon"`
	SortOrder      int     `json:"sort_order"`
}

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:               sub.ID,
		Name:             sub.Name,
		Amount:           sub.Amount,
		Currency:         sub.Currency,
		Status:           sub.Status,
		RenewalMode:      sub.RenewalMode,
		EndsAt:           formatDateOnly(sub.EndsAt),
		BillingType:      sub.BillingType,
		RecurrenceType:   sub.RecurrenceType,
		IntervalCount:    sub.IntervalCount,
		IntervalUnit:     sub.IntervalUnit,
		MonthlyDay:       sub.MonthlyDay,
		YearlyMonth:      sub.YearlyMonth,
		YearlyDay:        sub.YearlyDay,
		NextBillingDate:  formatDateOnly(sub.NextBillingDate),
		Category:         sub.Category,
		CategoryID:       sub.CategoryID,
		PaymentMethodID:  sub.PaymentMethodID,
		NotifyEnabled:    sub.NotifyEnabled,
		NotifyDaysBefore: sub.NotifyDaysBefore,
		Icon:             sub.Icon,
		URL:              sub.URL,
		Notes:            sub.Notes,
		CreatedAt:        sub.CreatedAt,
		UpdatedAt:        sub.UpdatedAt,
	}
}

func mapSubscriptionResponses(subs []model.Subscription) []subscriptionResponse {
	responses := make([]subscriptionResponse, len(subs))
	for i, sub := range subs {
		responses[i] = mapSubscriptionResponse(sub)
	}
	return responses
}

func mapCategoryResponse(category model.Category) categoryResponse {
	return categoryResponse{
		ID:             category.ID,
		Name:           category.Name,
		SystemKey:      category.SystemKey,
		NameCustomized: category.NameCustomized,
		DisplayOrder:   category.DisplayOrder,
	}
}

func mapCategoryResponses(categories []model.Category) []categoryResponse {
	responses := make([]categoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = mapCategoryResponse(category)
	}
	return responses
}

func mapPaymentMethodResponse(method model.PaymentMethod) paymentMethodResponse {
	return paymentMethodResponse{
		ID:             method.ID,
		Name:           method.Name,
		SystemKey:      method.SystemKey,
		NameCustomized: method.NameCustomized,
		Icon:           method.Icon,
		SortOrder:      method.SortOrder,
	}
}

func mapPaymentMethodResponses(methods []model.PaymentMethod) []paymentMethodResponse {
	responses := make([]paymentMethodResponse, len(methods))
	for i, method := range methods {
		responses[i] = mapPaymentMethodResponse(method)
	}
	return responses
}

func formatDateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format("2006-01-02")
	return &formatted
}

func isSubscriptionBadRequestError(message string) bool {
	if message == "payment method not found" || message == "category not found" {
		return true
	}
	return strings.Contains(message, "required") ||
		strings.Contains(message, "must be") ||
		strings.Contains(message, "invalid date format") ||
		strings.Contains(message, "invalid subscription url") ||
		strings.Contains(message, "no longer supported") ||
		strings.Contains(message, "read-only") ||
		strings.Contains(message, "only ")
}

func validateSubscriptionIcon(icon string) bool {
	if validateIcon(icon) {
		return true
	}
	return isExternalImageIconURL(icon) || isManagedProxyImageIconURL(icon)
}

func validateIcon(icon string) bool {
	if icon == "" {
		return true
	}

	if isManagedAssetIcon(icon) {
		return true
	}

	if isIconGoIcon(icon) {
		return true
	}

	for _, r := range icon {
		if !isEmojiRune(r) {
			return false
		}
	}
	return true
}

func isEmojiRune(r rune) bool {
	if r == '\u200D' || r == '\uFE0F' || r == '\uFE0E' {
		return true
	}
	if r >= 0x1F1E0 && r <= 0x1F1FF {
		return true
	}
	if r < 0x00A0 {
		return false
	}
	return unicode.IsGraphic(r) && !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsPunct(r) && !unicode.IsSpace(r)
}

func isManagedAssetIcon(icon string) bool {
	const iconPrefix = "file:"
	if !strings.HasPrefix(icon, iconPrefix) {
		return false
	}

	filename := strings.TrimPrefix(icon, iconPrefix)
	if filename == "" {
		return false
	}
	if strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return false
	}
	if filepath.Base(filename) != filename {
		return false
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".ico" {
		return false
	}

	return true
}

func isExternalImageIconURL(icon string) bool {
	parsed, err := url.ParseRequestURI(icon)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func isManagedProxyImageIconURL(icon string) bool {
	parsed, err := url.ParseRequestURI(icon)
	if err != nil {
		return false
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return false
	}

	switch parsed.Path {
	case "/api/icon-proxy/google", "/api/icon-proxy/icon-horse":
	default:
		return false
	}

	domain := strings.ToLower(strings.TrimSpace(parsed.Query().Get("domain")))
	if domain == "" || strings.Contains(domain, "://") || strings.Contains(domain, "/") {
		return false
	}

	normalized := strings.TrimRight(domain, ".")
	if normalized == "" {
		return false
	}
	return isValidDomainLike(normalized)
}

func isValidDomainLike(domain string) bool {
	if strings.Contains(domain, "..") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			ch := label[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return false
		}
	}

	tld := labels[len(labels)-1]
	return len(tld) >= 2
}

func isIconGoIcon(icon string) bool {
	prefix, slug, found := strings.Cut(icon, ":")
	if !found || prefix == "" || slug == "" {
		return false
	}

	if prefix == "si" || prefix == "file" || len(prefix) < 2 || len(prefix) > 16 {
		return false
	}

	for _, r := range prefix {
		if r < 'a' || r > 'z' {
			return false
		}
	}

	for _, r := range slug {
		isLowerAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLowerAlpha || isDigit || r == '-' || r == '_' {
			continue
		}
		return false
	}

	return true
}
