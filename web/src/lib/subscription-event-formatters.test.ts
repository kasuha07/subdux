import type { TFunction } from "i18next"
import { describe, expect, it } from "vitest"

import {
  formatSubscriptionEventAmountChange,
  reportRenewalModeLabel,
  subscriptionDetailEventChangeRows,
  subscriptionEventFieldLabel,
  subscriptionEventTypeLabel,
  subscriptionLifecycleStatusLabel,
  subscriptionRenewalModeLabel,
} from "@/lib/subscription-event-formatters"
import type { SubscriptionDetailEvent } from "@/types"

// Echoing translate stub that honours an explicit defaultValue, like i18next.
function makeT(translations: Record<string, string> = {}): TFunction {
  const t = (key: string, options?: { defaultValue?: string }): string => {
    if (key in translations) {
      return translations[key]
    }
    if (options && "defaultValue" in options) {
      return options.defaultValue ?? key
    }
    return key
  }
  return t as unknown as TFunction
}

// Format helper used by the amount-change rows; deterministic and locale-free.
const formatAmount = (amount: number, currency?: string): string =>
  `${currency ?? "USD"} ${amount.toFixed(2)}`

describe("label helpers", () => {
  it("builds the event type translation key", () => {
    const t = makeT({ "reports.recentChanges.types.updated": "Updated" })
    expect(subscriptionEventTypeLabel("updated", t)).toBe("Updated")
  })

  it("builds the renewal mode translation key", () => {
    const t = makeT({ "reports.renewalModes.auto_renew": "Auto renew" })
    expect(reportRenewalModeLabel("auto_renew", t)).toBe("Auto renew")
  })

  it("returns the translated field label when present", () => {
    const t = makeT({ "reports.recentChanges.fields.amount": "Amount" })
    expect(subscriptionEventFieldLabel("amount", t)).toBe("Amount")
  })

  it("returns the raw field name when no translation exists", () => {
    // The echoing stub returns the key unchanged, which the helper detects.
    const t = makeT({})
    expect(subscriptionEventFieldLabel("amount", t)).toBe("amount")
  })

  it("returns an empty string for blank lifecycle/renewal values", () => {
    const t = makeT({})
    expect(subscriptionLifecycleStatusLabel("", t)).toBe("")
    expect(subscriptionRenewalModeLabel("", t)).toBe("")
  })

  it("translates non-empty lifecycle and renewal values, defaulting to the raw value", () => {
    const t = makeT({ "subscription.card.status.active": "Active" })
    expect(subscriptionLifecycleStatusLabel("active", t)).toBe("Active")
    // No translation -> defaultValue is the raw value.
    expect(subscriptionRenewalModeLabel("manual_renew", t)).toBe("manual_renew")
  })
})

describe("formatSubscriptionEventAmountChange", () => {
  it("renders a transition when both amounts are present", () => {
    const result = formatSubscriptionEventAmountChange(
      { previous_amount: 10, previous_currency: "USD", new_amount: 12, new_currency: "USD" },
      formatAmount
    )
    expect(result).toBe("USD 10.00 -> USD 12.00")
  })

  it("renders only the new amount when there is no previous amount", () => {
    const result = formatSubscriptionEventAmountChange(
      { previous_amount: null, new_amount: 12, new_currency: "EUR" },
      formatAmount
    )
    expect(result).toBe("EUR 12.00")
  })

  it("renders only the previous amount when there is no new amount", () => {
    const result = formatSubscriptionEventAmountChange(
      { previous_amount: 8, previous_currency: "GBP", new_amount: null },
      formatAmount
    )
    expect(result).toBe("GBP 8.00")
  })

  it("returns an empty string when both amounts are null", () => {
    const result = formatSubscriptionEventAmountChange(
      { previous_amount: null, new_amount: null },
      formatAmount
    )
    expect(result).toBe("")
  })
})

function makeEvent(overrides: Partial<SubscriptionDetailEvent> = {}): SubscriptionDetailEvent {
  return {
    id: 1,
    type: "updated",
    changed_fields: [],
    previous_amount: null,
    new_amount: null,
    previous_monthly_amount: null,
    new_monthly_amount: null,
    previous_currency: "",
    new_currency: "",
    previous_next_billing_date: null,
    new_next_billing_date: null,
    previous_status: "",
    new_status: "",
    previous_renewal_mode: "",
    new_renewal_mode: "",
    previous_category_name: "",
    new_category_name: "",
    previous_payment_method_name: "",
    new_payment_method_name: "",
    changed_at: "2026-06-01",
    ...overrides,
  }
}

describe("subscriptionDetailEventChangeRows", () => {
  it("emits an amount transition row for an updated amount", () => {
    const event = makeEvent({
      type: "updated",
      changed_fields: ["amount"],
      previous_amount: 10,
      new_amount: 15,
      previous_currency: "USD",
      new_currency: "USD",
    })
    const t = makeT({ "reports.recentChanges.fields.amount": "Amount" })
    const rows = subscriptionDetailEventChangeRows(event, formatAmount, "en-US", t)
    expect(rows).toEqual([{ label: "Amount", previous: "USD 10.00", next: "USD 15.00" }])
  })

  it("uses the empty placeholder on the previous side for created events", () => {
    const event = makeEvent({
      type: "created",
      new_amount: 20,
      new_currency: "USD",
      new_status: "active",
    })
    const t = makeT({
      "subscription.detail.empty.none": "—",
      "subscription.card.status.active": "Active",
    })
    const rows = subscriptionDetailEventChangeRows(event, formatAmount, "en-US", t)
    const amountRow = rows.find((row) => row.next === "USD 20.00")
    expect(amountRow?.previous).toBe("—")
    const statusRow = rows.find((row) => row.next === "Active")
    expect(statusRow?.previous).toBe("—")
  })

  it("uses the empty placeholder on the next side for deleted events", () => {
    const event = makeEvent({
      type: "deleted",
      previous_amount: 30,
      previous_currency: "USD",
    })
    const t = makeT({ "subscription.detail.empty.none": "—" })
    const rows = subscriptionDetailEventChangeRows(event, formatAmount, "en-US", t)
    const amountRow = rows.find((row) => row.previous === "USD 30.00")
    expect(amountRow?.next).toBe("—")
  })

  it("renders a formatted next_billing_date transition", () => {
    const event = makeEvent({
      type: "updated",
      changed_fields: ["next_billing_date"],
      previous_next_billing_date: "2026-06-01",
      new_next_billing_date: "2026-07-01",
    })
    const t = makeT({ "subscription.detail.empty.none": "—" })
    const rows = subscriptionDetailEventChangeRows(event, formatAmount, "en-US", t)
    expect(rows).toEqual([
      { label: "next_billing_date", previous: "Jun 1, 2026", next: "Jul 1, 2026" },
    ])
  })

  it("defaults to the amount field when changed_fields is empty", () => {
    const event = makeEvent({
      type: "updated",
      changed_fields: [],
      previous_amount: 5,
      new_amount: 9,
      previous_currency: "USD",
      new_currency: "USD",
    })
    const t = makeT({})
    const rows = subscriptionDetailEventChangeRows(event, formatAmount, "en-US", t)
    expect(rows).toEqual([{ label: "amount", previous: "USD 5.00", next: "USD 9.00" }])
  })
})
