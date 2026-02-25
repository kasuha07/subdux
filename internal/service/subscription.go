package service

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	billingTypeRecurring = "recurring"
	billingTypeOneTime   = "one_time"
	billingTypeLifetime  = "lifetime"

	recurrenceTypeInterval    = "interval"
	recurrenceTypeMonthlyDate = "monthly_date"
	recurrenceTypeYearlyDate  = "yearly_date"

	intervalUnitDay   = "day"
	intervalUnitWeek  = "week"
	intervalUnitMonth = "month"
	intervalUnitYear  = "year"
)

type CurrencyConverter interface {
	Convert(amount float64, from, to string) float64
}

type SubscriptionService struct {
	DB *gorm.DB
}

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return &SubscriptionService{DB: db}
}

type CreateSubscriptionInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Enabled          *bool   `json:"enabled"`
	BillingType      string  `json:"billing_type"`
	RecurrenceType   string  `json:"recurrence_type"`
	IntervalCount    *int    `json:"interval_count"`
	IntervalUnit     string  `json:"interval_unit"`
	NextBillingDate  string  `json:"next_billing_date"`
	MonthlyDay       *int    `json:"monthly_day"`
	YearlyMonth      *int    `json:"yearly_month"`
	YearlyDay        *int    `json:"yearly_day"`
	Category         string  `json:"category"`
	CategoryID       *uint   `json:"category_id"`
	PaymentMethodID  *uint   `json:"payment_method_id"`
	NotifyEnabled    *bool   `json:"notify_enabled"`
	NotifyDaysBefore *int    `json:"notify_days_before"`
	Icon             string  `json:"icon"`
	URL              string  `json:"url"`
	Notes            string  `json:"notes"`
}

type UpdateSubscriptionInput struct {
	Name             *string  `json:"name"`
	Amount           *float64 `json:"amount"`
	Currency         *string  `json:"currency"`
	Enabled          *bool    `json:"enabled"`
	BillingType      *string  `json:"billing_type"`
	RecurrenceType   *string  `json:"recurrence_type"`
	IntervalCount    *int     `json:"interval_count"`
	IntervalUnit     *string  `json:"interval_unit"`
	NextBillingDate  *string  `json:"next_billing_date"`
	MonthlyDay       *int     `json:"monthly_day"`
	YearlyMonth      *int     `json:"yearly_month"`
	YearlyDay        *int     `json:"yearly_day"`
	Category         *string  `json:"category"`
	CategoryID       *uint    `json:"category_id"`
	PaymentMethodID  *uint    `json:"payment_method_id"`
	NotifyEnabled    *bool    `json:"notify_enabled"`
	NotifyDaysBefore *int     `json:"notify_days_before"`
	Icon             *string  `json:"icon"`
	URL              *string  `json:"url"`
	Notes            *string  `json:"notes"`

	NotifyEnabledSet    bool `json:"-"`
	NotifyDaysBeforeSet bool `json:"-"`
}

func (input *UpdateSubscriptionInput) UnmarshalJSON(data []byte) error {
	type alias UpdateSubscriptionInput
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*input = UpdateSubscriptionInput(decoded)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if _, ok := raw["notify_enabled"]; ok {
		input.NotifyEnabledSet = true
	}
	if _, ok := raw["notify_days_before"]; ok {
		input.NotifyDaysBeforeSet = true
	}

	return nil
}

type DashboardSummary struct {
	TotalMonthly         float64 `json:"total_monthly"`
	TotalYearly          float64 `json:"total_yearly"`
	DueThisMonth         float64 `json:"due_this_month"`
	EnabledCount         int64   `json:"enabled_count"`
	UpcomingRenewalCount int64   `json:"upcoming_renewal_count"`
	Currency             string  `json:"currency"`
}

type billingDraft struct {
	BillingType     string
	RecurrenceType  string
	IntervalCount   *int
	IntervalUnit    string
	NextBillingDate *time.Time
	MonthlyDay      *int
	YearlyMonth     *int
	YearlyDay       *int
}

func validateNotifyDaysBefore(value int) error {
	if value < 0 || value > maxNotificationDaysBefore {
		return fmt.Errorf("notify_days_before must be between 0 and %d", maxNotificationDaysBefore)
	}
	return nil
}
