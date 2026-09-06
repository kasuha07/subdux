import { renderToStaticMarkup } from "react-dom/server"
import { beforeEach, describe, expect, it, vi } from "vitest"

import SubscriptionCard from "./subscription-card"
import type { Subscription } from "@/types"
import { useHoverCapablePointer } from "./hooks/use-hover-capable-pointer"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => {
      const translations: Record<string, string> = {
        "common.edit": "Edit",
        "common.delete": "Delete",
        "common.moreActions": "More actions",
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

vi.mock("./hooks/use-hover-capable-pointer", () => ({
  useHoverCapablePointer: vi.fn(),
}))

describe("SubscriptionCard", () => {
  const sampleSubscription: Subscription = {
    id: 1,
    name: "Spotify",
    amount: 9.99,
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
    displayAmount: 9.99,
    displayCurrency: "USD",
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onOpenDetail: vi.fn(),
  }

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("applies swap animation classes to price summary and action toolbar when hover-capable", () => {
    vi.mocked(useHoverCapablePointer).mockReturnValue(true)

    const markup = renderToStaticMarkup(<SubscriptionCard {...defaultProps} />)

    // Price and badges summary block should have swap-out classes
    expect(markup).toContain("group-hover:opacity-0")
    expect(markup).toContain("group-hover:translate-x-2")
    expect(markup).toContain("group-hover:pointer-events-none")

    // Action buttons toolbar should have swap-in classes and data attribute
    expect(markup).toContain("data-card-actions")
    expect(markup).toContain("group-hover:opacity-100")
    expect(markup).toContain("group-hover:translate-x-0")
    expect(markup).toContain("opacity-0")
    expect(markup).toContain("translate-x-2")
  })

  it("does not apply hover swap-out classes to price summary on touch devices without hover pointer", () => {
    vi.mocked(useHoverCapablePointer).mockReturnValue(false)

    const markup = renderToStaticMarkup(<SubscriptionCard {...defaultProps} />)

    // Price and badges summary block should stay visible without hover hide classes
    expect(markup).not.toContain("group-hover:opacity-0")
    expect(markup).not.toContain("data-card-actions")
  })
})
