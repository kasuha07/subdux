import { CURRENCY_EXPONENTS } from "@/lib/currency-metadata.generated"

// Mirrors the quantization policy from internal/service/money on the Go side: the
// ISO 4217 minor unit of the amount's currency, rounding half away from zero.
// Client-side conversions/normalizations round the same way so displayed
// amounts line up with server-computed totals.

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
  return CURRENCY_EXPONENTS[normalized] ?? 2
}

// roundAmount quantizes amount to the currency's minor unit, rounding half
// away from zero. Non-finite input collapses to 0 so it can never poison an
// aggregate.
//
// The number's shortest decimal representation preserves decimal midpoints such
// as 1.005 without applying an epsilon that can move nearby values across them.
export function roundAmount(amount: number, currency: string): number {
  if (!Number.isFinite(amount)) {
    return 0
  }

  const exponent = currencyExponent(currency)
  const scale = 10 ** exponent
  const [mantissa, scientificExponent = "0"] = Math.abs(amount)
    .toString()
    .toLowerCase()
    .split("e")
  const [integerDigits, fractionDigits = ""] = mantissa.split(".")
  const coefficient = BigInt(integerDigits + fractionDigits)
  const decimalShift = Number(scientificExponent) - fractionDigits.length + exponent

  if (decimalShift >= 0) {
    return amount === 0 ? 0 : amount
  }

  const divisor = 10n ** BigInt(-decimalShift)
  const quotient = coefficient / divisor
  const remainder = coefficient % divisor
  const minorUnits = quotient + (remainder * 2n >= divisor ? 1n : 0n)

  if (minorUnits === 0n) {
    return 0
  }

  const rounded = Number(minorUnits) / scale
  return amount < 0 ? -rounded : rounded
}
