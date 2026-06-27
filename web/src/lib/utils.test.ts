import { afterEach, describe, expect, it, vi } from "vitest"

import {
  cn,
  daysUntil,
  formatCurrency,
  formatCurrencyWithSymbol,
  formatDate,
  formatDateKey,
} from "@/lib/utils"

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1")
  })

  it("dedupes conflicting tailwind utilities, last wins", () => {
    expect(cn("px-2", "px-4")).toBe("px-4")
  })

  it("drops falsy values and supports conditional objects", () => {
    expect(cn("a", false, null, undefined, { b: true, c: false })).toBe("a b")
  })
})

describe("formatCurrency", () => {
  it("formats USD with the default locale", () => {
    expect(formatCurrency(9.99)).toBe("$9.99")
  })

  it("always keeps two minimum fraction digits", () => {
    expect(formatCurrency(10, "USD")).toBe("$10.00")
  })

  it("honours an explicit currency and locale", () => {
    // Non-breaking space separates symbol and amount in many locales.
    expect(formatCurrency(1234.5, "EUR", "de-DE")).toMatch(/1\.234,50/)
    expect(formatCurrency(1234.5, "EUR", "de-DE")).toContain("€")
  })
})

describe("formatCurrencyWithSymbol", () => {
  it("falls back to formatCurrency when no symbol is provided", () => {
    expect(formatCurrencyWithSymbol(5, "USD")).toBe(formatCurrency(5, "USD"))
  })

  it("falls back to formatCurrency when the symbol is blank", () => {
    expect(formatCurrencyWithSymbol(5, "USD", "   ")).toBe(formatCurrency(5, "USD"))
  })

  it("replaces the currency token with the custom symbol", () => {
    expect(formatCurrencyWithSymbol(9.99, "USD", "US$")).toBe("US$9.99")
  })

  it("trims the provided symbol before substitution", () => {
    expect(formatCurrencyWithSymbol(9.99, "USD", "  ¥ ")).toBe("¥9.99")
  })
})

describe("formatDateKey", () => {
  it("zero-pads month and day from local date parts", () => {
    expect(formatDateKey(new Date(2026, 0, 5))).toBe("2026-01-05")
  })

  it("formats double-digit month and day", () => {
    expect(formatDateKey(new Date(2026, 10, 23))).toBe("2026-11-23")
  })
})

describe("formatDate", () => {
  it("formats an ISO date-only string in en-US", () => {
    expect(formatDate("2026-06-20")).toBe("Jun 20, 2026")
  })

  it("returns the raw input for an unparseable string", () => {
    expect(formatDate("not-a-date")).toBe("not-a-date")
  })

  it("returns the raw input for an invalid calendar date", () => {
    expect(formatDate("2026-02-30")).toBe("2026-02-30")
  })
})

describe("daysUntil", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("returns NaN for an unparseable date", () => {
    expect(daysUntil("nope")).toBeNaN()
  })

  it("counts whole days to a future date", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 15, 9, 30))
    expect(daysUntil("2026-06-20")).toBe(5)
  })

  it("returns zero for today regardless of time of day", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 15, 23, 59))
    expect(daysUntil("2026-06-15")).toBe(0)
  })

  it("returns a negative count for a past date", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 5, 15))
    expect(daysUntil("2026-06-10")).toBe(-5)
  })
})
