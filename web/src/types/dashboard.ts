export interface DashboardSummary {
  total_monthly: number
  total_yearly: number
  committed_monthly: number
  committed_yearly: number
  due_this_month: number
  active_count?: number
  upcoming_renewal_count: number
  currency: string
}
