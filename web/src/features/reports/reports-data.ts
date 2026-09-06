import { api } from "@/lib/api"
import type { AnalyticsReport, Subscription, UserCurrency } from "@/types"

export function requestReportsData(
  get: typeof api.get = api.get
): Promise<[AnalyticsReport, UserCurrency[], Subscription[]]> {
  return Promise.all([
    get<AnalyticsReport>("/reports/analytics"),
    get<UserCurrency[]>("/currencies"),
    get<Subscription[]>("/subscriptions"),
  ])
}
