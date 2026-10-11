import { describe, expect, it } from "vitest"

import type { Subscription } from "@/types"

import { computeIntroPlan, type SubscriptionFormValues } from "./hooks/use-subscription-form-state"
import {
  chargeAmountOn,
  chargeDatesFrom,
  getIntroKind,
  regularAmount,
  remainingIntroCharges,
} from "./subscription-pending-price"

function subscription(overrides: Partial<Subscription>): Subscription {
  return {
    revision: 1,
    id: 1,
    name: "Example",
    amount: 1,
    currency: "USD",
    pending_amount: 30,
    pending_from: "2026-06-01",
    status: "active",
    renewal_mode: "auto_renew",
    ends_at: null,
    billing_type: "recurring",
    recurrence_type: "interval",
    interval_count: 1,
    interval_unit: "month",
    monthly_day: null,
    yearly_month: null,
    yearly_day: null,
    next_billing_date: "2026-04-01",
    category: "",
    category_id: null,
    payment_method_id: null,
    notify_enabled: null,
    notify_days_before: null,
    icon: "",
    url: "",
    notes: "",
    created_at: "",
    updated_at: "",
    ...overrides,
  }
}

function formValues(overrides: Partial<SubscriptionFormValues>): SubscriptionFormValues {
  return {
    amount: "30",
    introMode: "none",
    introAmount: "",
    introCycles: "3",
    trialDays: "7",
    trialStart: "2026-03-01",
    nextBillingDate: "2026-01-31",
    endsAt: "2026-01-31",
    categoryId: "",
    currency: "USD",
    icon: "",
    intervalCount: "1",
    intervalUnit: "month",
    monthlyDay: "15",
    name: "Example",
    notes: "",
    notifyDaysBefore: "",
    notifyEnabled: "default",
    paymentMethodId: "",
    recurrenceType: "interval",
    renewalMode: "auto_renew",
    status: "active",
    url: "",
    yearlyDay: "1",
    yearlyMonth: "1",
    ...overrides,
  }
}

describe("subscription pending price", () => {
  it("bills charges on or after pending_from at the scheduled price", () => {
    const sub = subscription({})
    expect(chargeAmountOn(sub, "2026-05-01")).toBe(1)
    expect(chargeAmountOn(sub, "2026-06-01")).toBe(30)
    expect(regularAmount(sub)).toBe(30)
    expect(regularAmount(subscription({ pending_amount: null, pending_from: null }))).toBe(1)
  })

  it("classifies a zero current price as a trial", () => {
    expect(getIntroKind(subscription({ amount: 0 }))).toBe("trial")
    expect(getIntroKind(subscription({}))).toBe("intro")
    expect(getIntroKind(subscription({ pending_amount: null, pending_from: null }))).toBeNull()
  })

  it("counts the discounted charges left from the next billing date", () => {
    expect(remainingIntroCharges(subscription({}))).toBe(2)
    expect(remainingIntroCharges(subscription({ next_billing_date: "2026-06-01" }))).toBe(0)
  })

  it("clamps month-end interval charges like the backend", () => {
    const schedule = {
      recurrenceType: "interval",
      intervalCount: 1,
      intervalUnit: "month",
      monthlyDay: null,
      yearlyMonth: null,
      yearlyDay: null,
    }
    expect(chargeDatesFrom("2026-01-31", schedule, 3)).toEqual(["2026-01-31", "2026-02-28", "2026-03-28"])
  })

  it("follows monthly-date schedules", () => {
    const schedule = {
      recurrenceType: "monthly_date",
      intervalCount: null,
      intervalUnit: "",
      monthlyDay: 31,
      yearlyMonth: null,
      yearlyDay: null,
    }
    expect(chargeDatesFrom("2026-01-31", schedule, 3)).toEqual(["2026-01-31", "2026-02-28", "2026-03-31"])
  })
})

describe("computeIntroPlan", () => {
  it("derives the first charge from a free trial", () => {
    expect(computeIntroPlan(formValues({ introMode: "trial", trialDays: "7", trialStart: "2026-03-01" }))).toEqual({
      regularFrom: "2026-03-08",
      introDates: [],
      nextBillingDate: "2026-03-08",
    })
  })

  it("derives the regular-price date after N discounted charges", () => {
    expect(computeIntroPlan(formValues({ introMode: "intro", introAmount: "1", introCycles: "2" }))).toEqual({
      regularFrom: "2026-03-28",
      introDates: ["2026-01-31", "2026-02-28"],
      nextBillingDate: "2026-01-31",
    })
  })

  it("rejects incomplete offers", () => {
    expect(computeIntroPlan(formValues({ introMode: "trial", trialDays: "0" }))).toBeNull()
    expect(computeIntroPlan(formValues({ introMode: "intro", introAmount: "" }))).toBeNull()
    expect(computeIntroPlan(formValues({ introMode: "none" }))).toBeNull()
  })
})
