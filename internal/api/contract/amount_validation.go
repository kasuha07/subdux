package contract

import "github.com/kasuha07/subdux/internal/pkg/money"

// MaxSubscriptionAmount is the conservative upper bound for a storable
// subscription amount. It is money.MaxAmount, which keeps all supported
// currency grids exactly quantizable; see that constant's doc comment.
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
