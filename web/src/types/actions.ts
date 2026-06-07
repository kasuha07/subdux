import type { SubscriptionRenewalMode, SubscriptionStatus } from "./subscription"

export type SubscriptionActionType =
  | "upcoming_renewal"
  | "manual_renewal_due"
  | "ending_soon"
  | "notification_failed"
  | "missing_next_billing"
  | "price_increase"

export type SubscriptionActionSeverity = "critical" | "high" | "medium" | "low"

export interface SubscriptionAction {
  key: string
  type: SubscriptionActionType
  severity: SubscriptionActionSeverity
  needs_decision: boolean
  needs_repair: boolean
  upcoming_charge: boolean
  subscription_id: number
  subscription_name: string
  subscription_icon: string
  amount: number
  currency: string
  renewal_mode: SubscriptionRenewalMode
  status: SubscriptionStatus
  due_date: string | null
  days_until: number | null
  event_date: string | null
  message: string
  detail: string
  previous_monthly_amount: number | null
  new_monthly_amount: number | null
  delta_monthly_amount: number | null
  delta_percentage: number | null
  notification_channel: string
  notification_error: string
  allowed_actions: string[]
  snoozed_until: string | null
}

export interface ActionCenterCounts {
  total: number
  critical: number
  high: number
  medium: number
  low: number
  needs_decision: number
  needs_repair: number
  upcoming_charge: number
  snoozed: number
}

export interface ActionCenter {
  generated_at: string
  window_days: number
  urgent_days: number
  items: SubscriptionAction[]
  counts: ActionCenterCounts
  available_types: SubscriptionActionType[]
}

export interface SnoozeSubscriptionActionInput {
  key: string
  days?: number
  until_date?: string
}

export interface SubscriptionActionSnooze {
  id: number
  user_id: number
  subscription_id: number
  action_key: string
  snoozed_until: string
  created_at: string
  updated_at: string
}
