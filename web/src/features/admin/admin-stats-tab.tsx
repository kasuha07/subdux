import { useTranslation } from "react-i18next"
import { BarChart3, DollarSign, Users } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"
import { TabsContent } from "@/components/ui/tabs"
import type { AdminStats } from "@/types"

interface AdminStatsTabProps {
  stats: AdminStats | null
}

export default function AdminStatsTab({ stats }: AdminStatsTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="stats">
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Users className="size-4" />
              <span className="text-xs font-medium uppercase tracking-wider">
                {t("admin.stats.totalUsers")}
              </span>
            </div>
            <p className="mt-2 text-3xl font-bold tabular-nums">{stats?.total_users ?? 0}</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-2 text-muted-foreground">
              <BarChart3 className="size-4" />
              <span className="text-xs font-medium uppercase tracking-wider">
                {t("admin.stats.totalSubscriptions")}
              </span>
            </div>
            <p className="mt-2 text-3xl font-bold tabular-nums">
              {stats?.total_subscriptions ?? 0}
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-2 text-muted-foreground">
              <DollarSign className="size-4" />
              <span className="text-xs font-medium uppercase tracking-wider">
                {t("admin.stats.monthlySpend")}
              </span>
            </div>
            <p className="mt-2 text-3xl font-bold tabular-nums">
              ${(stats?.total_monthly_spend ?? 0).toFixed(2)}
            </p>
          </CardContent>
        </Card>
      </div>
    </TabsContent>
  )
}
