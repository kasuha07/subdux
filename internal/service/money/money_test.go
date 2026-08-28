package money

import (
	"errors"
	"math"
	"testing"
)

func TestExponent(t *testing.T) {
	cases := []struct {
		currency string
		want     int
	}{
		{"USD", 2},
		{"EUR", 2},
		{"JPY", 0},
		{"UYI", 0},
		{"jpy", 0},
		{" krw ", 0},
		{"KWD", 3},
		{"bhd", 3},
		{"CLF", 4},
		{" uyw ", 4},
		{"", 2},
		{"XXX", 2},
	}
	for _, tc := range cases {
		if got := Exponent(tc.currency); got != tc.want {
			t.Errorf("Exponent(%q) = %d, want %d", tc.currency, got, tc.want)
		}
	}
}

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   AmountValidation
	}{
		{name: "zero", amount: 0, want: AmountValid},
		{name: "positive", amount: 9.99, want: AmountValid},
		{name: "negative", amount: -1, want: AmountNegative},
		{name: "nan", amount: math.NaN(), want: AmountNonFinite},
		{name: "negative infinity", amount: math.Inf(-1), want: AmountNonFinite},
		{name: "positive infinity", amount: math.Inf(1), want: AmountAboveMaximum},
		{name: "above maximum", amount: MaxAmount + 0.01, want: AmountAboveMaximum},
		{name: "maximum", amount: MaxAmount, want: AmountValid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateAmount(tc.amount); got != tc.want {
				t.Fatalf("ValidateAmount(%v) = %v, want %v", tc.amount, got, tc.want)
			}
		})
	}
}

func TestRound(t *testing.T) {
	midpoint := 1.005
	cases := []struct {
		name     string
		amount   float64
		currency string
		want     float64
	}{
		{"below midpoint", math.Nextafter(midpoint, math.Inf(-1)), "USD", 1.00},
		{"midpoint below after scaling", midpoint, "USD", 1.01},
		{"above midpoint", math.Nextafter(midpoint, math.Inf(1)), "USD", 1.01},
		{"not a midpoint", 1.004999999999, "USD", 1.00},
		{"negative below midpoint magnitude", math.Nextafter(-midpoint, math.Inf(1)), "USD", -1.00},
		{"negative midpoint", -midpoint, "USD", -1.01},
		{"negative above midpoint magnitude", math.Nextafter(-midpoint, math.Inf(-1)), "USD", -1.01},
		{"scaled multiplication collision below midpoint", math.Nextafter(0.025, math.Inf(-1)), "USD", 0.02},
		{"scaled multiplication collision midpoint", 0.025, "USD", 0.03},
		{"classic banker trap", 2.675, "USD", 2.68},
		{"half away from zero negative", -1.005, "USD", -1.01},
		{"accumulated noise", 0.1 + 0.2, "USD", 0.3},
		{"zero decimal currency", 1234.5, "JPY", 1235},
		{"zero decimal currency down", 1234.4, "JPY", 1234},
		{"three decimal currency", 1.2345, "KWD", 1.235},
		{"four decimal currency", 1.23445, "CLF", 1.2345},
		{"four decimal currency negative", -1.23445, "UYW", -1.2345},
		{"already exact", 9.99, "USD", 9.99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Round(tc.amount, tc.currency); got != tc.want {
				t.Errorf("Round(%v, %q) = %v, want %v", tc.amount, tc.currency, got, tc.want)
			}
		})
	}
}

func TestRoundAggregateCheckedUsesStrictMidpointRounding(t *testing.T) {
	midpoint := 1.005
	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{name: "below midpoint", amount: math.Nextafter(midpoint, math.Inf(-1)), want: 1.00},
		{name: "midpoint", amount: midpoint, want: 1.01},
		{name: "above midpoint", amount: math.Nextafter(midpoint, math.Inf(1)), want: 1.01},
		{name: "more than one ulp below midpoint", amount: 1.004999999999, want: 1.00},
		{name: "negative below midpoint magnitude", amount: math.Nextafter(-midpoint, math.Inf(1)), want: -1.00},
		{name: "negative midpoint", amount: -midpoint, want: -1.01},
		{name: "negative above midpoint magnitude", amount: math.Nextafter(-midpoint, math.Inf(-1)), want: -1.01},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := RoundAggregateChecked(tt.amount, "USD")
			if !ok || got != tt.want {
				t.Fatalf("RoundAggregateChecked(%0.15f) = %v, %v; want %v, true", tt.amount, got, ok, tt.want)
			}
		})
	}
}

