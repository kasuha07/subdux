import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import { LoadErrorState } from "@/components/load-error-state"
import { BackendAPIError } from "@/lib/api"
import { buildLoadErrorCopy } from "@/lib/load-error"

import { requestReportsData } from "./reports-data"
import { ReportsLoadError } from "./reports-load-error"

describe("ReportsLoadError", () => {
  it("keeps the production report error ahead of the empty state and retries both requests", async () => {
    const error = new BackendAPIError("exchange rate is unavailable", {
      code: "exchange_rate_unavailable",
      status: 503,
    })
    const get = vi.fn()
      .mockRejectedValueOnce(error)
      .mockResolvedValueOnce([])
      .mockResolvedValueOnce({})
      .mockResolvedValueOnce([]) as unknown as
        Parameters<typeof requestReportsData>[0]

    await expect(requestReportsData(get)).rejects.toBe(error)

    const copy = buildLoadErrorCopy(error, (key) => key, "reports")
    const retry = () => void requestReportsData(get)
    const element = (
      <LoadErrorState
        error={error}
        fallback={<ReportsLoadError copy={copy} onRetry={retry} />}
      >
        <span>reports.empty.title</span>
      </LoadErrorState>
    )
    const markup = renderToStaticMarkup(element)

    expect(markup).toContain("reports.error.exchangeRateTitle")
    expect(markup).toContain("reports.error.exchangeRateDescription")
    expect(markup).toContain("border-dashed p-8 text-center")
    expect(markup).not.toContain("rounded-full bg-muted p-4")
    expect(markup).not.toContain("reports.empty.title")

    ReportsLoadError({ copy, onRetry: retry }).props.onRetry()
    expect(get).toHaveBeenCalledTimes(4)
    expect(get.mock.calls.map(([path]) => path)).toEqual([
      "/reports/analytics",
      "/currencies",
      "/reports/analytics",
      "/currencies",
    ])
  })
})
