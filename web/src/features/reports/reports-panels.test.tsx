import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"
import { TrendingUp, RefreshCw, ReceiptText } from "lucide-react"

import {
  AnnualGrowthPanel,
  BreakdownPanel,
  KpiCard,
  PriceIncreasesPanel,
  TopSubscriptionsPanel,
  UpcomingRenewalsPanel,
} from "./reports-panels"
import type {
  ReportAnnualGrowthItem,
  ReportBreakdownItem,
  ReportPriceIncrease,
  ReportSubscriptionSpend,
  ReportUpcomingRenewal,
} from "@/types"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => {
      if (options?.amount) return `${key}: ${options.amount}`
      if (options?.count !== undefined) return `${key}: ${options.count}`
      return key
    },
    i18n: { language: "en" },
  }),
}))

describe("ReportsPanels with Animated Numbers", () => {
  const formatAmount = (val: number) => `$${val.toFixed(2)}`

  describe("KpiCard", () => {
    it("renders with static value prop for backward compatibility", () => {
      const markup = renderToStaticMarkup(
        <KpiCard
          icon={TrendingUp}
          label="Total Monthly"
          value="$120.00"
          detail="Yearly: $1,440.00"
          testId="kpi-monthly"
        />
      )

      expect(markup).toContain("Total Monthly")
      expect(markup).toContain("$120.00")
      expect(markup).toContain("Yearly: $1,440.00")
      expect(markup).toContain('data-testid="kpi-monthly"')
      expect(markup).toContain("subscription-card-enter")
    })

    it("renders with numericValue and formatValue in SSR/static markup", () => {
      const markup = renderToStaticMarkup(
        <KpiCard
          icon={TrendingUp}
          label="Total Monthly"
          numericValue={128.5}
          formatValue={formatAmount}
          detail="Yearly detail"
          testId="kpi-monthly"
          cardDelayMs={70}
        />
      )

      expect(markup).toContain("Total Monthly")
      expect(markup).toContain("$128.50")
      expect(markup).toContain('title="$128.50"')
      expect(markup).toContain('data-testid="kpi-monthly"')
      expect(markup).toContain("--card-delay:70ms")
    })

    it("renders integer count formatted when formatValue is omitted", () => {
      const markup = renderToStaticMarkup(
        <KpiCard
          icon={RefreshCw}
          label="Active Subscriptions"
          numericValue={12}
          detail="12 active"
          testId="kpi-active"
        />
      )

      expect(markup).toContain("Active Subscriptions")
      expect(markup).toContain("12")
      expect(markup).toContain('data-testid="kpi-active"')
    })
  })

  describe("BreakdownPanel", () => {
    const sampleItems: ReportBreakdownItem[] = [
      {
        key: "streaming",
        label: "Streaming",
        count: 3,
        monthly_amount: 45.99,
        percentage: 60.5,
      },
      {
        key: "software",
        label: "Software",
        count: 1,
        monthly_amount: 30.0,
        percentage: 39.5,
      },
    ]

    it("renders breakdown rows with formatted amount, count, and percentage", () => {
      const markup = renderToStaticMarkup(
        <BreakdownPanel
          title="Categories"
          icon={ReceiptText}
          items={sampleItems}
          formatAmount={formatAmount}
          emptyTitle="No items"
          emptyDescription="No categories found"
          labelForKey={(item) => item.label || item.key}
        />
      )

      expect(markup).toContain("Categories")
      expect(markup).toContain("Streaming")
      expect(markup).toContain("$45.99")
      expect(markup).toContain("60.5%")
      expect(markup).toContain("Software")
      expect(markup).toContain("$30.00")
      expect(markup).toContain("39.5%")
      expect(markup).toContain("cycle-progress-fill")
    })
  })

  describe("TopSubscriptionsPanel", () => {
    const sampleTop: ReportSubscriptionSpend[] = [
      {
        id: 1,
        name: "Netflix",
        monthly_amount: 15.99,
        renewal_mode: "auto",
        category: "Entertainment",
        next_billing_date: "2026-10-01",
        icon: null,
      },
    ]

    it("renders top subscriptions with animated monthly amount", () => {
      const markup = renderToStaticMarkup(
        <TopSubscriptionsPanel
          items={sampleTop}
          formatAmount={formatAmount}
        />
      )

      expect(markup).toContain("Netflix")
      expect(markup).toContain("$15.99")
      expect(markup).toContain("Entertainment")
    })
  })

  describe("UpcomingRenewalsPanel", () => {
    const sampleUpcoming: ReportUpcomingRenewal[] = [
      {
        id: 2,
        name: "Spotify",
        amount: 9.99,
        billing_date: "2026-09-25",
        days_until: 5,
        renewal_mode: "auto",
        currency: "USD",
        icon: null,
      },
    ]

    it("renders upcoming renewals with formatted renewal amount", () => {
      const markup = renderToStaticMarkup(
        <UpcomingRenewalsPanel
          items={sampleUpcoming}
          formatAmount={formatAmount}
          language="en"
        />
      )

      expect(markup).toContain("Spotify")
      expect(markup).toContain("$9.99")
    })
  })

  describe("PriceIncreasesPanel", () => {
    const sampleIncreases: ReportPriceIncrease[] = [
      {
        subscription_id: 3,
        name: "iCloud",
        previous_monthly_amount: 2.99,
        new_monthly_amount: 4.99,
        delta_monthly_amount: 2.0,
        delta_percentage: 66.9,
        changed_at: "2026-08-01",
      },
    ]

    it("renders price increases with amounts and percentage changes", () => {
      const markup = renderToStaticMarkup(
        <PriceIncreasesPanel
          items={sampleIncreases}
          formatAmount={formatAmount}
          language="en"
        />
      )

      expect(markup).toContain("iCloud")
      expect(markup).toContain("+66.9%")
      expect(markup).toContain("$2.99")
      expect(markup).toContain("$4.99")
    })
  })

  describe("AnnualGrowthPanel", () => {
    const sampleGrowth: ReportAnnualGrowthItem[] = [
      {
        subscription_id: 4,
        name: "GitHub Copilot",
        baseline_monthly_amount: 10.0,
        current_monthly_amount: 19.0,
        delta_monthly_amount: 9.0,
        delta_percentage: 90.0,
      },
    ]

    it("renders annual growth items with delta amounts and percentages", () => {
      const markup = renderToStaticMarkup(
        <AnnualGrowthPanel
          items={sampleGrowth}
          formatAmount={formatAmount}
        />
      )

      expect(markup).toContain("GitHub Copilot")
      expect(markup).toContain("$9.00")
      expect(markup).toContain("+90.0%")
      expect(markup).toContain("cycle-progress-fill")
    })
  })
})
