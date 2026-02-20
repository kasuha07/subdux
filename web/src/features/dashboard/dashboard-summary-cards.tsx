import { useTranslation } from "react-i18next"
import { CalendarDays, DollarSign, Repeat, TrendingUp } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { formatCurrency } from "@/lib/utils"
import type { DashboardSummary } from "@/types"

interface DashboardSummaryCardsProps {
  language: string
  preferredCurrency: string
  summary: DashboardSummary
}

export default function DashboardSummaryCards({
  language,
  preferredCurrency,
  summary,
}: DashboardSummaryCardsProps) {
  const { t } = useTranslation()

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
            {formatCurrency(summary.total_monthly, summary.currency || preferredCurrency, language)}
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
            {formatCurrency(summary.total_yearly, summary.currency || preferredCurrency, language)}
          </p>
        </CardContent>
      </Card>
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Repeat className="size-4" />
            <span className="text-xs font-medium uppercase tracking-wider">
              {t("dashboard.stats.enabled")}
            </span>
          </div>
          <p className="mt-1 text-2xl font-bold tabular-nums">{summary.enabled_count}</p>
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
