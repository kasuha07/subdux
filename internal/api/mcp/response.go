package mcp

import (
	"errors"
	"time"

	"github.com/kasuha07/subdux/internal/api/contract"
	"github.com/kasuha07/subdux/internal/model"
)

type subscriptionResponse = contract.SubscriptionResponse
type categoryResponse = contract.CategoryResponse
type paymentMethodResponse = contract.PaymentMethodResponse

func mapSubscriptionResponse(sub model.Subscription) subscriptionResponse {
	return contract.MapSubscriptionResponse(sub)
}

func mapSubscriptionResponses(subs []model.Subscription) []subscriptionResponse {
	return contract.MapSubscriptionResponses(subs)
}

func mapCategoryResponse(category model.Category) categoryResponse {
	return contract.MapCategoryResponse(category)
}

func mapCategoryResponses(categories []model.Category) []categoryResponse {
	return contract.MapCategoryResponses(categories)
}

func mapPaymentMethodResponse(method model.PaymentMethod) paymentMethodResponse {
	return contract.MapPaymentMethodResponse(method)
}

func mapPaymentMethodResponses(methods []model.PaymentMethod) []paymentMethodResponse {
	return contract.MapPaymentMethodResponses(methods)
}

func formatDateOnly(value *time.Time) *string {
	return contract.FormatDateOnly(value)
}

func validateSubscriptionIcon(icon string) bool {
	return contract.ValidateSubscriptionIcon(icon)
}

func validateIcon(icon string) bool {
	return contract.ValidateIcon(icon)
}

func validateSubscriptionAmount(amount float64) bool {
	return contract.ValidateSubscriptionAmount(amount)
}

// subscriptionAmountError explains why validateSubscriptionAmount rejected an
// amount, so an over-the-limit value is not reported as a negative one.
func subscriptionAmountError(amount float64) error {
	if contract.SubscriptionAmountNonFinite(amount) {
		return errors.New("amount must be finite")
	}
	if contract.SubscriptionAmountTooLarge(amount) {
		return errors.New("amount is too large")
	}
	return errors.New("amount must not be negative")
}
