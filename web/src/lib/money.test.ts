import { describe, expect, it } from "vitest"

import { currencyExponent, isWellFormedCurrencyCode, roundAmount } from "@/lib/money"

describe("isWellFormedCurrencyCode", () => {
  it("accepts a well-formed three-letter code", () => {
    expect(isWellFormedCurrencyCode("USD")).toBe(true)
  })

  it("is case-insensitive", () => {
    expect(isWellFormedCurrencyCode("usd")).toBe(true)
  })

  it("trims surrounding whitespace", () => {
    expect(isWellFormedCurrencyCode(" jpy ")).toBe(true)
  })

  it("rejects an empty code", () => {
    expect(isWellFormedCurrencyCode("")).toBe(false)
  })

  it("rejects a non-letter symbol", () => {
    expect(isWellFormedCurrencyCode("€")).toBe(false)
  })

  it("rejects a two-letter code", () => {
    expect(isWellFormedCurrencyCode("AB")).toBe(false)
  })

  it("rejects a four-letter code", () => {
    expect(isWellFormedCurrencyCode("USDT")).toBe(false)
  })
})

describe("currencyExponent", () => {
  it("defaults to 2 decimal places", () => {
    expect(currencyExponent("USD")).toBe(2)
    expect(currencyExponent("EUR")).toBe(2)
  })

  it("returns 0 for zero-decimal currencies", () => {
    expect(currencyExponent("JPY")).toBe(0)
    expect(currencyExponent("KRW")).toBe(0)
  })

  it("returns 3 for three-decimal currencies", () => {
    expect(currencyExponent("KWD")).toBe(3)
    expect(currencyExponent("BHD")).toBe(3)
  })

  it("is case-insensitive and trims whitespace", () => {
    expect(currencyExponent("jpy")).toBe(0)
    expect(currencyExponent(" krw ")).toBe(0)
    expect(currencyExponent("bhd")).toBe(3)
  })

  it("falls back to 2 for empty or unknown codes", () => {
    expect(currencyExponent("")).toBe(2)
    expect(currencyExponent("XXX")).toBe(2)
  })
})

describe("roundAmount", () => {
  it("rounds a half up for a positive amount whose float form sits just below the midpoint", () => {
    expect(roundAmount(1.005, "USD")).toBe(1.01)
  })

  it("rounds a half away from zero for a negative amount", () => {
    expect(roundAmount(-1.005, "USD")).toBe(-1.01)
  })

  it("absorbs float accumulation noise", () => {
    expect(roundAmount(0.1 + 0.2, "USD")).toBe(0.3)
  })

  it("rounds to a whole number for a zero-decimal currency", () => {
    expect(roundAmount(1234.5, "JPY")).toBe(1235)
    expect(roundAmount(1234.4, "JPY")).toBe(1234)
  })

  it("rounds an exact negative half away from zero for a zero-decimal currency", () => {
    expect(roundAmount(-0.5, "JPY")).toBe(-1)
  })

  it("rounds to three decimal places for a three-decimal currency", () => {
    expect(roundAmount(1.2345, "KWD")).toBe(1.235)
  })

  it("returns 0 for non-finite input", () => {
    expect(roundAmount(Number.NaN, "USD")).toBe(0)
    expect(roundAmount(Number.POSITIVE_INFINITY, "USD")).toBe(0)
    expect(roundAmount(Number.NEGATIVE_INFINITY, "USD")).toBe(0)
  })

  it("leaves an already-exact amount unchanged", () => {
    expect(roundAmount(9.99, "USD")).toBe(9.99)
  })
})
