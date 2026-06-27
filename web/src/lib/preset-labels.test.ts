import type { TFunction } from "i18next"
import { describe, expect, it } from "vitest"

import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import type { Category, PaymentMethod } from "@/types"

// Minimal translate stub: resolves keys from a lookup map, otherwise returns the
// provided defaultValue (mirroring how i18next handles `{ defaultValue }`).
function makeT(translations: Record<string, string>): TFunction {
  const t = (key: string, options?: { defaultValue?: string }): string => {
    if (key in translations) {
      return translations[key]
    }
    return options?.defaultValue ?? key
  }
  return t as unknown as TFunction
}

function makeCategory(overrides: Partial<Category> = {}): Category {
  return {
    id: 1,
    name: "My Category",
    system_key: null,
    name_customized: false,
    display_order: 0,
    ...overrides,
  }
}

function makePaymentMethod(overrides: Partial<PaymentMethod> = {}): PaymentMethod {
  return {
    id: 1,
    name: "My Method",
    system_key: null,
    name_customized: false,
    icon: "",
    sort_order: 0,
    ...overrides,
  }
}

describe("getCategoryLabel", () => {
  it("returns the preset translation for a non-customized system category", () => {
    const t = makeT({ "presets.category.streaming": "Streaming" })
    const category = makeCategory({ system_key: "streaming", name: "raw" })
    expect(getCategoryLabel(category, t)).toBe("Streaming")
  })

  it("returns the stored name when the system category was customized", () => {
    const t = makeT({ "presets.category.streaming": "Streaming" })
    const category = makeCategory({
      system_key: "streaming",
      name: "My Streaming",
      name_customized: true,
    })
    expect(getCategoryLabel(category, t)).toBe("My Streaming")
  })

  it("falls back to the stored name when the preset translation is empty", () => {
    const t = makeT({}) // no translation -> defaultValue "" -> falls back
    const category = makeCategory({ system_key: "unknown_key", name: "Fallback Name" })
    expect(getCategoryLabel(category, t)).toBe("Fallback Name")
  })

  it("returns the name for a category without a system key", () => {
    const t = makeT({ "presets.category.streaming": "Streaming" })
    const category = makeCategory({ system_key: null, name: "Custom" })
    expect(getCategoryLabel(category, t)).toBe("Custom")
  })
})

describe("getPaymentMethodLabel", () => {
  it("returns the preset translation for a non-customized system method", () => {
    const t = makeT({ "presets.payment_method.credit_card": "Credit Card" })
    const method = makePaymentMethod({ system_key: "credit_card", name: "raw" })
    expect(getPaymentMethodLabel(method, t)).toBe("Credit Card")
  })

  it("returns the stored name when the system method was customized", () => {
    const t = makeT({ "presets.payment_method.credit_card": "Credit Card" })
    const method = makePaymentMethod({
      system_key: "credit_card",
      name: "My Card",
      name_customized: true,
    })
    expect(getPaymentMethodLabel(method, t)).toBe("My Card")
  })

  it("falls back to the stored name when the preset translation is empty", () => {
    const t = makeT({})
    const method = makePaymentMethod({ system_key: "unknown", name: "Fallback" })
    expect(getPaymentMethodLabel(method, t)).toBe("Fallback")
  })

  it("returns the name for a method without a system key", () => {
    const t = makeT({})
    const method = makePaymentMethod({ system_key: null, name: "Cash" })
    expect(getPaymentMethodLabel(method, t)).toBe("Cash")
  })
})
