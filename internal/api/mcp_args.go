package api

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shiroha/subdux/internal/service"
)

func createSubscriptionInputFromMCPArgs(args map[string]interface{}) service.CreateSubscriptionInput {
	intervalCount := 1
	if value, ok := readIntArg(args, "interval_count"); ok {
		intervalCount = value
	}

	input := service.CreateSubscriptionInput{
		Name:            readStringArgOrDefault(args, "name", ""),
		Amount:          readFloatArgOrDefault(args, "amount", 0),
		Currency:        readStringArgOrDefault(args, "currency", "USD"),
		Status:          readStringArgOrDefault(args, "status", "active"),
		RenewalMode:     readStringArgOrDefault(args, "renewal_mode", "auto_renew"),
		EndsAt:          readStringArgOrDefault(args, "ends_at", ""),
		BillingType:     readStringArgOrDefault(args, "billing_type", "recurring"),
		RecurrenceType:  readStringArgOrDefault(args, "recurrence_type", "interval"),
		IntervalCount:   &intervalCount,
		IntervalUnit:    readStringArgOrDefault(args, "interval_unit", "month"),
		NextBillingDate: readStringArgOrDefault(args, "next_billing_date", ""),
		Category:        readStringArgOrDefault(args, "category", ""),
		Icon:            readStringArgOrDefault(args, "icon", ""),
		URL:             readStringArgOrDefault(args, "url", ""),
		Notes:           readStringArgOrDefault(args, "notes", ""),
	}

	if value, ok := readUintPointerArg(args, "category_id"); ok {
		input.CategoryID = value
	}
	if value, ok := readUintPointerArg(args, "payment_method_id"); ok {
		input.PaymentMethodID = value
	}
	if value, ok := readBoolPointerArg(args, "notify_enabled"); ok {
		input.NotifyEnabled = value
	}
	if value, ok := readIntPointerArg(args, "notify_days_before"); ok {
		input.NotifyDaysBefore = value
	}

	switch input.RecurrenceType {
	case "monthly_date":
		input.IntervalCount = nil
		input.IntervalUnit = ""
		if value, ok := readIntPointerArg(args, "monthly_day"); ok {
			input.MonthlyDay = value
		}
	case "yearly_date":
		input.IntervalCount = nil
		input.IntervalUnit = ""
		if value, ok := readIntPointerArg(args, "yearly_month"); ok {
			input.YearlyMonth = value
		}
		if value, ok := readIntPointerArg(args, "yearly_day"); ok {
			input.YearlyDay = value
		}
	default:
		input.MonthlyDay = nil
		input.YearlyMonth = nil
		input.YearlyDay = nil
	}

	return input
}

func updateSubscriptionInputFromMCPArgs(args map[string]interface{}) (service.UpdateSubscriptionInput, error) {
	var input service.UpdateSubscriptionInput
	if err := validateSubscriptionWriteArgTypes(args); err != nil {
		return input, err
	}

	if value, ok := readStringArg(args, "name"); ok {
		trimmed := strings.TrimSpace(value)
		input.Name = &trimmed
	}
	if value, ok := readFloatArg(args, "amount"); ok {
		input.Amount = &value
	}
	if value, ok := readStringArg(args, "currency"); ok {
		input.Currency = &value
	}
	if value, ok := readStringArg(args, "status"); ok {
		input.Status = &value
	}
	if value, ok := readStringArg(args, "renewal_mode"); ok {
		input.RenewalMode = &value
	}
	if value, ok := readNullableStringArg(args, "ends_at"); ok {
		input.EndsAt = &value
	}
	if value, ok := readStringArg(args, "billing_type"); ok {
		input.BillingType = &value
	}
	if value, ok := readStringArg(args, "recurrence_type"); ok {
		input.RecurrenceType = &value
	}
	if value, ok := readNullableIntArg(args, "interval_count"); ok {
		input.IntervalCount = value
	}
	if value, ok := readStringArg(args, "interval_unit"); ok {
		input.IntervalUnit = &value
	}
	if value, ok := readNullableStringArg(args, "next_billing_date"); ok {
		input.NextBillingDate = &value
	}
	if value, ok := readNullableIntArg(args, "monthly_day"); ok {
		input.MonthlyDay = value
	}
	if value, ok := readNullableIntArg(args, "yearly_month"); ok {
		input.YearlyMonth = value
	}
	if value, ok := readNullableIntArg(args, "yearly_day"); ok {
		input.YearlyDay = value
	}
	if value, ok := readStringArg(args, "category"); ok {
		input.Category = &value
	}
	if value, ok := readNullableUintArg(args, "category_id"); ok {
		input.CategoryIDSet = true
		input.CategoryID = value
	}
	if value, ok := readNullableUintArg(args, "payment_method_id"); ok {
		input.PaymentMethodIDSet = true
		input.PaymentMethodID = value
	}
	if value, ok := readNullableBoolArg(args, "notify_enabled"); ok {
		input.NotifyEnabledSet = true
		input.NotifyEnabled = value
	}
	if value, ok := readNullableIntArg(args, "notify_days_before"); ok {
		input.NotifyDaysBeforeSet = true
		input.NotifyDaysBefore = value
	}
	if value, ok := readStringArg(args, "icon"); ok {
		input.Icon = &value
	}
	if value, ok := readStringArg(args, "url"); ok {
		input.URL = &value
	}
	if value, ok := readStringArg(args, "notes"); ok {
		input.Notes = &value
	}
	return input, nil
}

