export type SortField = "next_billing_date" | "name" | "created_at" | "amount"
export type SortDirection = "asc" | "desc"
export type EnabledFilter = "enabled" | "disabled"

export const defaultSortField: SortField = "next_billing_date"
export const defaultSortDirection: SortDirection = "asc"
export const enabledOptions: EnabledFilter[] = ["enabled", "disabled"]
export const sortFieldOptions: SortField[] = [
  "next_billing_date",
  "name",
  "created_at",
  "amount",
]