func TestRoundCheckedRejectsValuesOutsidePersistedInputRange(t *testing.T) {
	if got, ok := RoundChecked(MaxAmount, "CLF"); !ok || got != MaxAmount {
		t.Fatalf("RoundChecked(MaxAmount, CLF) = %v, %v; want %v, true", got, ok, MaxAmount)
	}
	if got, ok := RoundChecked(MaxAmount+0.0001, "CLF"); ok || got != 0 {
		t.Fatalf("RoundChecked(value above MaxAmount) = %v, %v; want 0, false", got, ok)
	}
	if got, ok := RoundChecked(math.Inf(1), "USD"); ok || got != 0 {
		t.Fatalf("RoundChecked(+Inf) = %v, %v; want 0, false", got, ok)
	}
}

func TestAggregateArithmeticExceedsSubscriptionMaximumSafely(t *testing.T) {
	sum, ok := AddAggregateChecked(MaxAmount, MaxAmount, "USD")
	if !ok || sum != 1_000_000_000_000 {
		t.Fatalf("AddAggregateChecked() = %v, %v; want 1000000000000, true", sum, ok)
	}

	yearly, ok := MultiplyAggregateChecked(sum, 12, "USD")
	if !ok || yearly != 12_000_000_000_000 {
		t.Fatalf("MultiplyAggregateChecked() = %v, %v; want 12000000000000, true", yearly, ok)
	}
}

func TestAggregateArithmeticRejectsUnsafeMinorUnitRange(t *testing.T) {
	if got, ok := RoundAggregateChecked(520_000_000_000, "CLF"); !ok || got != 520_000_000_000 {
		t.Fatalf("RoundAggregateChecked(CLF safe aggregate) = %v, %v; want 520000000000, true", got, ok)
	}
	if got, ok := RoundAggregateChecked(900_000_000_000, "CLF"); !ok || got != 900_000_000_000 {
		t.Fatalf("RoundAggregateChecked(CLF legacy-safe aggregate) = %v, %v; want 900000000000, true", got, ok)
	}
	if got, ok := RoundAggregateChecked(901_000_000_000, "CLF"); ok || got != 0 {
		t.Fatalf("RoundAggregateChecked(CLF minor-unit overflow) = %v, %v; want 0, false", got, ok)
	}
	if got, ok := AddAggregateChecked(MaxAmount, MaxAmount, "CLF"); ok || got != 0 {
		t.Fatalf("AddAggregateChecked(CLF unsafe range) = %v, %v; want 0, false", got, ok)
	}
	if got, ok := RoundAggregateChecked(math.Inf(1), "USD"); ok || got != 0 {
		t.Fatalf("RoundAggregateChecked(+Inf) = %v, %v; want 0, false", got, ok)
	}
}

// TestMaxAmountKeepsAdjacentFourDecimalGridValuesDistinct verifies the
// property the previous 2^53-only proof missed: at the accepted boundary,
// neighboring CLF/UYW decimal grid values must not map to one float64 value.
func TestMaxAmountKeepsAdjacentFourDecimalGridValuesDistinct(t *testing.T) {
	ulp := math.Nextafter(float64(MaxAmount), math.Inf(1)) - float64(MaxAmount)
	if ulp >= 0.0001 {
		t.Fatalf("float64 ULP at MaxAmount = %v, want below one CLF grid unit", ulp)
	}

	previous := Round(MaxAmount-0.0002, "CLF")
	last := Round(MaxAmount-0.0001, "CLF")
	maximum := Round(MaxAmount, "CLF")
	if previous == last || last == maximum || previous == maximum {
		t.Fatalf("adjacent CLF grid values collided near MaxAmount: %v, %v, %v", previous, last, maximum)
	}

	if scaled := MaxAmount * 10000; scaled >= (1 << 53) {
		t.Fatalf("MaxAmount * 10000 = %v, want < 1<<53 (%v)", scaled, float64(int64(1)<<53))
	}
}

