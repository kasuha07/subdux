package apicontract

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

type SubscriptionResponse struct {
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

type CategoryResponse struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	SystemKey      *string `json:"system_key"`
	NameCustomized bool    `json:"name_customized"`
	DisplayOrder   int     `json:"display_order"`
}

type PaymentMethodResponse struct {
	ID             uint    `json:"id"`
	Name           string  `json:"name"`
	SystemKey      *string `json:"system_key"`
	NameCustomized bool    `json:"name_customized"`
	Icon           string  `json:"icon"`
	SortOrder      int     `json:"sort_order"`
}

func MapSubscriptionResponse(sub model.Subscription) SubscriptionResponse {
	return SubscriptionResponse{
		ID:               sub.ID,
		Name:             sub.Name,
		Amount:           sub.Amount,
		Currency:         sub.Currency,
		Status:           sub.Status,
		RenewalMode:      sub.RenewalMode,
		EndsAt:           FormatDateOnly(sub.EndsAt),
		BillingType:      sub.BillingType,
		RecurrenceType:   sub.RecurrenceType,
		IntervalCount:    sub.IntervalCount,
		IntervalUnit:     sub.IntervalUnit,
		MonthlyDay:       sub.MonthlyDay,
		YearlyMonth:      sub.YearlyMonth,
		YearlyDay:        sub.YearlyDay,
		NextBillingDate:  FormatDateOnly(sub.NextBillingDate),
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

func MapSubscriptionResponses(subs []model.Subscription) []SubscriptionResponse {
	responses := make([]SubscriptionResponse, len(subs))
	for i, sub := range subs {
		responses[i] = MapSubscriptionResponse(sub)
	}
	return responses
}

func MapCategoryResponse(category model.Category) CategoryResponse {
	return CategoryResponse{
		ID:             category.ID,
		Name:           category.Name,
		SystemKey:      category.SystemKey,
		NameCustomized: category.NameCustomized,
		DisplayOrder:   category.DisplayOrder,
	}
}

func MapCategoryResponses(categories []model.Category) []CategoryResponse {
	responses := make([]CategoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = MapCategoryResponse(category)
	}
	return responses
}

func MapPaymentMethodResponse(method model.PaymentMethod) PaymentMethodResponse {
	return PaymentMethodResponse{
		ID:             method.ID,
		Name:           method.Name,
		SystemKey:      method.SystemKey,
		NameCustomized: method.NameCustomized,
		Icon:           method.Icon,
		SortOrder:      method.SortOrder,
	}
}

func MapPaymentMethodResponses(methods []model.PaymentMethod) []PaymentMethodResponse {
	responses := make([]PaymentMethodResponse, len(methods))
	for i, method := range methods {
		responses[i] = MapPaymentMethodResponse(method)
	}
	return responses
}

func FormatDateOnly(value *time.Time) *string {
	if value == nil {
		return nil
	}

	formatted := value.Format("2006-01-02")
	return &formatted
}
