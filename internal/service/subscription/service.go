package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

const (
	billingTypeRecurring = "recurring"

	subscriptionStatusActive = "active"
	subscriptionStatusEnded  = "ended"

	renewalModeAutoRenew         = "auto_renew"
	renewalModeManualRenew       = "manual_renew"
	renewalModeCancelAtPeriodEnd = "cancel_at_period_end"

	recurrenceTypeInterval    = "interval"
	recurrenceTypeMonthlyDate = "monthly_date"
	recurrenceTypeYearlyDate  = "yearly_date"

	intervalUnitDay   = "day"
	intervalUnitWeek  = "week"
	intervalUnitMonth = "month"
	intervalUnitYear  = "year"

	MaxNotificationDaysBefore = 10

	BillingTypeRecurring = billingTypeRecurring

	StatusActive = subscriptionStatusActive
	StatusEnded  = subscriptionStatusEnded

	RenewalModeAutoRenew         = renewalModeAutoRenew
	RenewalModeManualRenew       = renewalModeManualRenew
	RenewalModeCancelAtPeriodEnd = renewalModeCancelAtPeriodEnd

	RecurrenceTypeInterval    = recurrenceTypeInterval
	RecurrenceTypeMonthlyDate = recurrenceTypeMonthlyDate
	RecurrenceTypeYearlyDate  = recurrenceTypeYearlyDate

	IntervalUnitDay   = intervalUnitDay
	IntervalUnitWeek  = intervalUnitWeek
	IntervalUnitMonth = intervalUnitMonth
	IntervalUnitYear  = intervalUnitYear

	TriggerDaysBefore  = "days_before"
	TriggerDueDay      = "due_day"
	TriggerManualDaily = "manual_renew_daily"
	TriggerManualEnded = "manual_renew_ended"
	TriggerEndingSoon  = "ending_soon"
)

type CurrencyConverter interface {
	Convert(amount float64, from, to string) float64
}

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

type CreateSubscriptionInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	Currency         string  `json:"currency"`
	Status           string  `json:"status"`
	RenewalMode      string  `json:"renewal_mode"`
	EndsAt           string  `json:"ends_at"`
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
	Status           *string  `json:"status"`
	RenewalMode      *string  `json:"renewal_mode"`
	EndsAt           *string  `json:"ends_at"`
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

	CategoryIDSet       bool `json:"-"`
	PaymentMethodIDSet  bool `json:"-"`
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
	if _, ok := raw["category_id"]; ok {
		input.CategoryIDSet = true
	}
	if _, ok := raw["payment_method_id"]; ok {
		input.PaymentMethodIDSet = true
	}

	return nil
}

type DashboardSummary struct {
	TotalMonthly         float64 `json:"total_monthly"`
	TotalYearly          float64 `json:"total_yearly"`
	CommittedMonthly     float64 `json:"committed_monthly"`
	CommittedYearly      float64 `json:"committed_yearly"`
	DueThisMonth         float64 `json:"due_this_month"`
	ActiveCount          int64   `json:"active_count"`
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

type BillingDraft = billingDraft

func validateNotifyDaysBefore(value int) error {
	if value < 0 || value > MaxNotificationDaysBefore {
		return serviceerr.New(serviceerr.KindInvalid, fmt.Sprintf("notify_days_before must be between 0 and %d", MaxNotificationDaysBefore))
	}
	return nil
}
