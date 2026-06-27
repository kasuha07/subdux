import { describe, expect, it } from "vitest"

import {
  DEFAULT_CURRENCY_FALLBACK,
  PRESET_CURRENCIES,
  formatCurrencyDisplay,
  getPresetCurrencies,
  getPresetCurrencyMeta,
} from "@/lib/currencies"

describe("formatCurrencyDisplay", () => {
  it("combines code, alias and symbol when both are present", () => {
    expect(formatCurrencyDisplay("USD", "US Dollar", "$")).toBe("USD - US Dollar ($)")
  })

  it("shows only the alias when no symbol is given", () => {
    expect(formatCurrencyDisplay("USD", "US Dollar")).toBe("USD - US Dollar")
  })

  it("shows only the symbol when no alias is given", () => {
    expect(formatCurrencyDisplay("USD", undefined, "$")).toBe("USD ($)")
  })

  it("falls back to the code alone when alias and symbol are blank", () => {
    expect(formatCurrencyDisplay("USD", "  ", "  ")).toBe("USD")
    expect(formatCurrencyDisplay("USD")).toBe("USD")
  })

  it("trims alias and symbol whitespace", () => {
    expect(formatCurrencyDisplay("EUR", "  Euro  ", "  €  ")).toBe("EUR - Euro (€)")
  })
})

describe("getPresetCurrencies", () => {
  it("returns the full preset list with code/symbol/alias on every entry", () => {
    const currencies = getPresetCurrencies("en")
    expect(currencies.length).toBeGreaterThan(0)
    expect(currencies.every((c) => c.code && c.symbol && c.alias)).toBe(true)
    expect(currencies.map((c) => c.code)).toContain("USD")
  })

  it("returns a stable cached reference for the same locale", () => {
    expect(getPresetCurrencies("en")).toBe(getPresetCurrencies("en"))
  })

  it("treats blank locale as the default 'en' locale", () => {
    expect(getPresetCurrencies("   ")).toBe(getPresetCurrencies("en"))
  })

  it("exposes the eagerly-built english preset list", () => {
    expect(PRESET_CURRENCIES).toBe(getPresetCurrencies("en"))
  })
})

describe("getPresetCurrencyMeta", () => {
  it("looks up metadata case-insensitively", () => {
    const lower = getPresetCurrencyMeta("usd")
    expect(lower?.code).toBe("USD")
    expect(getPresetCurrencyMeta("USD")).toBe(lower)
  })

  it("returns undefined for an unknown code", () => {
    expect(getPresetCurrencyMeta("ZZZ")).toBeUndefined()
  })
})

describe("DEFAULT_CURRENCY_FALLBACK", () => {
  it("contains the common base currencies", () => {
    expect(DEFAULT_CURRENCY_FALLBACK).toContain("USD")
    expect(DEFAULT_CURRENCY_FALLBACK).toContain("CNY")
  })
})
