import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import SubscriptionBatchBar from "./subscription-batch-bar"
import { createActivateBatchInput } from "./subscription-batch-inputs"

vi.mock("@/lib/api", () => ({
  api: { post: vi.fn() },
  localizeBackendError: (code: string) => code,
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}))

describe("SubscriptionBatchBar", () => {
  it("activates subscriptions without changing their renewal policy", () => {
    expect(createActivateBatchInput([1, 3])).toEqual({
      action: "update",
      ids: [1, 3],
      status: "active",
    })
  })

  it("renders the selection count and action entry points", () => {
    const markup = renderToStaticMarkup(
      <SubscriptionBatchBar
        categories={[{ id: 1, name: "Video", system_key: null, name_customized: false, display_order: 0 }]}
        getSubscriptionName={(id) => (id === 1 ? "Netflix" : "")}
        onBatchApplied={() => void 0}
        onClearSelection={() => void 0}
        paymentMethodLabelMap={new Map([[2, "Card"]])}
        paymentMethods={[{ id: 2, name: "Card", system_key: null, name_customized: false, icon: "", sort_order: 0 }]}
        selectedCount={2}
        selectedIDs={[1, 3]}
      />
    )

    expect(markup).toContain("subscription.batch.selected")
    expect(markup).toContain("subscription.batch.actions")
    expect(markup).toContain("subscription.batch.clear")
  })
})
