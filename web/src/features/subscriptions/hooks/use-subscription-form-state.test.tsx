// @vitest-environment happy-dom
import { act, useEffect, type FormEvent } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import type { Subscription } from "@/types"
import { computeIntroPlan, useSubscriptionFormState } from "./use-subscription-form-state"

vi.mock("./use-jev-category", () => ({ useJevCategory: () => () => {} }))

const original: Subscription = {
  id: 1, revision: 1, name: "Intro", amount: 1, currency: "USD",
  pending_amount: 30, pending_from: "2099-11-15", status: "active",
  renewal_mode: "auto_renew", ends_at: null, billing_type: "recurring",
  recurrence_type: "interval", interval_count: 1, interval_unit: "month",
  monthly_day: null, yearly_month: null, yearly_day: null,
  next_billing_date: "2099-10-01", category: "", category_id: null,
  payment_method_id: null, notify_enabled: null, notify_days_before: null,
  icon: "", url: "", notes: "", created_at: "", updated_at: "",
}

describe("editing subscriptions with scheduled prices", () => {
  let container: HTMLDivElement
  let root: Root
  let state: ReturnType<typeof useSubscriptionFormState>
  const onSubmit = vi.fn().mockResolvedValue(original)

  function Harness({ subscription, open }: { subscription: Subscription; open: boolean }) {
    const formState = useSubscriptionFormState({
      subscription, open, language: "en", onSubmit, onOpenChange: () => {},
      paymentMethods: [], userCurrencies: [], t: (key) => key,
    })
    useEffect(() => { state = formState }, [formState])
    return null
  }

  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  async function render(subscription = original, open = true) {
    await act(async () => root.render(<Harness subscription={subscription} open={open} />))
  }

  async function submit() {
    const event: Pick<FormEvent, "preventDefault"> = { preventDefault: () => {} }
    await act(async () => state.handleSubmit(event as FormEvent))
    expect(onSubmit).toHaveBeenCalledOnce()
    return onSubmit.mock.calls[0][0]
  }

  it.each(["2099-10-01", "2099-12-01"])("preserves the exact effective date when only notes change (%s)", async (nextBillingDate) => {
    await render({ ...original, next_billing_date: nextBillingDate })
    await act(async () => state.setField("notes", "Edited notes"))
    expect(computeIntroPlan(state.values)?.regularFrom).toBe("2099-11-15")
    const payload = await submit()
    expect(payload).toMatchObject({ amount: 1, next_billing_date: nextBillingDate, notes: "Edited notes" })
    expect(payload).not.toHaveProperty("pending_amount")
    expect(payload).not.toHaveProperty("pending_from")
  })

  it.each([0, 1])("preserves an ended offer with an overdue effective date (amount %s)", async (amount) => {
    await render({ ...original, amount, status: "ended", pending_from: "2020-11-15", ends_at: "2020-11-01" })
    await act(async () => state.setField("name", "Renamed"))
    const payload = await submit()
    expect(payload).toMatchObject({ amount, status: "ended", name: "Renamed" })
    expect(payload).not.toHaveProperty("pending_amount")
    expect(payload).not.toHaveProperty("pending_from")
  })

  it("keeps the trial price when changing status to ended", async () => {
    await render({ ...original, amount: 0 })
    await act(async () => state.setField("status", "ended"))
    const payload = await submit()
    expect(payload).toMatchObject({ amount: 0, status: "ended", next_billing_date: "2099-10-01" })
    expect(payload).not.toHaveProperty("pending_from")
  })

  it("clears the offer only when None is explicitly selected", async () => {
    await render()
    await act(async () => state.setField("introMode", "none"))
    expect(await submit()).toMatchObject({ amount: 30, pending_amount: null, pending_from: null })
  })

  it("recalculates dates when the user changes discounted charges", async () => {
    await render()
    await act(async () => state.setField("introCycles", "3"))
    expect(await submit()).toMatchObject({ amount: 1, pending_amount: 30, pending_from: "2100-01-01" })
  })

  it("changes the regular price without moving its effective date", async () => {
    await render()
    await act(async () => state.setField("amount", "40"))
    expect(await submit()).toMatchObject({ amount: 1, pending_amount: 40, pending_from: "2099-11-15" })
  })

  it("restores the original offer on reopening after an unsaved edit", async () => {
    await render()
    await act(async () => state.setField("introMode", "none"))
    await render(original, false)
    await render()
    expect(computeIntroPlan(state.values)?.regularFrom).toBe("2099-11-15")
    const payload = await submit()
    expect(payload.amount).toBe(1)
    expect(payload).not.toHaveProperty("pending_from")
  })
})
