import { useTranslation } from "react-i18next"
import { CalendarDays, DollarSign, Repeat, TrendingUp } from "lucide-react"

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

  return (
    <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <DollarSign className="size-4" />
            <span className="text-xs font-medium uppercase tracking-wider">
              {t("dashboard.stats.monthly")}
            </span>
          </div>
          <p className="mt-1 text-2xl font-bold tabular-nums">
            {formatCurrencyWithSymbol(summary.total_monthly, displayCurrency, currencySymbol, language)}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <TrendingUp className="size-4" />
            <span className="text-xs font-medium uppercase tracking-wider">
              {t("dashboard.stats.yearly")}
            </span>
          </div>
          <p className="mt-1 text-2xl font-bold tabular-nums">
            {formatCurrencyWithSymbol(summary.total_yearly, displayCurrency, currencySymbol, language)}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Repeat className="size-4" />
            <span className="text-xs font-medium uppercase tracking-wider">
              {t("dashboard.stats.thisMonth")}
            </span>
          </div>
          <p className="mt-1 text-2xl font-bold tabular-nums">
            {formatCurrencyWithSymbol(summary.due_this_month, displayCurrency, currencySymbol, language)}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <CalendarDays className="size-4" />
            <span className="text-xs font-medium uppercase tracking-wider">
              {t("dashboard.stats.upcoming")}
            </span>
          </div>
          <p className="mt-1 text-2xl font-bold tabular-nums">
            {summary.upcoming_renewal_count ?? 0}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
