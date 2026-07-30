import { api } from "@/lib/api"
import type { AnalyticsReport, UserCurrency } from "@/types"

export function requestReportsData(
  get: typeof api.get = api.get
): Promise<[AnalyticsReport, UserCurrency[]]> {
  return Promise.all([
    get<AnalyticsReport>("/reports/analytics"),
    get<UserCurrency[]>("/currencies"),
  ])
}
