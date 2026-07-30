package contract

import (
	"math"
	"testing"
)

func TestValidateSubscriptionAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   bool
	}{
		{name: "zero is storable", amount: 0, want: true},
		{name: "positive is storable", amount: 9.99, want: true},
		{name: "negative is rejected", amount: -0.01, want: false},
		{name: "nan is rejected", amount: math.NaN(), want: false},
		{name: "positive infinity is rejected", amount: math.Inf(1), want: false},
		{name: "negative infinity is rejected", amount: math.Inf(-1), want: false},
		{name: "the maximum is storable", amount: MaxSubscriptionAmount, want: true},
		{name: "just above the maximum is rejected", amount: MaxSubscriptionAmount + 0.01, want: false},
		// Past float64's exact-integer range, scaling to minor units degrades to
		// identity; near math.MaxFloat64 it overflows to +Inf and the amount
		// silently collapses to 0.
		{name: "beyond exact minor-unit precision is rejected", amount: 9e13, want: false},
		{name: "amount that overflows rounding is rejected", amount: 1.8e306, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateSubscriptionAmount(tt.amount); got != tt.want {
				t.Fatalf("ValidateSubscriptionAmount(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestSubscriptionAmountTooLarge(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   bool
	}{
		{name: "the maximum is not too large", amount: MaxSubscriptionAmount, want: false},
		{name: "just above the maximum is too large", amount: MaxSubscriptionAmount + 0.01, want: true},
		{name: "positive infinity is too large", amount: math.Inf(1), want: true},
		// Rejected for other reasons; they must not claim to be over the bound.
		{name: "negative is not too large", amount: -0.01, want: false},
		{name: "nan is not too large", amount: math.NaN(), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SubscriptionAmountTooLarge(tt.amount); got != tt.want {
				t.Fatalf("SubscriptionAmountTooLarge(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestSubscriptionAmountNonFinite(t *testing.T) {
	if !SubscriptionAmountNonFinite(math.NaN()) {
		t.Fatal("NaN should be classified as non-finite")
	}
	if !SubscriptionAmountNonFinite(math.Inf(1)) {
		t.Fatal("+Inf should be classified as non-finite")
	}
	if SubscriptionAmountNonFinite(MaxSubscriptionAmount) {
		t.Fatal("MaxSubscriptionAmount should be finite")
	}
}
