package api

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

type mcpSubscriptionSearchFilters struct {
	Query              string
	Status             string
	Currency           string
	RenewalMode        string
	BillingType        string
	RecurrenceType     string
	Category           string
	CategoryID         *uint
	CategoryIDSet      bool
	PaymentMethodID    *uint
	PaymentMethodIDSet bool
	NextBillingFrom    *time.Time
	NextBillingTo      *time.Time
	Limit              int
}

func readMCPSubscriptionSearchFilters(args map[string]interface{}) (mcpSubscriptionSearchFilters, error) {
	filters := mcpSubscriptionSearchFilters{Limit: 20}
	if err := validateMCPArgTypes(args, []mcpArgSpec{
		{Key: "query", Type: "string"},
		{Key: "status", Type: "string"},
		{Key: "currency", Type: "string"},
		{Key: "renewal_mode", Type: "string"},
		{Key: "billing_type", Type: "string"},
		{Key: "recurrence_type", Type: "string"},
		{Key: "category", Type: "string"},
		{Key: "category_id", Type: "integer", Nullable: true},
		{Key: "payment_method_id", Type: "integer", Nullable: true},
		{Key: "next_billing_from", Type: "string"},
		{Key: "next_billing_to", Type: "string"},
		{Key: "limit", Type: "integer"},
	}); err != nil {
		return filters, err
	}

	if value, ok := readStringArg(args, "query"); ok {
		filters.Query = strings.ToLower(strings.TrimSpace(value))
	}
	if value, ok := readStringArg(args, "status"); ok {
		filters.Status = strings.TrimSpace(value)
		if filters.Status != "" && filters.Status != "active" && filters.Status != "ended" {
			return filters, errors.New("status must be active or ended")
		}
	}
	if value, ok := readStringArg(args, "currency"); ok {
		filters.Currency = strings.ToUpper(strings.TrimSpace(value))
	}
	if value, ok := readStringArg(args, "renewal_mode"); ok {
		filters.RenewalMode = strings.TrimSpace(value)
		switch filters.RenewalMode {
		case "", "auto_renew", "manual_renew", "cancel_at_period_end":
		default:
			return filters, errors.New("renewal_mode must be auto_renew, manual_renew, or cancel_at_period_end")
		}
	}
	if value, ok := readStringArg(args, "billing_type"); ok {
		filters.BillingType = strings.TrimSpace(value)
		if filters.BillingType != "" && filters.BillingType != "recurring" {
			return filters, errors.New("billing_type must be recurring")
		}
	}
	if value, ok := readStringArg(args, "recurrence_type"); ok {
		filters.RecurrenceType = strings.TrimSpace(value)
		switch filters.RecurrenceType {
		case "", "interval", "monthly_date", "yearly_date":
		default:
			return filters, errors.New("recurrence_type must be interval, monthly_date, or yearly_date")
		}
	}
	if value, ok := readStringArg(args, "category"); ok {
		filters.Category = strings.ToLower(strings.TrimSpace(value))
	}
	if value, ok := readNullableUintArg(args, "category_id"); ok {
		filters.CategoryID = value
		filters.CategoryIDSet = true
	}
	if value, ok := readNullableUintArg(args, "payment_method_id"); ok {
		filters.PaymentMethodID = value
		filters.PaymentMethodIDSet = true
	}
	if value, ok := readStringArg(args, "next_billing_from"); ok {
		parsed, err := parseMCPDateArg("next_billing_from", value)
		if err != nil {
			return filters, err
		}
		filters.NextBillingFrom = parsed
	}
	if value, ok := readStringArg(args, "next_billing_to"); ok {
		parsed, err := parseMCPDateArg("next_billing_to", value)
		if err != nil {
			return filters, err
		}
		filters.NextBillingTo = parsed
	}
	if filters.NextBillingFrom != nil && filters.NextBillingTo != nil && filters.NextBillingFrom.After(*filters.NextBillingTo) {
		return filters, errors.New("next_billing_from must be on or before next_billing_to")
	}
	if value, ok := readIntArg(args, "limit"); ok {
		if value < 1 || value > 100 {
			return filters, errors.New("limit must be between 1 and 100")
		}
		filters.Limit = value
	}

	return filters, nil
}

func parseMCPDateArg(key, value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, fmt.Errorf("%s must be in YYYY-MM-DD format", key)
	}
	return &parsed, nil
}

func matchesMCPSubscriptionSearch(sub model.Subscription, filters mcpSubscriptionSearchFilters, categoryLabels map[uint]string) bool {
	if filters.Query != "" && !strings.Contains(mcpSubscriptionSearchText(sub, categoryLabels), filters.Query) {
		return false
	}
	if filters.Status != "" && sub.Status != filters.Status {
		return false
	}
	if filters.Currency != "" && !strings.EqualFold(sub.Currency, filters.Currency) {
		return false
	}
	if filters.RenewalMode != "" && sub.RenewalMode != filters.RenewalMode {
		return false
	}
	if filters.BillingType != "" && sub.BillingType != filters.BillingType {
		return false
	}
	if filters.RecurrenceType != "" && sub.RecurrenceType != filters.RecurrenceType {
		return false
	}
	if filters.Category != "" && !strings.Contains(mcpSubscriptionCategorySearchText(sub, categoryLabels), filters.Category) {
		return false
	}
	if filters.CategoryIDSet && !uintPointersEqual(sub.CategoryID, filters.CategoryID) {
		return false
	}
	if filters.PaymentMethodIDSet && !uintPointersEqual(sub.PaymentMethodID, filters.PaymentMethodID) {
		return false
	}
	if filters.NextBillingFrom != nil {
		if sub.NextBillingDate == nil || dateOnlyBefore(*sub.NextBillingDate, *filters.NextBillingFrom) {
			return false
		}
	}
	if filters.NextBillingTo != nil {
		if sub.NextBillingDate == nil || dateOnlyAfter(*sub.NextBillingDate, *filters.NextBillingTo) {
			return false
		}
	}
	return true
}

func mcpSubscriptionSearchText(sub model.Subscription, categoryLabels map[uint]string) string {
	return strings.ToLower(strings.Join([]string{
		sub.Name,
		sub.Category,
		mcpSubscriptionCategoryName(sub, categoryLabels),
		sub.Currency,
		sub.Status,
		sub.RenewalMode,
		sub.BillingType,
		sub.RecurrenceType,
		sub.URL,
		sub.Notes,
	}, " "))
}

func mcpSubscriptionCategorySearchText(sub model.Subscription, categoryLabels map[uint]string) string {
	return strings.ToLower(strings.Join([]string{
		sub.Category,
		mcpSubscriptionCategoryName(sub, categoryLabels),
	}, " "))
}

func mcpSubscriptionCategoryName(sub model.Subscription, categoryLabels map[uint]string) string {
	if sub.CategoryID == nil {
		return ""
	}
	return strings.TrimSpace(categoryLabels[*sub.CategoryID])
}

func uintPointersEqual(left, right *uint) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func dateOnlyBefore(left, right time.Time) bool {
	leftDate := time.Date(left.Year(), left.Month(), left.Day(), 0, 0, 0, 0, time.UTC)
	rightDate := time.Date(right.Year(), right.Month(), right.Day(), 0, 0, 0, 0, time.UTC)
	return leftDate.Before(rightDate)
}

func dateOnlyAfter(left, right time.Time) bool {
	leftDate := time.Date(left.Year(), left.Month(), left.Day(), 0, 0, 0, 0, time.UTC)
	rightDate := time.Date(right.Year(), right.Month(), right.Day(), 0, 0, 0, 0, time.UTC)
	return leftDate.After(rightDate)
}
