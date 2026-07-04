package mcp

import (
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/api/apicontract"
	"github.com/kasuha07/subdux/internal/model"
)

type subscriptionResponse = apicontract.SubscriptionResponse
type categoryResponse = apicontract.CategoryResponse
type paymentMethodResponse = apicontract.PaymentMethodResponse

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return apicontract.MapSubscriptionResponse(sub)
}

func mapSubscriptionResponses(subs []model.Subscription) []subscriptionResponse {
	return apicontract.MapSubscriptionResponses(subs)
}

func mapCategoryResponse(category model.Category) categoryResponse {
	return apicontract.MapCategoryResponse(category)
}

func mapCategoryResponses(categories []model.Category) []categoryResponse {
	return apicontract.MapCategoryResponses(categories)
}

func mapPaymentMethodResponse(method model.PaymentMethod) paymentMethodResponse {
	return apicontract.MapPaymentMethodResponse(method)
}

func mapPaymentMethodResponses(methods []model.PaymentMethod) []paymentMethodResponse {
	return apicontract.MapPaymentMethodResponses(methods)
}

func formatDateOnly(value *time.Time) *string {
	return apicontract.FormatDateOnly(value)
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
	return apicontract.ValidateSubscriptionIcon(icon)
}

func validateIcon(icon string) bool {
	return apicontract.ValidateIcon(icon)
}
