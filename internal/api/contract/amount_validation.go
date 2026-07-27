package contract

import "github.com/kasuha07/subdux/internal/pkg/money"

// MaxSubscriptionAmount is the largest storable subscription amount. It is
// money.MaxAmount, the largest value the money helpers can quantize
// correctly — see that constant's doc comment for the full rationale.
const MaxSubscriptionAmount = money.MaxAmount

// ValidateSubscriptionAmount reports whether an incoming subscription amount is
// storable: a finite, non-negative number no larger than MaxSubscriptionAmount.
// NaN and ±Inf compare false against every bound, so a plain `amount < 0` check
// would let them through and poison every aggregate derived from the
// subscription.
func ValidateSubscriptionAmount(amount float64) bool {
	return money.IsFinite(amount) && amount >= 0 && amount <= MaxSubscriptionAmount
}

// SubscriptionAmountTooLarge reports whether an amount is rejected specifically
// for exceeding MaxSubscriptionAmount, so each transport can surface that
// reason instead of the negative-amount one. Only meaningful for amounts
// ValidateSubscriptionAmount already rejected.
func SubscriptionAmountTooLarge(amount float64) bool {
	return amount > MaxSubscriptionAmount
}
