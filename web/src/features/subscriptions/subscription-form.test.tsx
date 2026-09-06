import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"
import type { ReactNode } from "react"

import SubscriptionForm from "./subscription-form"
import type { Subscription } from "@/types"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => key,
      i18n: { language: "en" },
    }),
  }
})

vi.mock("radix-ui", async (importOriginal) => {
  const actual = await importOriginal<typeof import("radix-ui")>()
  return {
    ...actual,
    Dialog: {
      ...actual.Dialog,
      Portal: ({ children }: { children: ReactNode }) => <div data-slot="dialog-portal">{children}</div>,
    },
  }
})

describe("SubscriptionForm", () => {
  const sampleSubscription: Subscription = {
    id: 1,
    name: "Spotify",
    amount: 9.99,
    currency: "USD",
    billing_cycle: "monthly",
    billing_cycle_count: 1,
    billing_type: "recurring",
    renewal_mode: "auto_renew",
    status: "active",
    start_date: "2024-01-01",
    next_billing_date: "2026-10-01",
    notes: "",
    user_id: 1,
    created_at: "2024-01-01T00:00:00Z",
    updated_at: "2024-01-01T00:00:00Z",
  }

  it("applies modal animation classes in add mode", () => {
    const markup = renderToStaticMarkup(
      <SubscriptionForm
        open={true}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        userCurrencies={[]}
        categories={[]}
        paymentMethods={[]}
      />
    )

    expect(markup).toContain("subscription-form-dialog")
    expect(markup).toContain("subscription-edit-modal")
  })

  it("applies modal animation classes in edit mode", () => {
    const markup = renderToStaticMarkup(
      <SubscriptionForm
        open={true}
        subscription={sampleSubscription}
        onOpenChange={vi.fn()}
        onSubmit={vi.fn()}
        userCurrencies={[]}
        categories={[]}
        paymentMethods={[]}
      />
    )

    expect(markup).toContain("subscription-form-dialog")
    expect(markup).toContain("subscription-edit-modal")
  })
})
