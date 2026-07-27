// Package money pins down one quantization policy for monetary float64
// amounts: the ISO 4217 minor unit of the amount's currency. Amounts stay
// float64 across the app; aggregation, change detection, and display must
// all round and compare through these helpers so they agree with each other.
package money

import (
	"math"
	"strconv"
	"strings"
)

// currencyExponents lists ISO 4217 currencies whose minor unit is not the
// default 2 decimal places.
var currencyExponents = map[string]int{
	"BIF": 0, "CLP": 0, "DJF": 0, "GNF": 0, "ISK": 0, "JPY": 0,
	"KMF": 0, "KRW": 0, "PYG": 0, "RWF": 0, "UGX": 0, "VND": 0,
	"VUV": 0, "XAF": 0, "XOF": 0, "XPF": 0,
	"BHD": 3, "IQD": 3, "JOD": 3, "KWD": 3, "LYD": 3, "OMR": 3, "TND": 3,
}

// Exponent returns the number of minor-unit decimal places for an ISO 4217
// currency code. Unknown or empty codes fall back to 2.
func Exponent(currency string) int {
	if exp, ok := currencyExponents[strings.ToUpper(strings.TrimSpace(currency))]; ok {
		return exp
	}
	return 2
}

// IsFinite reports whether v is a usable amount (not NaN or ±Inf).
func IsFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// MaxAmount is the largest amount these helpers can quantize correctly. Past
// it Round stops behaving: scaling by the minor unit overflows to +Inf near
// math.MaxFloat64 (collapsing the amount to 0), and beyond roughly 9e13 the
// scaled value exceeds float64's exact-integer range (1<<53) so rounding
// silently degrades to identity. 1e12 keeps every currency's minor-unit grid
// exact — even KWD's three-decimal scale (amount*1000) stays well under
// 1<<53 — and still sits far above any real subscription price. Callers that
// accept amounts from outside the app (API requests, imported files) must
// reject values above MaxAmount before they ever reach Round.
const MaxAmount = 1_000_000_000_000

// Round quantizes amount to the currency's minor unit, rounding half away
// from zero. Non-finite input collapses to 0 so it can never poison an
// aggregate; reject it earlier with IsFinite where the value is user input.
func Round(amount float64, currency string) float64 {
	if !IsFinite(amount) {
		return 0
	}
	scale := pow10(Exponent(currency))
	// The nudge keeps decimal halves whose float64 form sits just below the
	// midpoint (1.005 is stored as 1.00499...) rounding up as a human expects.
	rounded := math.Round(amount*scale+math.Copysign(1e-9, amount)) / scale
	// amount*scale can overflow to ±Inf near math.MaxFloat64; keep the
	// non-finite-in, zero-out contract on the output side too.
	if !IsFinite(rounded) {
		return 0
	}
	return rounded
}

// Cmp compares two amounts at minor-unit resolution: -1 when a < b, 0 when
// they land on the same minor-unit grid point, 1 when a > b. Sub-minor-unit
// float noise (converted or accumulated values) therefore never registers
// as a difference.
func Cmp(a, b float64, currency string) int {
	ra, rb := Round(a, currency), Round(b, currency)
	switch {
	case ra < rb:
		return -1
	case ra > rb:
		return 1
	}
	return 0
}

// Equal reports whether a and b are the same monetary value at minor-unit
// resolution.
func Equal(a, b float64, currency string) bool {
	return Cmp(a, b, currency) == 0
}

// Diff returns the minor-unit difference a - b with both operands and the
// result quantized, so callers get an exact grid value rather than raw
// float subtraction noise.
func Diff(a, b float64, currency string) float64 {
	return Round(Round(a, currency)-Round(b, currency), currency)
}

// Format renders amount rounded and zero-padded to the currency's minor
// unit, e.g. "9.99" (USD), "1235" (JPY), "1.200" (KWD).
func Format(amount float64, currency string) string {
	return strconv.FormatFloat(Round(amount, currency), 'f', Exponent(currency), 64)
}

func pow10(exp int) float64 {
	return math.Pow10(exp)
}
