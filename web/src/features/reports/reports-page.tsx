import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  ArrowLeft,
  CalendarDays,
  CreditCard,
  Layers3,
  PieChart,
  ReceiptText,
  RefreshCw,
  Settings,
  TrendingUp,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import { reportRenewalModeLabel } from "@/lib/subscription-event-formatters"
import { formatCurrencyWithSymbol } from "@/lib/utils"
import type {
  AnalyticsReport,
  SubscriptionRenewalMode,
  UserCurrency,
} from "@/types"

import {
  AnnualGrowthPanel,
  BreakdownPanel,
  EmptyState,
  KpiCard,
  MonthlyForecastPanel,
  PriceIncreasesPanel,
  RecentChangesPanel,
  ReportsSkeleton,
  TopSubscriptionsPanel,
  UpcomingRenewalsPanel,
} from "./reports-panels"

export default function ReportsPage() {
  const { t, i18n } = useTranslation()
  const [report, setReport] = useState<AnalyticsReport | null>(null)
  const [currencies, setCurrencies] = useState<UserCurrency[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([
      api.get<AnalyticsReport>("/reports/analytics"),
      api.get<UserCurrency[]>("/currencies"),
    ])
      .then(([reportData, currencyData]) => {
        setReport(reportData)
        setCurrencies(currencyData || [])
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  const currencySymbolMap = useMemo(() => {
    return new Map(currencies.map((currency) => [currency.code.toUpperCase(), currency.symbol]))
  }, [currencies])

  const displayCurrency = report?.currency || "USD"
  const currencySymbol = currencySymbolMap.get(displayCurrency.toUpperCase())
  const formatAmount = (amount: number) =>
    formatCurrencyWithSymbol(amount, displayCurrency, currencySymbol, i18n.language)

  const monthlyForecast = report?.monthly_forecast ?? []
  const categoryBreakdown = report?.category_breakdown ?? []
  const paymentMethodBreakdown = report?.payment_method_breakdown ?? []
  const renewalModeBreakdown = report?.renewal_mode_breakdown ?? []
  const topSubscriptions = report?.top_subscriptions ?? []
  const upcomingRenewals = report?.upcoming_renewals ?? []
  const priceIncreases = report?.price_increases ?? []
  const annualGrowth = report?.annual_growth ?? []
  const recentChanges = report?.recent_changes ?? []
  const maxForecastAmount = Math.max(1, ...monthlyForecast.map((item) => item.amount_due))
  const forecast12MonthTotal = monthlyForecast.reduce((sum, item) => sum + item.amount_due, 0)
  const forecast12MonthPayments = monthlyForecast.reduce((sum, item) => sum + item.occurrence_count, 0)

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/">
                <ArrowLeft className="size-4" />
              </Link>
            </Button>
            <h1 className="text-lg font-bold tracking-tight">{t("reports.title")}</h1>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/settings">
                <Settings className="size-4" />
              </Link>
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-6xl space-y-6 px-4 py-6">
        {loading ? (
          <ReportsSkeleton />
        ) : !report ? (
          <EmptyState title={t("reports.error.title")} description={t("reports.error.description")} />
        ) : (
          <>
            <section className="-mx-4 overflow-x-auto pb-1 sm:mx-0 sm:overflow-visible sm:pb-0">
              <div className="flex snap-x gap-2 pl-4 sm:grid sm:grid-cols-2 sm:gap-3 sm:px-0 lg:grid-cols-4">
                <KpiCard
                  icon={TrendingUp}
                  label={t("reports.kpis.monthly")}
                  value={formatAmount(report.kpis.total_monthly)}
                  detail={t("reports.kpis.yearlyDetail", { amount: formatAmount(report.kpis.total_yearly) })}
                />
                <KpiCard
                  icon={TrendingUp}
                  label={t("reports.yearlyStats.totalYearly")}
                  value={formatAmount(report.kpis.total_yearly)}
                  detail={t("reports.yearlyStats.monthlyAverage", { amount: formatAmount(report.kpis.total_yearly / 12) })}
                />
                <KpiCard
                  icon={RefreshCw}
                  label={t("reports.kpis.committed")}
                  value={formatAmount(report.kpis.committed_monthly)}
                  detail={t("reports.kpis.autoRenewDetail", { count: report.kpis.auto_renew_count })}
                />
                <KpiCard
                  icon={RefreshCw}
                  label={t("reports.yearlyStats.committedYearly")}
                  value={formatAmount(report.kpis.committed_yearly)}
                  detail={t("reports.yearlyStats.autoRenewCount", { count: report.kpis.auto_renew_count })}
                />
                <KpiCard
                  icon={ReceiptText}
                  label={t("reports.kpis.next30Days")}
                  value={formatAmount(report.kpis.due_next_30_days)}
                  detail={t("reports.kpis.renewalDetail", { count: report.kpis.upcoming_renewal_count })}
                />
                <KpiCard
                  icon={CalendarDays}
                  label={t("reports.yearlyStats.forecast12Months")}
                  value={formatAmount(forecast12MonthTotal)}
                  detail={t("reports.yearlyStats.forecastMonths", { count: monthlyForecast.length })}
                />
                <KpiCard
                  icon={Layers3}
                  label={t("reports.kpis.active")}
                  value={String(report.kpis.active_count)}
                  detail={t("reports.kpis.modeDetail", {
                    auto: report.kpis.auto_renew_count,
                    manual: report.kpis.manual_renew_count,
                    canceling: report.kpis.canceling_count,
                  })}
                />
                <KpiCard
                  icon={ReceiptText}
                  label={t("reports.yearlyStats.forecastPayments")}
                  value={String(forecast12MonthPayments)}
                  detail={t("reports.yearlyStats.paymentDetail")}
                />
                <div className="w-2 shrink-0 sm:hidden" aria-hidden="true" />
              </div>
            </section>

            <section>
              <MonthlyForecastPanel
                items={monthlyForecast}
                maxAmount={maxForecastAmount}
                formatAmount={formatAmount}
              />
            </section>

            <section className="grid gap-6 lg:grid-cols-3">
              <BreakdownPanel
                title={t("reports.renewalModes.title")}
                icon={RefreshCw}
                items={renewalModeBreakdown}
                formatAmount={formatAmount}
                emptyTitle={t("reports.empty.title")}
                emptyDescription={t("reports.empty.description")}
                labelForKey={(item) => reportRenewalModeLabel(item.key as SubscriptionRenewalMode, t)}
              />
              <BreakdownPanel
                title={t("reports.categories.title")}
                icon={PieChart}
                items={categoryBreakdown}
                formatAmount={formatAmount}
                emptyTitle={t("reports.empty.title")}
                emptyDescription={t("reports.empty.description")}
                labelForKey={(item) => item.label || t("reports.categories.none")}
              />
              <BreakdownPanel
                title={t("reports.paymentMethods.title")}
                icon={CreditCard}
                items={paymentMethodBreakdown}
                formatAmount={formatAmount}
                emptyTitle={t("reports.empty.title")}
                emptyDescription={t("reports.empty.description")}
                labelForKey={(item) => item.label || t("reports.paymentMethods.none")}
              />
            </section>

            <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
              <TopSubscriptionsPanel items={topSubscriptions} formatAmount={formatAmount} />
              <UpcomingRenewalsPanel
                items={upcomingRenewals}
                formatAmount={formatAmount}
                language={i18n.language}
              />
            </section>

            <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
              <PriceIncreasesPanel
                items={priceIncreases}
                formatAmount={formatAmount}
                language={i18n.language}
              />
              <AnnualGrowthPanel items={annualGrowth} formatAmount={formatAmount} />
            </section>

            <section>
              <RecentChangesPanel items={recentChanges} language={i18n.language} formatAmount={formatAmount} />
            </section>
          </>
        )}
      </main>
    </div>
  )
}
