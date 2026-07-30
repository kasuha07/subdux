import { describe, expect, it } from "vitest"

import { BackendAPIError } from "@/lib/api"

import { buildLoadErrorCopy } from "./load-error"

describe("buildLoadErrorCopy", () => {
  it.each(["dashboard", "reports"])(
    "applies the shared backend error category to the %s namespace",
    (namespace) => {
      const error = new BackendAPIError("exchange rate is unavailable", {
        code: "exchange_rate_unavailable",
        status: 503,
      })

      expect(buildLoadErrorCopy(error, (key) => key, namespace)).toEqual({
        title: `${namespace}.error.exchangeRateTitle`,
        description: `${namespace}.error.exchangeRateDescription`,
        retry: `${namespace}.error.retry`,
      })
    }
  )

  it("falls back to generic copy for unclassified backend errors", () => {
    const error = new BackendAPIError("request failed", {
      code: "new_backend_error",
      status: 503,
    })

    expect(buildLoadErrorCopy(error, (key) => key, "dashboard")).toEqual({
      title: "dashboard.error.title",
      description: "dashboard.error.description",
      retry: "dashboard.error.retry",
    })
  })
})
