// Package money owns the subscription domain's monetary policy: ISO 4217
// minor-unit quantization, amount validation, and the maximum safe float64
// amount. It lives in the service layer so every business entry point shares
// the same invariant instead of relying on transport validation.
package money

import (
	"math"
	"strconv"
	"strings"
)

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

// MaxAmount is a conservative ceiling for subscription amounts. At this
// magnitude float64 spacing remains below the four-decimal minor-unit grid,
// so adjacent CLF/UYW values remain distinct after conversion to float64 and
// quantization. It also keeps four-decimal scaling well below 2^53.
//
// ValidateAmount applies this ceiling only to newly persisted inputs. Derived
// values and aggregate totals use the separate checked minor-unit helpers;
// Round remains a defensive formatting helper for legacy/untrusted data.
const MaxAmount = 500_000_000_000

// maxAggregateMinorUnits is the largest integer minor-unit value that a
// float64 can represent exactly. Derived and aggregate values are intentionally
// not capped by MaxAmount: they are safe when the concrete currency-scaled
// result can round-trip through this integer range.
const maxAggregateMinorUnits = int64(1<<53 - 1)

// AmountValidation classifies a value against the subscription amount
// invariant. The order intentionally treats positive infinity as too large,
// matching the transport error shown for values above MaxAmount.
type AmountValidation uint8

const (
	AmountValid AmountValidation = iota
	AmountNonFinite
	AmountNegative
	AmountAboveMaximum
)

// ValidateAmount reports why amount is not storable as a subscription amount.
func ValidateAmount(amount float64) AmountValidation {
	if math.IsInf(amount, 1) || amount > MaxAmount {
		return AmountAboveMaximum
	}
	if !IsFinite(amount) {
		return AmountNonFinite
	}
	if amount < 0 {
		return AmountNegative
	}
	return AmountValid
}

// IsWithinMaxMagnitude reports whether a finite value, including a negative
// difference used by comparison helpers, fits the exact amount ceiling.
func IsWithinMaxMagnitude(amount float64) bool {
	return IsFinite(amount) && math.Abs(amount) <= MaxAmount
}

// Round quantizes amount to the currency's minor unit, rounding half away
// from zero. Non-finite input collapses to 0 so it cannot poison an aggregate;
// service input boundaries must call ValidateAmount first.
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

// RoundChecked quantizes a value only when its magnitude is within the
// persisted-input ceiling. A false result is a domain failure, not a zero
// amount, and callers must propagate or skip it rather than showing 0.
func RoundChecked(amount float64, currency string) (float64, bool) {
	if !IsWithinMaxMagnitude(amount) {
		return 0, false
	}
	rounded := Round(amount, currency)
	if !IsWithinMaxMagnitude(rounded) {
		return 0, false
	}
	return rounded, true
}

// RoundAggregateChecked quantizes a derived or aggregate value without
// applying the persisted per-subscription MaxAmount ceiling. It only succeeds
// while the concrete currency-scaled minor-unit result remains exactly
// representable and round-trips to the same integer.
func RoundAggregateChecked(amount float64, currency string) (float64, bool) {
	minorUnits, scale, ok := aggregateMinorUnits(amount, currency)
	if !ok {
		return 0, false
	}
	return amountFromAggregateMinorUnits(minorUnits, scale)
}

// AddAggregateChecked adds two aggregate values as integer minor units. This
// prevents repeated float64 additions from accumulating sub-minor-unit noise.
func AddAggregateChecked(a, b float64, currency string) (float64, bool) {
	aMinor, scale, ok := aggregateMinorUnits(a, currency)
	if !ok {
		return 0, false
	}
	bMinor, _, ok := aggregateMinorUnits(b, currency)
	if !ok {
		return 0, false
	}
	if bMinor > 0 && aMinor > maxAggregateMinorUnits-bMinor {
		return 0, false
	}
	if bMinor < 0 && aMinor < -maxAggregateMinorUnits-bMinor {
		return 0, false
	}
	return amountFromAggregateMinorUnits(aMinor+bMinor, scale)
}

// MultiplyAggregateChecked multiplies an aggregate by an integer factor using
// checked minor-unit arithmetic.
func MultiplyAggregateChecked(amount float64, factor int64, currency string) (float64, bool) {
	minorUnits, scale, ok := aggregateMinorUnits(amount, currency)
	if !ok {
		return 0, false
	}
	if factor == 0 || minorUnits == 0 {
		return 0, true
	}
	if factor < 0 {
		return 0, false
	}
	if minorUnits > maxAggregateMinorUnits/factor || minorUnits < -maxAggregateMinorUnits/factor {
		return 0, false
	}
	return amountFromAggregateMinorUnits(minorUnits*factor, scale)
}

// Cmp compares two amounts at minor-unit resolution: -1 when a < b, 0 when
// they land on the same minor-unit grid point, 1 when a > b. Sub-minor-unit
// float noise (converted or accumulated values) therefore never registers as
// a difference.
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
// result quantized, so callers get an exact grid value rather than raw float
// subtraction noise.
func Diff(a, b float64, currency string) float64 {
	return Round(Round(a, currency)-Round(b, currency), currency)
}

// Format renders amount rounded and zero-padded to the currency's minor unit,
// e.g. "9.99" (USD), "1235" (JPY), "1.200" (KWD).
func Format(amount float64, currency string) string {
	return strconv.FormatFloat(Round(amount, currency), 'f', Exponent(currency), 64)
}

func pow10(exp int) float64 {
	return math.Pow10(exp)
}

func aggregateMinorUnits(amount float64, currency string) (int64, float64, bool) {
	if !IsFinite(amount) {
		return 0, 0, false
	}
	scale := pow10(Exponent(currency))
	scaled := amount * scale
	if !IsFinite(scaled) {
		return 0, 0, false
	}
	rounded := math.Round(scaled + math.Copysign(1e-9, amount))
	if math.Abs(rounded) > float64(maxAggregateMinorUnits) {
		return 0, 0, false
	}
	minorUnits := int64(rounded)
	if _, ok := amountFromAggregateMinorUnits(minorUnits, scale); !ok {
		return 0, 0, false
	}
	return minorUnits, scale, true
}

func amountFromAggregateMinorUnits(minorUnits int64, scale float64) (float64, bool) {
	amount := float64(minorUnits) / scale
	if !IsFinite(amount) {
		return 0, false
	}

	// Persisted inputs use MaxAmount to guarantee that every adjacent minor-unit
	// value stays distinguishable. Derived values can be larger: validate the
	// concrete result instead, accepting it only when it recovers the exact
	// checked minor-unit integer.
	recovered := math.Round(amount*scale + math.Copysign(1e-9, amount))
	if recovered != float64(minorUnits) {
		return 0, false
	}
	return amount, true
}
