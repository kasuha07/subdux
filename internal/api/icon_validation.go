package api

import "github.com/kasuha07/subdux/internal/api/contract"

func validateIcon(icon string) bool {
	return contract.ValidateIcon(icon)
}

func validateSubscriptionIcon(icon string) bool {
	return contract.ValidateSubscriptionIcon(icon)
}

func validateSubscriptionAmount(amount float64) bool {
	return contract.ValidateSubscriptionAmount(amount)
}

// subscriptionAmountErrorCode names why validateSubscriptionAmount rejected an
// amount, so an over-the-limit value is not reported as a negative one.
func subscriptionAmountErrorCode(amount float64) string {
	if contract.SubscriptionAmountNonFinite(amount) {
		return "amount_must_be_finite"
	}
	if contract.SubscriptionAmountTooLarge(amount) {
		return "amount_too_large"
	}
	return "amount_must_not_be_negative"
}
