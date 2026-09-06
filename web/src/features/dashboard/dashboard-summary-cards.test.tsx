import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import DashboardSummaryCards, { DashboardSummaryCardsSkeleton } from "./dashboard-summary-cards"
import type { DashboardSummary } from "@/types"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const translations: Record<string, string> = {
        "dashboard.stats.thisMonth": "This month",
        "dashboard.stats.activeMonthly": "Active monthly",
        "dashboard.stats.activeYearly": "Active yearly",
        "dashboard.stats.activeSubscriptions": "active subscriptions",
        "dashboard.stats.upcoming": "Upcoming",
      }
      return translations[key] ?? key
    },
    i18n: { language: "en" },
  }),
}))

vi.mock("react-router", () => ({
  Link: ({
    children,
    className,
    onFocus,
    onPointerEnter,
    to,
  }: React.AnchorHTMLAttributes<HTMLAnchorElement> & { to: string }) => (
    <a
      href={to}
      className={className}
      onPointerEnter={onPointerEnter}
      onFocus={onFocus}
      data-testid="upcoming-link"
    >
      {children}
    </a>
  ),
}))

vi.mock("@/lib/route-preload", () => ({
  preloadRouteForPath: vi.fn(),
}))

describe("DashboardSummaryCards", () => {
  const sampleSummary: DashboardSummary = {
    due_this_month: 128.5,
    total_monthly: 95.2,
    total_yearly: 1142.4,
    committed_monthly: 95.2,
    committed_yearly: 1142.4,
    active_count: 7,
    upcoming_renewal_count: 2,
    currency: "USD",
  }

  it("renders this month due amount, labels, and currency tag", () => {
    const markup = renderToStaticMarkup(
      <DashboardSummaryCards
        summary={sampleSummary}
        preferredCurrency="USD"
        currencySymbol="$"
        language="en"
      />
    )

    expect(markup).toContain("This month")
    expect(markup).toContain("$128.50")
    expect(markup).toContain("USD")
  })

  it("renders secondary metrics for active monthly and active yearly in order", () => {
    const markup = renderToStaticMarkup(
      <DashboardSummaryCards
        summary={sampleSummary}
        preferredCurrency="USD"
        currencySymbol="$"
        language="en"
      />
    )

    expect(markup).toContain("Active monthly")
    expect(markup).toContain("$95.20")
    expect(markup).toContain("Active yearly")
    expect(markup).toContain("$1,142.40")

    const monthlyIdx = markup.indexOf("Active monthly")
    const yearlyIdx = markup.indexOf("Active yearly")
    expect(monthlyIdx).toBeLessThan(yearlyIdx)
  })

  it("renders active subscriptions count and pill", () => {
    const markup = renderToStaticMarkup(
      <DashboardSummaryCards
        summary={sampleSummary}
        preferredCurrency="USD"
        currencySymbol="$"
        language="en"
      />
    )

    expect(markup).toContain("7")
    expect(markup).toContain("active subscriptions")
  })

  it("highlights upcoming renewals pill when count > 0", () => {
    const markup = renderToStaticMarkup(
      <DashboardSummaryCards
        summary={sampleSummary}
        preferredCurrency="USD"
        currencySymbol="$"
        language="en"
      />
    )

    expect(markup).toContain("Upcoming")
    expect(markup).toContain("2")
    expect(markup).toContain("border-amber-500/30")
    expect(markup).toContain("bg-amber-500/10")
  })

  it("uses neutral styling for upcoming renewals when count is 0", () => {
    const zeroUpcomingSummary: DashboardSummary = {
      ...sampleSummary,
      upcoming_renewal_count: 0,
    }

    const markup = renderToStaticMarkup(
      <DashboardSummaryCards
        summary={zeroUpcomingSummary}
        preferredCurrency="USD"
        currencySymbol="$"
        language="en"
      />
    )

    expect(markup).toContain("Upcoming")
    expect(markup).toContain("0")
    expect(markup).not.toContain("border-amber-500/30")
    expect(markup).toContain("border-border/70")
  })

  it("renders DashboardSummaryCardsSkeleton with matching responsive layout", () => {
    const markup = renderToStaticMarkup(<DashboardSummaryCardsSkeleton />)

    expect(markup).toContain("md:grid-cols-[minmax(0,1.3fr)_minmax(0,0.9fr)]")
    expect(markup).toContain("grid-cols-2 divide-x divide-border/60")
  })
})
