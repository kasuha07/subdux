package money

import (
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

func TestRound(t *testing.T) {
	cases := []struct {
		name     string
		amount   float64
		currency string
		want     float64
	}{
		{"half up at midpoint below float form", 1.005, "USD", 1.01},
		{"classic banker trap", 2.675, "USD", 2.68},
		{"half away from zero negative", -1.005, "USD", -1.01},
		{"accumulated noise", 0.1 + 0.2, "USD", 0.3},
		{"zero decimal currency", 1234.5, "JPY", 1235},
		{"zero decimal currency down", 1234.4, "JPY", 1234},
		{"three decimal currency", 1.2345, "KWD", 1.235},
		{"four decimal currency", 1.23445, "CLF", 1.2345},
		{"four decimal currency negative", -1.23445, "UYW", -1.2345},
		{"already exact", 9.99, "USD", 9.99},
		{"scaled overflow collapses", 1e307, "USD", 0},
		{"negative scaled overflow collapses", -1e307, "USD", 0},
		{"nan collapses", math.NaN(), "USD", 0},
		{"inf collapses", math.Inf(1), "USD", 0},
		{"neg inf collapses", math.Inf(-1), "USD", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Round(tc.amount, tc.currency); got != tc.want {
				t.Errorf("Round(%v, %q) = %v, want %v", tc.amount, tc.currency, got, tc.want)
			}
		})
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

// TestMaxAmountStaysExactAtFourDecimalScale pins the rationale in MaxAmount's
// doc comment: even CLF/UYW's four-decimal minor unit (amount*10000) must keep
// MaxAmount within float64's exact-integer range (1<<53), and Round must still
// be a no-op on it rather than silently degrading.
func TestMaxAmountStaysExactAtFourDecimalScale(t *testing.T) {
	scaled := MaxAmount * 10000
	if scaled >= (1 << 53) {
		t.Fatalf("MaxAmount * 10000 = %v, want < 1<<53 (%v) to stay exact", scaled, float64(int64(1)<<53))
	}
	if got := Round(MaxAmount, "CLF"); got != MaxAmount {
		t.Errorf("Round(MaxAmount, %q) = %v, want %v", "CLF", got, MaxAmount)
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
