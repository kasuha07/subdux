import { useTranslation } from "react-i18next"
import { CalendarDays, DollarSign, Layers3, Repeat, TrendingUp } from "lucide-react"

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
      label: t("dashboard.stats.committedMonthly"),
      value: formatAmount(summary.committed_monthly),
      icon: Repeat,
      iconClassName: "bg-violet-500/10 text-violet-600 dark:text-violet-400",
    },
    {
      label: t("dashboard.stats.committedYearly"),
      value: formatAmount(summary.committed_yearly),
      icon: CalendarDays,
      iconClassName: "bg-sky-500/10 text-sky-600 dark:text-sky-400",
    },
    {
      label: t("dashboard.stats.thisMonth"),
      value: formatAmount(summary.due_this_month),
      icon: DollarSign,
      iconClassName: "bg-amber-500/10 text-amber-600 dark:text-amber-400",
    },
  ] as const

  return (
    <Card className="mb-6 overflow-hidden border-border/60 bg-gradient-to-br from-primary/5 via-background to-background py-0 shadow-sm">
      <CardContent className="grid gap-0 p-0 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
        <div className="p-6 sm:p-7">
          <div className="flex items-start gap-4">
            <div className="rounded-2xl bg-primary/10 p-3 text-primary">
              <DollarSign className="size-5" />
            </div>

            <div className="min-w-0 flex-1">
              <p className="text-sm font-medium text-muted-foreground">
                {t("dashboard.stats.activeMonthly")}
              </p>
              <p className="mt-3 text-3xl font-semibold tracking-tight text-foreground tabular-nums sm:text-4xl">
                {formatAmount(summary.total_monthly)}
              </p>

              <div className="mt-5 flex flex-wrap gap-2.5">
                <div className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/80 px-3.5 py-2 text-sm shadow-xs">
                  <span className="rounded-full bg-muted p-1 text-muted-foreground">
                    <Layers3 className="size-3.5" />
                  </span>
                  <span className="font-semibold tabular-nums">{summary.active_count ?? 0}</span>
                  <span className="text-muted-foreground">
                    {t("dashboard.stats.activeSubscriptions")}
                  </span>
                </div>

                <div className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/80 px-3.5 py-2 text-sm shadow-xs">
                  <span className="rounded-full bg-muted p-1 text-muted-foreground">
                    <CalendarDays className="size-3.5" />
                  </span>
                  <span className="font-semibold tabular-nums">
                    {summary.upcoming_renewal_count ?? 0}
                  </span>
                  <span className="text-muted-foreground">{t("dashboard.stats.upcoming")}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="grid gap-px bg-border/60 sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
          {detailStats.map(({ icon: Icon, iconClassName, label, value }) => (
            <div key={label} className="bg-background/80 p-4 sm:p-5">
              <div className="flex items-center gap-3">
                <div className={`rounded-xl p-2 ${iconClassName}`}>
                  <Icon className="size-4" />
                </div>

                <div className="min-w-0">
                  <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">
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
