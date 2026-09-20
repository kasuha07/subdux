// @vitest-environment happy-dom
import { act, StrictMode } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import SubscriptionSquareCard from "./subscription-square-card"
import type { Subscription } from "@/types"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => {
      const translations: Record<string, string> = {
        "subscription.detail.open": "Open detail",
        "subscription.detail.openCard": `Open subscription details for ${options?.name ?? ""}`,
        "subscription.card.status.active": "Active",
        "subscription.card.renewalMode.auto": "Auto-renew",
        "subscription.cycle.monthly": "month",
      }
      return translations[key] ?? key
    },
    i18n: { language: "en" },
  }),
}))

describe("SubscriptionSquareCard", () => {
  const sampleSubscription: Subscription = {
    id: 1,
    name: "Netflix",
    amount: 15.99,
    currency: "USD",
    billing_cycle: "monthly",
    billing_cycle_count: 1,
    billing_type: "recurring",
    renewal_mode: "auto",
    status: "active",
    start_date: "2024-01-01",
    next_billing_date: "2026-10-01",
    notes: "",
    user_id: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  }

  const defaultProps = {
    subscription: sampleSubscription,
    displayAmount: 15.99,
    displayCurrency: "USD",
    onOpenDetail: vi.fn(),
  }

  let container: HTMLDivElement
  let root: Root

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

  it("opens detail drawer when square card body is clicked in normal mode", async () => {
    const onOpenDetail = vi.fn()
    await act(async () => {
      root.render(
        <StrictMode>
          <SubscriptionSquareCard {...defaultProps} onOpenDetail={onOpenDetail} />
        </StrictMode>
      )
    })

    const card = container.firstElementChild as HTMLElement
    await act(async () => {
      card.click()
    })

    expect(onOpenDetail).toHaveBeenCalledWith(sampleSubscription)
  })

  it("toggles selection when square card body is clicked in batch mode", async () => {
    const onToggleSelect = vi.fn()
    const onOpenDetail = vi.fn()
    await act(async () => {
      root.render(
        <StrictMode>
          <SubscriptionSquareCard
            {...defaultProps}
            onToggleSelect={onToggleSelect}
            onOpenDetail={onOpenDetail}
          />
        </StrictMode>
      )
    })

    const heading = container.querySelector("h3") as HTMLElement
    await act(async () => {
      heading.click()
    })

    expect(onToggleSelect).toHaveBeenCalledTimes(1)
    expect(onToggleSelect).toHaveBeenCalledWith(1)
    expect(onOpenDetail).not.toHaveBeenCalled()
  })

  it("toggles selection once when checkbox is clicked in batch mode", async () => {
    const onToggleSelect = vi.fn()
    await act(async () => {
      root.render(
        <StrictMode>
          <SubscriptionSquareCard
            {...defaultProps}
            onToggleSelect={onToggleSelect}
          />
        </StrictMode>
      )
    })

    const checkbox = container.querySelector("[data-slot='checkbox']") as HTMLElement
    await act(async () => {
      checkbox.click()
    })

    expect(onToggleSelect).toHaveBeenCalledTimes(1)
    expect(onToggleSelect).toHaveBeenCalledWith(1)
  })

  it("opens detail drawer and does not toggle selection when open detail button is clicked in batch mode", async () => {
    const onToggleSelect = vi.fn()
    const onOpenDetail = vi.fn()
    await act(async () => {
      root.render(
        <StrictMode>
          <SubscriptionSquareCard
            {...defaultProps}
            onToggleSelect={onToggleSelect}
            onOpenDetail={onOpenDetail}
          />
        </StrictMode>
      )
    })

    const openButton = container.querySelector("button[aria-label='Open detail']") as HTMLElement
    await act(async () => {
      openButton.click()
    })

    expect(onOpenDetail).toHaveBeenCalledWith(sampleSubscription)
    expect(onToggleSelect).not.toHaveBeenCalled()
  })
})