func TestCmpAndEqual(t *testing.T) {
	if !Equal(0.1+0.2, 0.3, "USD") {
		t.Error("Equal should absorb float accumulation noise")
	}
	if !Equal(10.1+1e-13, 10.1, "USD") {
		t.Error("Equal should absorb conversion noise below the minor unit")
	}
	if Cmp(10.11, 10.10, "USD") != 1 {
		t.Error("Cmp should detect a one-cent increase")
	}
	if Cmp(10.10, 10.11, "USD") != -1 {
		t.Error("Cmp should detect a one-cent decrease")
	}
	if !Equal(100.4, 100, "JPY") {
		t.Error("JPY comparison should ignore sub-unit fractions")
	}
	if Cmp(100.5, 100, "JPY") != 1 {
		t.Error("JPY midpoint should round up and register as an increase")
	}
	if Cmp(1.236, 1.235, "KWD") != 1 {
		t.Error("KWD should resolve differences at the third decimal")
	}
	if !Equal(1.2351, 1.2349, "KWD") {
		t.Error("KWD amounts on the same grid point should compare equal")
	}
	if Cmp(1.2346, 1.2345, "CLF") != 1 {
		t.Error("CLF should resolve differences at the fourth decimal")
	}
	if !Equal(1.23451, 1.23449, "UYW") {
		t.Error("UYW amounts on the same grid point should compare equal")
	}
}

func TestDiff(t *testing.T) {
	if got := Diff(10.15, 10.05, "USD"); got != 0.1 {
		t.Errorf("Diff(10.15, 10.05) = %v, want 0.1", got)
	}
	if got := Diff(10.1+1e-13, 10.1, "USD"); got != 0 {
		t.Errorf("Diff with sub-cent noise = %v, want 0", got)
	}
	if got := Diff(100, 100.4, "JPY"); got != 0 {
		t.Errorf("Diff(100, 100.4 JPY) = %v, want 0", got)
	}
}

func TestCheckedComparisonRejectsUnsafeAmounts(t *testing.T) {
	const legacyMonthlyAmount = 27_393_187_500_000.0

	if got, ok := CmpChecked(3_000, legacyMonthlyAmount, "CLF"); ok || got != 0 {
		t.Fatalf("CmpChecked() = %d, %v; want 0, false for unsafe legacy amount", got, ok)
	}
	if got, ok := DiffChecked(3_000, legacyMonthlyAmount, "CLF"); ok || got != 0 {
		t.Fatalf("DiffChecked() = %v, %v; want 0, false for unsafe legacy amount", got, ok)
	}

	if got, ok := CmpChecked(10.11, 10.10, "USD"); !ok || got != 1 {
		t.Fatalf("CmpChecked() = %d, %v; want 1, true for a valid cent increase", got, ok)
	}
	if got, ok := DiffChecked(10.11, 10.10, "USD"); !ok || got != 0.01 {
		t.Fatalf("DiffChecked() = %v, %v; want 0.01, true", got, ok)
	}
}

func TestFormat(t *testing.T) {
	cases := []struct {
		amount   float64
		currency string
		want     string
	}{
		{9.99, "USD", "9.99"},
		{0.1 + 0.2, "USD", "0.30"},
		{1234.5, "JPY", "1235"},
		{1.2, "KWD", "1.200"},
		{1.2, "CLF", "1.2000"},
		{1.23445, "UYW", "1.2345"},
		{5, "EUR", "5.00"},
	}
	for _, tc := range cases {
		if got := Format(tc.amount, tc.currency); got != tc.want {
			t.Errorf("Format(%v, %q) = %q, want %q", tc.amount, tc.currency, got, tc.want)
		}
	}
}

func TestFormatCheckedRejectsUnsafeAmounts(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
	}{
		{name: "nan", amount: math.NaN()},
		{name: "positive infinity", amount: math.Inf(1)},
		{name: "negative infinity", amount: math.Inf(-1)},
		{name: "scaled overflow", amount: 1e307},
		{name: "minor unit representability overflow", amount: 901_000_000_000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatChecked(tt.amount, "CLF")
			if !errors.Is(err, ErrUnsafeFormat) || got != "" {
				t.Fatalf("FormatChecked(%v) = %q, %v; want empty ErrUnsafeFormat", tt.amount, got, err)
			}
		})
	}
}
