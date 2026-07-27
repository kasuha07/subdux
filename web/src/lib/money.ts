// Mirrors the quantization policy from internal/pkg/money on the Go side: the
// ISO 4217 minor unit of the amount's currency, rounding half away from zero.
// Client-side conversions/normalizations round the same way so displayed
// amounts line up with server-computed totals.

// Currencies whose minor unit is 0 decimal places.
const ZERO_DECIMAL_CURRENCIES = new Set([
  "BIF", "CLP", "DJF", "GNF", "ISK", "JPY",
  "KMF", "KRW", "PYG", "RWF", "UGX", "VND",
  "VUV", "XAF", "XOF", "XPF",
])

// Currencies whose minor unit is 3 decimal places.
const THREE_DECIMAL_CURRENCIES = new Set([
  "BHD", "IQD", "JOD", "KWD", "LYD", "OMR", "TND",
])

// A well-formed ISO 4217 code is exactly three ASCII letters. This is the
// precondition for Intl.NumberFormat's `currency` option: anything else
// (empty, symbols, wrong length, stray whitespace) makes it throw a
// RangeError instead of just producing an unrecognized-currency format.
const WELL_FORMED_CURRENCY_CODE_PATTERN = /^[A-Za-z]{3}$/

// isWellFormedCurrencyCode reports whether currency, once trimmed, is a
// syntactically valid ISO 4217 code. It does not check that the code is a
// real, assigned currency.
export function isWellFormedCurrencyCode(currency: string): boolean {
  return WELL_FORMED_CURRENCY_CODE_PATTERN.test(currency.trim())
}

// currencyExponent returns the number of minor-unit decimal places for an
// ISO 4217 currency code. Unknown or empty codes fall back to 2.
export function currencyExponent(currency: string): number {
  const normalized = currency.trim().toUpperCase()

  if (ZERO_DECIMAL_CURRENCIES.has(normalized)) {
    return 0
  }

  if (THREE_DECIMAL_CURRENCIES.has(normalized)) {
    return 3
  }

  return 2
}

// roundAmount quantizes amount to the currency's minor unit, rounding half
// away from zero. Non-finite input collapses to 0 so it can never poison an
// aggregate.
//
// JS Math.round rounds -0.5 toward +0 (half up, not half away from zero), so
// negative amounts are negated before rounding and negated back after.
export function roundAmount(amount: number, currency: string): number {
  if (!Number.isFinite(amount)) {
    return 0
  }

  const scale = 10 ** currencyExponent(currency)

  // The nudge keeps decimal halves whose float64 form sits just below the
  // midpoint (1.005 is stored as 1.00499...) rounding up as a human expects.
  if (amount < 0) {
    return -Math.round(-amount * scale + 1e-9) / scale
  }

  return Math.round(amount * scale + 1e-9) / scale
}
