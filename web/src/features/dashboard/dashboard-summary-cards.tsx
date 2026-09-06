import { useTranslation } from "react-i18next"
import { Link } from "react-router"
import { ArrowUpRight, CalendarDays, CreditCard, DollarSign, Layers3, TrendingUp } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { preloadRouteForPath } from "@/lib/route-preload"
import { cn, formatCurrencyWithSymbol } from "@/lib/utils"
import type { DashboardSummary } from "@/types"

interface DashboardSummaryCardsProps {
  currencySymbol?: string
  language: string
  preferredCurrency: string
  summary: DashboardSummary
}

export default function DashboardSummaryCards({
  currencySymbol,
  language,
  preferredCurrency,
  summary,
}: DashboardSummaryCardsProps) {
  const { t } = useTranslation()
  const displayCurrency = summary.currency || preferredCurrency
  const formatAmount = (amount: number) =>
    formatCurrencyWithSymbol(amount, displayCurrency, currencySymbol, language)

  const upcomingCount = summary.upcoming_renewal_count ?? 0
  const hasUpcoming = upcomingCount > 0

  const detailStats = [
    {
      label: t("dashboard.stats.activeMonthly"),
      value: formatAmount(summary.total_monthly),
      icon: DollarSign,
      iconClassName: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
    },
    {
      label: t("dashboard.stats.activeYearly"),
      value: formatAmount(summary.total_yearly),
      icon: TrendingUp,
      iconClassName: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    },
  ] as const

  return (
    <Card className="mb-4 overflow-hidden border-border/70 bg-gradient-to-br from-primary/[0.03] via-card to-card py-0 shadow-xs dark:from-primary/[0.05] dark:via-card dark:to-card sm:mb-6">
      <CardContent className="grid gap-0 p-0 md:grid-cols-[minmax(0,1.3fr)_minmax(0,0.9fr)]">
        {/* Left Hero: This Month's Due */}
        <div className="flex flex-col justify-between p-4 sm:p-6 lg:p-7">
          <div>
            <div className="flex items-center gap-2.5 sm:gap-3">
              <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary shadow-2xs sm:size-10 sm:rounded-xl">
                <CreditCard className="size-4 sm:size-5" />
              </div>

              <div className="flex min-w-0 items-center gap-2">
                <span className="truncate text-xs font-medium text-muted-foreground sm:text-sm">
                  {t("dashboard.stats.thisMonth")}
                </span>
                {displayCurrency && (
                  <span className="rounded-md border border-border/60 bg-muted/60 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    {displayCurrency}
                  </span>
                )}
              </div>
            </div>

            <div className="mt-2.5 sm:mt-3">
              <p
                className="truncate text-2xl font-bold tracking-tight text-foreground tabular-nums sm:text-3xl lg:text-4xl"
                title={formatAmount(summary.due_this_month)}
              >
                {formatAmount(summary.due_this_month)}
              </p>
            </div>
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-2 sm:mt-5 sm:gap-2.5">
            <div className="inline-flex min-w-0 items-center gap-1.5 rounded-full border border-border/70 bg-background/80 px-2.5 py-1 text-xs font-medium text-muted-foreground shadow-2xs backdrop-blur-xs sm:gap-2 sm:px-3.5 sm:py-1.5 sm:text-sm">
              <Layers3 className="size-3.5 shrink-0 text-foreground/70" />
              <span className="font-semibold tabular-nums text-foreground">
                {summary.active_count ?? 0}
              </span>
              <span className="truncate">{t("dashboard.stats.activeSubscriptions")}</span>
            </div>

            <Link
              to="/actions"
              onPointerEnter={() => preloadRouteForPath("/actions")}
              onFocus={() => preloadRouteForPath("/actions")}
              className={cn(
                "inline-flex min-w-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-medium shadow-2xs backdrop-blur-xs transition-all duration-150 sm:gap-2 sm:px-3.5 sm:py-1.5 sm:text-sm",
                "focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring active:scale-[0.98]",
                hasUpcoming
                  ? "border-amber-500/30 bg-amber-500/10 text-amber-900 hover:border-amber-500/50 hover:bg-amber-500/15 dark:border-amber-500/30 dark:text-amber-200"
                  : "border-border/70 bg-background/80 text-muted-foreground hover:bg-muted/80 hover:text-foreground"
              )}
            >
              <CalendarDays
                className={cn(
                  "size-3.5 shrink-0",
                  hasUpcoming ? "text-amber-600 dark:text-amber-400" : "text-foreground/70"
                )}
              />
              <span
                className={cn(
                  "font-semibold tabular-nums",
                  hasUpcoming ? "text-amber-900 dark:text-amber-200" : "text-foreground"
                )}
              >
                {upcomingCount}
              </span>
              <span className="truncate">{t("dashboard.stats.upcoming")}</span>
              {hasUpcoming && <ArrowUpRight className="size-3 shrink-0 opacity-70" />}
            </Link>
          </div>
        </div>

        {/* Right / Secondary Stats: Active Monthly & Active Yearly */}
        <div className="grid grid-cols-2 divide-x divide-border/60 border-t border-border/60 bg-muted/15 md:grid-cols-1 md:divide-x-0 md:divide-y md:border-t-0 md:border-l">
          {detailStats.map(({ icon: Icon, iconClassName, label, value }) => (
            <div
              key={label}
              className="flex flex-col justify-between p-3.5 transition-colors hover:bg-muted/30 sm:p-4.5 md:flex-row md:items-center md:justify-start md:gap-3.5 md:p-5"
            >
              <div className="flex items-center gap-2 md:contents">
                <div
                  className={cn(
                    "flex size-7 shrink-0 items-center justify-center rounded-lg md:size-10 md:rounded-xl",
                    iconClassName
                  )}
                >
                  <Icon className="size-3.5 md:size-5" />
                </div>

                <p className="truncate text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground sm:text-xs md:hidden">
                  {label}
                </p>
              </div>

              <div className="mt-2 min-w-0 flex-1 md:mt-0">
                <p className="hidden text-xs font-medium uppercase tracking-[0.08em] text-muted-foreground sm:tracking-[0.1em] md:block">
                  {label}
                </p>
                <p
                  className="truncate text-base font-semibold tracking-tight text-foreground tabular-nums sm:text-lg md:mt-1 md:text-xl"
                  title={value}
                >
                  {value}
                </p>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export function DashboardSummaryCardsSkeleton() {
  return (
    <Card className="mb-4 overflow-hidden border-border/70 bg-gradient-to-br from-primary/[0.03] via-card to-card py-0 shadow-xs sm:mb-6">
      <CardContent className="grid gap-0 p-0 md:grid-cols-[minmax(0,1.3fr)_minmax(0,0.9fr)]">
        <div className="p-4 sm:p-6 lg:p-7">
          <div className="flex items-center gap-2.5 sm:gap-3">
            <Skeleton className="size-8 rounded-lg sm:size-10 sm:rounded-xl" />
            <Skeleton className="h-4 w-24 sm:w-28" />
            <Skeleton className="h-4 w-10 rounded-md" />
          </div>

          <div className="mt-2.5 sm:mt-3">
            <Skeleton className="h-8 w-44 sm:h-9 sm:w-56 lg:h-10 lg:w-64" />
          </div>

          <div className="mt-4 flex flex-wrap items-center gap-2 sm:mt-5 sm:gap-2.5">
            <Skeleton className="h-7 w-32 rounded-full sm:h-8 sm:w-36" />
            <Skeleton className="h-7 w-28 rounded-full sm:h-8 sm:w-32" />
          </div>
        </div>

        <div className="grid grid-cols-2 divide-x divide-border/60 border-t border-border/60 bg-muted/15 md:grid-cols-1 md:divide-x-0 md:divide-y md:border-t-0 md:border-l">
          {Array.from({ length: 2 }).map((_, i) => (
            <div
              key={i}
              className="flex flex-col justify-between p-3.5 sm:p-4.5 md:flex-row md:items-center md:justify-start md:gap-3.5 md:p-5"
            >
              <div className="flex items-center gap-2 md:contents">
                <Skeleton className="size-7 rounded-lg md:size-10 md:rounded-xl" />
                <Skeleton className="h-3 w-16 md:hidden" />
              </div>
              <div className="mt-2 space-y-1.5 md:mt-0 md:min-w-0 md:flex-1">
                <Skeleton className="hidden h-3.5 w-20 md:block" />
                <Skeleton className="h-5 w-24 sm:h-6 sm:w-28" />
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
