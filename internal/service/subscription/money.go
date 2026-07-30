package subscription

import (
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
)

func validateSubscriptionAmount(amount float64) error {
	switch money.ValidateAmount(amount) {
	case money.AmountValid:
		return nil
	case money.AmountNegative:
		return ErrAmountMustNotBeNegative
	case money.AmountAboveMaximum:
		return ErrAmountTooLarge
	default:
		return ErrAmountMustBeFinite
	}
}

// ValidateBillingAmount is the canonical invariant for a newly persisted
// subscription amount. The stored input uses MaxAmount, while schedule-derived
// values use the wider currency-aware minor-unit range.
func ValidateBillingAmount(amount float64, currency string, draft BillingDraft) error {
	if err := validateSubscriptionAmount(amount); err != nil {
		return err
	}
	return validateBillingDerivedAmount(amount, currency, draft)
}

// validateBillingDerivedAmount validates only the schedule-derived result. It
// is used when an existing amount is grandfathered but its currency or billing
// schedule is actually changed.
func validateBillingDerivedAmount(amount float64, currency string, draft BillingDraft) error {
	factor := billingDraftMonthlyFactor(draft)
	if factor <= 0 {
		return nil
	}
	monthly, err := roundDerivedAmount(amount*factor, currency)
	if err != nil {
		return err
	}
	_, err = multiplyAggregateAmount(monthly, 12, currency)
	return err
}

func billingDraftMonthlyFactor(draft BillingDraft) float64 {
	return subscriptionMonthlyFactor(model.Subscription{
		BillingType:    draft.BillingType,
		RecurrenceType: draft.RecurrenceType,
		IntervalCount:  draft.IntervalCount,
		IntervalUnit:   draft.IntervalUnit,
	})
}

// roundDerivedAmount keeps computed values on the target currency's minor-unit
// grid and rejects values that can no longer be represented safely. Returning a
// domain error is important: silently turning an overflow into zero would make
// a valid subscription appear free.
func roundDerivedAmount(amount float64, currency string) (float64, error) {
	if err := validateDerivedAmount(amount); err != nil {
		return 0, err
	}
	rounded, ok := money.RoundAggregateChecked(amount, currency)
	if !ok {
		return 0, ErrAmountTooLarge
	}
	return rounded, nil
}

func validateDerivedAmount(amount float64) error {
	if !money.IsFinite(amount) {
		return ErrAmountMustBeFinite
	}
	return nil
}

func roundAggregateAmount(amount float64, currency string) (float64, error) {
	if !money.IsFinite(amount) {
		return 0, ErrAmountMustBeFinite
	}
	rounded, ok := money.RoundAggregateChecked(amount, currency)
	if !ok {
		return 0, ErrAmountTooLarge
	}
	return rounded, nil
}

func addAggregateAmounts(total, amount float64, currency string) (float64, error) {
	if !money.IsFinite(total) || !money.IsFinite(amount) {
		return 0, ErrAmountMustBeFinite
	}
	sum, ok := money.AddAggregateChecked(total, amount, currency)
	if !ok {
		return 0, ErrAmountTooLarge
	}
	return sum, nil
}

func multiplyAggregateAmount(amount float64, factor int64, currency string) (float64, error) {
	if !money.IsFinite(amount) {
		return 0, ErrAmountMustBeFinite
	}
	product, ok := money.MultiplyAggregateChecked(amount, factor, currency)
	if !ok {
		return 0, ErrAmountTooLarge
	}
	return product, nil
}
