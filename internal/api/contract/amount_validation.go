package contract

import "github.com/kasuha07/subdux/internal/service/money"

// AmountValidation is the canonical subscription amount classification shared
// by API transports and the service layer.
type AmountValidation = money.AmountValidation

const (
	AmountValid        = money.AmountValid
	AmountNonFinite    = money.AmountNonFinite
	AmountNegative     = money.AmountNegative
	AmountAboveMaximum = money.AmountAboveMaximum
)

// MaxSubscriptionAmount is the conservative upper bound for a storable
// subscription amount. It is money.MaxAmount, which keeps all supported
// currency grids exactly quantizable; see that constant's doc comment.
const MaxSubscriptionAmount = money.MaxAmount

// ValidateSubscriptionAmount classifies an incoming subscription amount.
func ValidateSubscriptionAmount(amount float64) AmountValidation {
	return money.ValidateAmount(amount)
}

// SubscriptionAmountErrorCode maps a rejected amount to the stable API code
// shared by REST and MCP. Valid amounts have no error code.
func SubscriptionAmountErrorCode(validation AmountValidation) string {
	switch validation {
	case AmountNonFinite:
		return "amount_must_be_finite"
	case AmountNegative:
		return "amount_must_not_be_negative"
	case AmountAboveMaximum:
		return "amount_too_large"
	default:
		return ""
	}
}

// SubscriptionAmountErrorMessage maps a rejected amount to the human-readable
// message used by MCP JSON-RPC errors. Valid amounts have no error message.
func SubscriptionAmountErrorMessage(validation AmountValidation) string {
	switch validation {
	case AmountNonFinite:
		return "amount must be finite"
	case AmountNegative:
		return "amount must not be negative"
	case AmountAboveMaximum:
		return "amount is too large"
	default:
		return ""
	}
}
