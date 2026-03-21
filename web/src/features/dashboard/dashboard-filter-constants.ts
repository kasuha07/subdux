import type { SubscriptionRenewalMode, SubscriptionStatus } from "@/types"

export type SortField = "next_billing_date" | "name" | "created_at" | "amount"
export type SortDirection = "asc" | "desc"
export type StatusFilter = SubscriptionStatus

export const defaultSortField: SortField = "next_billing_date"
export const defaultSortDirection: SortDirection = "asc"
export const statusOptions: StatusFilter[] = ["active", "ended"]
export const renewalModeOptions: SubscriptionRenewalMode[] = [
  "auto_renew",
  "manual_renew",
  "cancel_at_period_end",
]
export const sortFieldOptions: SortField[] = [
  "next_billing_date",
  "name",
  "created_at",
  "amount",
]