func validateSubscriptionWriteArgTypes(args map[string]interface{}) error {
	if err := validateMCPArgTypes(args, []mcpArgSpec{
		{Key: "id", Type: "integer"},
		{Key: "name", Type: "string"},
		{Key: "amount", Type: "number"},
		{Key: "currency", Type: "string"},
		{Key: "status", Type: "string"},
		{Key: "renewal_mode", Type: "string"},
		{Key: "ends_at", Type: "string", Nullable: true},
		{Key: "billing_type", Type: "string"},
		{Key: "recurrence_type", Type: "string"},
		{Key: "interval_count", Type: "integer", Nullable: true},
		{Key: "interval_unit", Type: "string"},
		{Key: "next_billing_date", Type: "string", Nullable: true},
		{Key: "monthly_day", Type: "integer", Nullable: true},
		{Key: "yearly_month", Type: "integer", Nullable: true},
		{Key: "yearly_day", Type: "integer", Nullable: true},
		{Key: "category", Type: "string"},
		{Key: "category_id", Type: "integer", Nullable: true},
		{Key: "payment_method_id", Type: "integer", Nullable: true},
		{Key: "notify_enabled", Type: "boolean", Nullable: true},
		{Key: "notify_days_before", Type: "integer", Nullable: true},
		{Key: "icon", Type: "string"},
		{Key: "url", Type: "string"},
		{Key: "notes", Type: "string"},
	}); err != nil {
		return err
	}

	if value, ok := readNullableIntArg(args, "notify_days_before"); ok && value != nil {
		if *value < 0 || *value > 10 {
			return errors.New("notify_days_before must be between 0 and 10")
		}
	}
	return nil
}

type mcpArgSpec struct {
	Key      string
	Type     string
	Nullable bool
}

func validateMCPArgTypes(args map[string]interface{}, specs []mcpArgSpec) error {
	for _, spec := range specs {
		value, exists := args[spec.Key]
		if !exists {
			continue
		}
		if value == nil {
			if spec.Nullable {
				continue
			}
			return fmt.Errorf("%s must be %s", spec.Key, spec.Type)
		}

		var ok bool
		switch spec.Type {
		case "string":
			_, ok = value.(string)
		case "number":
			_, ok = readFloatArg(args, spec.Key)
		case "integer":
			_, ok = readIntArg(args, spec.Key)
		case "boolean":
			_, ok = readBoolArg(args, spec.Key)
		default:
			ok = true
		}
		if !ok {
			return fmt.Errorf("%s must be %s", spec.Key, spec.Type)
		}
	}
	return nil
}

func readRequiredIDArg(args map[string]interface{}, key string) (uint, error) {
	value, ok := readIntArg(args, key)
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	if value < 1 {
		return 0, nil
	}
	return uint(value), nil
}

func readStringArgOrDefault(args map[string]interface{}, key, fallback string) string {
	if value, ok := readStringArg(args, key); ok {
		return value
	}
	return fallback
}

func readFloatArgOrDefault(args map[string]interface{}, key string, fallback float64) float64 {
	if value, ok := readFloatArg(args, key); ok {
		return value
	}
	return fallback
}

func readStringArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return typed, true
	default:
		return fmt.Sprint(typed), true
	}
}

func readNullableStringArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}
	if value == nil {
		return "", true
	}
	return readStringArg(args, key)
}

func readFloatArg(args map[string]interface{}, key string) (float64, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func readIntArg(args map[string]interface{}, key string) (int, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		asInt := int(typed)
		return asInt, typed == float64(asInt)
	case int:
		return typed, true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func readNullableIntArg(args map[string]interface{}, key string) (*int, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readIntArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readIntPointerArg(args map[string]interface{}, key string) (*int, bool) {
	parsed, ok := readIntArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readUintArg(args map[string]interface{}, key string) (uint, bool) {
	parsed, ok := readIntArg(args, key)
	if !ok || parsed < 0 {
		return 0, false
	}
	return uint(parsed), true
}

func readNullableUintArg(args map[string]interface{}, key string) (*uint, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readUintArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readUintPointerArg(args map[string]interface{}, key string) (*uint, bool) {
	parsed, ok := readUintArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readBoolPointerArg(args map[string]interface{}, key string) (*bool, bool) {
	parsed, ok := readBoolArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readNullableBoolArg(args map[string]interface{}, key string) (*bool, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readBoolArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readBoolArg(args map[string]interface{}, key string) (bool, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return false, false
	}
}
