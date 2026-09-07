import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import { ReportsSkeleton } from "./reports-panels"

describe("ReportsSkeleton", () => {
  it("renders with page loading animation, shimmer effects, and complete 6-section structure", () => {
    const markup = renderToStaticMarkup(<ReportsSkeleton />)

    // Root accessibility & loading classes
    expect(markup).toContain('role="status"')
    expect(markup).toContain('aria-label="Loading reports"')
    expect(markup).toContain("page-loading-enter")
    expect(markup).toContain("reports-skeleton")

    // Staggered KPI cards with enter delays
    expect(markup).toContain("subscription-card-enter")
    expect(markup).toContain("--card-delay")
    expect(markup).toContain("reports-skeleton-shimmer")

    // Forecast chart wave and grid SVG
    expect(markup).toContain("reports-skeleton-wave")
    expect(markup).toContain("forecast-skeleton-area-fill")
    expect(markup).toContain("stroke-border")

    // Progress bar skeletons & pulsing elements
    expect(markup).toContain("animate-pulse")
  })
})
