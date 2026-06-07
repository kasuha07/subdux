export type SubscriptionStatus = "active" | "ended"
export type SubscriptionRenewalMode = "auto_renew" | "manual_renew" | "cancel_at_period_end"

export interface Subscription {
  id: number
  name: string
  amount: number
  currency: string
  status: SubscriptionStatus
  renewal_mode: SubscriptionRenewalMode
  ends_at: string | null
  billing_type: "recurring"
  recurrence_type: "interval" | "monthly_date" | "yearly_date" | ""
  interval_count: number | null
  interval_unit: "day" | "week" | "month" | "year" | ""
  monthly_day: number | null
  yearly_month: number | null
  yearly_day: number | null
  next_billing_date: string | null
  category: string
  category_id: number | null
  payment_method_id: number | null
  notify_enabled: boolean | null
  notify_days_before: number | null
  icon: string
  url: string
  notes: string
  created_at: string
  updated_at: string
}

export type SubscriptionEventType = "created" | "updated" | "manual_renewed" | "deleted" | "system_change"

export interface SubscriptionDetailEvent {
  id: number
  type: SubscriptionEventType
  changed_fields: string[]
  previous_amount: number | null
  new_amount: number | null
  previous_monthly_amount: number | null
  new_monthly_amount: number | null
  previous_currency: string
  new_currency: string
  previous_next_billing_date: string | null
  new_next_billing_date: string | null
  previous_status: SubscriptionStatus | ""
  new_status: SubscriptionStatus | ""
  previous_renewal_mode: SubscriptionRenewalMode | ""
  new_renewal_mode: SubscriptionRenewalMode | ""
  previous_category_name: string
  new_category_name: string
  previous_payment_method_name: string
  new_payment_method_name: string
  changed_at: string
}

export interface SubscriptionDetailPriceHistoryItem {
  event_id: number
  type: SubscriptionEventType
  amount: number
  currency: string
  monthly_amount: number | null
  previous_amount: number | null
  previous_currency: string
  previous_monthly_amount: number | null
  changed_at: string
}

export interface SubscriptionDetailNotificationLog {
  id: number
  channel_type: string
  notify_date: string
  status: string
  error: string
  sent_at: string
}

export interface SubscriptionDetailUpcomingCharge {
  date: string
  amount: number
  currency: string
  renewal_mode: SubscriptionRenewalMode
}

export interface SubscriptionDetailCalendar {
  path: string
  feed_path: string
  has_upcoming_event: boolean
  next_event_date: string | null
}

export interface SubscriptionDetail {
  subscription: Subscription
  timeline: SubscriptionDetailEvent[]
  price_history: SubscriptionDetailPriceHistoryItem[]
  notification_logs: SubscriptionDetailNotificationLog[]
  upcoming_charges: SubscriptionDetailUpcomingCharge[]
  calendar: SubscriptionDetailCalendar
}

export interface CreateSubscriptionInput {
  name: string
  amount: number
  currency: string
  status: SubscriptionStatus
  renewal_mode: SubscriptionRenewalMode
  ends_at: string | null
  billing_type: "recurring"
  recurrence_type: string
  interval_count: number | null
  interval_unit: string
  next_billing_date: string
  monthly_day: number | null
  yearly_month: number | null
  yearly_day: number | null
  category: string
  category_id: number | null
  payment_method_id: number | null
  notify_enabled: boolean | null
  notify_days_before: number | null
  icon: string
  url: string
  notes: string
}

export interface UpdateSubscriptionInput {
  name?: string
  amount?: number
  currency?: string
  status?: SubscriptionStatus
  renewal_mode?: SubscriptionRenewalMode
  ends_at?: string | null
  billing_type?: "recurring"
  recurrence_type?: string
  interval_count?: number | null
  interval_unit?: string
  next_billing_date?: string
  monthly_day?: number | null
  yearly_month?: number | null
  yearly_day?: number | null
  category?: string
  category_id?: number | null
  payment_method_id?: number | null
  notify_enabled?: boolean | null
  notify_days_before?: number | null
  icon?: string
  url?: string
  notes?: string
}
