import { useTranslation } from "react-i18next"
import { CalendarDays, DollarSign, Layers3, TrendingUp } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { formatCurrencyWithSymbol } from "@/lib/utils"
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

  const detailStats = [
    {
      label: t("dashboard.stats.activeYearly"),
      value: formatAmount(summary.total_yearly),
      icon: TrendingUp,
      iconClassName: "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
    },
    {
      label: t("dashboard.stats.activeMonthly"),
      value: formatAmount(summary.total_monthly),
      icon: DollarSign,
      iconClassName: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
    },
  ] as const

  return (
    <Card className="mb-4 overflow-hidden border-border/60 bg-gradient-to-br from-primary/5 via-background to-background py-0 shadow-sm sm:mb-6">
      <CardContent className="grid gap-0 p-0 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <div className="p-4 sm:p-7">
          <div className="flex items-center gap-3 sm:items-start sm:gap-4">
            <div className="hidden rounded-2xl bg-primary/10 p-3 text-primary sm:block">
              <DollarSign className="size-5" />
            </div>

            <div className="min-w-0 flex-1">
              <div className="flex min-w-0 items-baseline justify-between gap-3 sm:block">
                <p className="shrink-0 text-xs font-medium text-muted-foreground sm:text-sm">
                  {t("dashboard.stats.thisMonth")}
                </p>
                <p className="min-w-0 truncate text-right text-xl font-semibold tracking-tight text-foreground tabular-nums sm:mt-3 sm:text-left sm:text-4xl">
                  {formatAmount(summary.due_this_month)}
                </p>
              </div>

              <div className="mt-3 grid grid-cols-2 gap-2 sm:mt-5 sm:flex sm:flex-wrap sm:gap-2.5">
                <div className="inline-flex min-w-0 items-center gap-1.5 rounded-lg border border-border/70 bg-background/80 px-2.5 py-1.5 text-xs shadow-xs sm:gap-2 sm:rounded-full sm:px-3.5 sm:py-2 sm:text-sm">
                  <Layers3 className="size-3.5 shrink-0 text-muted-foreground" />
                  <span className="font-semibold tabular-nums">{summary.active_count ?? 0}</span>
                  <span className="min-w-0 truncate text-muted-foreground">
                    {t("dashboard.stats.activeSubscriptions")}
                  </span>
                </div>

                <div className="inline-flex min-w-0 items-center gap-1.5 rounded-lg border border-border/70 bg-background/80 px-2.5 py-1.5 text-xs shadow-xs sm:gap-2 sm:rounded-full sm:px-3.5 sm:py-2 sm:text-sm">
                  <CalendarDays className="size-3.5 shrink-0 text-muted-foreground" />
                  <span className="font-semibold tabular-nums">
                    {summary.upcoming_renewal_count ?? 0}
                  </span>
                  <span className="min-w-0 truncate text-muted-foreground">{t("dashboard.stats.upcoming")}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid gap-px bg-border/60">
          {detailStats.map(({ icon: Icon, iconClassName, label, value }) => (
            <div key={label} className="bg-background/80 p-3 sm:p-5">
              <div className="flex items-center gap-2.5 sm:gap-3">
                <div className={`rounded-lg p-1.5 sm:rounded-xl sm:p-2 ${iconClassName}`}>
                  <Icon className="size-4" />
                </div>

                <div className="min-w-0">
                  <p className="text-xs font-medium uppercase tracking-[0.08em] text-muted-foreground sm:tracking-[0.12em]">
                    {label}
                  </p>
                  <p className="mt-1 truncate text-base font-semibold text-foreground tabular-nums sm:text-lg">
                    {value}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
