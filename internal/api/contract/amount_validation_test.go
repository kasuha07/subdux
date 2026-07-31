package contract

import (
	"math"
	"testing"
)

func TestValidateSubscriptionAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   AmountValidation
	}{
		{name: "zero is storable", amount: 0, want: AmountValid},
		{name: "positive is storable", amount: 9.99, want: AmountValid},
		{name: "negative is rejected", amount: -0.01, want: AmountNegative},
		{name: "nan is rejected", amount: math.NaN(), want: AmountNonFinite},
		{name: "positive infinity is too large", amount: math.Inf(1), want: AmountAboveMaximum},
		{name: "negative infinity is non-finite", amount: math.Inf(-1), want: AmountNonFinite},
		{name: "the maximum is storable", amount: MaxSubscriptionAmount, want: AmountValid},
		{name: "just above the maximum is rejected", amount: MaxSubscriptionAmount + 0.01, want: AmountAboveMaximum},
		// Past float64's exact-integer range, scaling to minor units degrades to
		// identity; near math.MaxFloat64 it overflows to +Inf and the amount
		// silently collapses to 0.
		{name: "beyond exact minor-unit precision is rejected", amount: 9e13, want: AmountAboveMaximum},
		{name: "amount that overflows rounding is rejected", amount: 1.8e306, want: AmountAboveMaximum},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateSubscriptionAmount(tt.amount); got != tt.want {
				t.Fatalf("ValidateSubscriptionAmount(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestSubscriptionAmountErrorContract(t *testing.T) {
	tests := []struct {
		name        string
		validation  AmountValidation
		wantCode    string
		wantMessage string
	}{
		{name: "valid", validation: AmountValid},
		{name: "non-finite", validation: AmountNonFinite, wantCode: "amount_must_be_finite", wantMessage: "amount must be finite"},
		{name: "negative", validation: AmountNegative, wantCode: "amount_must_not_be_negative", wantMessage: "amount must not be negative"},
		{name: "too large", validation: AmountAboveMaximum, wantCode: "amount_too_large", wantMessage: "amount is too large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubscriptionAmountErrorCode(tt.validation); got != tt.wantCode {
				t.Fatalf("SubscriptionAmountErrorCode(%v) = %q, want %q", tt.validation, got, tt.wantCode)
			}
			if got := SubscriptionAmountErrorMessage(tt.validation); got != tt.wantMessage {
				t.Fatalf("SubscriptionAmountErrorMessage(%v) = %q, want %q", tt.validation, got, tt.wantMessage)
			}
		})
	}
}
