import type { SubscriptionEventType, SubscriptionRenewalMode } from "./subscription"

export interface AnalyticsReportKPIs {
  active_count: number
  auto_renew_count: number
  manual_renew_count: number
  canceling_count: number
  total_monthly: number
  total_yearly: number
  committed_monthly: number
  committed_yearly: number
  due_this_month: number
  due_next_30_days: number
  upcoming_renewal_count: number
}

export interface MonthlyForecastItem {
  month: string
  amount_due: number
  occurrence_count: number
}

export interface ReportBreakdownItem {
  key: string
  label: string
  count: number
  monthly_amount: number
  yearly_amount: number
  percentage: number
}

export interface ReportSubscriptionSpend {
  id: number
  name: string
  icon: string
  category: string
  payment_method: string
  renewal_mode: SubscriptionRenewalMode
  next_billing_date: string
  monthly_amount: number
  yearly_amount: number
  original_amount: number
  original_currency: string
}

export interface ReportUpcomingRenewal {
  id: number
  name: string
  icon: string
  billing_date: string
  days_until: number
  amount: number
  category: string
  payment_method: string
  renewal_mode: SubscriptionRenewalMode
}

export interface ReportPriceIncrease {
  subscription_id: number
  name: string
  previous_monthly_amount: number
  new_monthly_amount: number
  delta_monthly_amount: number
  delta_percentage: number
  currency: string
  changed_at: string
}

export interface ReportSubscriptionEvent {
  id: number
  subscription_id: number | null
  name: string
  type: SubscriptionEventType
  changed_fields: string[]
  previous_amount: number | null
  new_amount: number | null
  previous_currency: string
  new_currency: string
  changed_at: string
}

export interface ReportAnnualGrowthItem {
  subscription_id: number
  name: string
  baseline_monthly_amount: number
  current_monthly_amount: number
  delta_monthly_amount: number
  delta_percentage: number
  currency: string
}

export interface AnalyticsReport {
  currency: string
  generated_at: string
  kpis: AnalyticsReportKPIs
  monthly_forecast: MonthlyForecastItem[]
  category_breakdown: ReportBreakdownItem[]
  payment_method_breakdown: ReportBreakdownItem[]
  renewal_mode_breakdown: ReportBreakdownItem[]
  top_subscriptions: ReportSubscriptionSpend[]
  upcoming_renewals: ReportUpcomingRenewal[]
  price_increases: ReportPriceIncrease[]
  recent_changes: ReportSubscriptionEvent[]
  annual_growth: ReportAnnualGrowthItem[]
}
