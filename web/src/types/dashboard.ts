import type { Subscription } from "./subscription"
import type { Category, PaymentMethod, UserCurrency } from "./settings"

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

// DashboardBootstrap is the aggregated first-screen payload returned by
// GET /api/dashboard/bootstrap. It replaces six parallel requests with one,
// reconciling subscription lifecycle a single time on the server.
export interface DashboardBootstrap {
  subscriptions: Subscription[]
  summary: DashboardSummary | null
  categories: Category[]
  payment_methods: PaymentMethod[]
  currencies: UserCurrency[]
  preferred_currency: string
}
