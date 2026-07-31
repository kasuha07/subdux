import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import { LoadErrorState } from "@/components/load-error-state"
import { BackendAPIError } from "@/lib/api"
import { buildLoadErrorCopy } from "@/lib/load-error"

import { DashboardLoadError } from "./dashboard-load-error"
import { requestDashboardBootstrap } from "./hooks/use-dashboard-data"

describe("DashboardLoadError", () => {
  it("keeps the production bootstrap error ahead of the empty state and retries the request", async () => {
    const error = new BackendAPIError("exchange rate is unavailable", {
      code: "exchange_rate_unavailable",
      status: 503,
    })
    const get = vi.fn()
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce({ subscriptions: [] }) as unknown as
        Parameters<typeof requestDashboardBootstrap>[0]

    await expect(requestDashboardBootstrap(get)).rejects.toBe(error)

    const copy = buildLoadErrorCopy(error, (key) => key, "dashboard")
    const retry = () => void requestDashboardBootstrap(get)
    const element = (
      <LoadErrorState
        error={error}
        fallback={<DashboardLoadError copy={copy} onRetry={retry} />}
      >
        <span>dashboard.empty.title</span>
      </LoadErrorState>
    )
    const markup = renderToStaticMarkup(element)

    expect(markup).toContain("dashboard.error.exchangeRateTitle")
    expect(markup).toContain("dashboard.error.exchangeRateDescription")
    expect(markup).toContain("py-16")
    expect(markup).toContain("rounded-full bg-muted p-4")
    expect(markup).not.toContain("dashboard.empty.title")

    DashboardLoadError({ copy, onRetry: retry }).props.onRetry()
    expect(get).toHaveBeenCalledTimes(2)
    expect(get).toHaveBeenLastCalledWith("/dashboard/bootstrap")
  })
})
