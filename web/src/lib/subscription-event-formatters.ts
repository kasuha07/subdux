import type { TFunction } from "i18next"

import { formatDate } from "@/lib/utils"
import type {
  SubscriptionDetailEvent,
  SubscriptionRenewalMode,
  SubscriptionStatus,
} from "@/types"

interface EventAmountChange {
  new_amount: number | null
  new_currency?: string
  previous_amount: number | null
  previous_currency?: string
}

export interface SubscriptionEventChangeRow {
  label: string
  next: string
  previous: string
}

export function subscriptionEventTypeLabel(type: string, t: TFunction): string {
  return t(`reports.recentChanges.types.${type}`)
}

export function subscriptionEventFieldLabel(field: string, t: TFunction): string {
  const translated = t(`reports.recentChanges.fields.${field}`)
  if (translated === `reports.recentChanges.fields.${field}`) {
    return field
  }
  return translated
}

export function reportRenewalModeLabel(mode: SubscriptionRenewalMode, t: TFunction): string {
  return t(`reports.renewalModes.${mode}`)
}

export function subscriptionLifecycleStatusLabel(value: SubscriptionStatus | "", t: TFunction): string {
  if (!value) {
    return ""
  }
  return t(`subscription.card.status.${value}`, { defaultValue: value })
}

export function subscriptionRenewalModeLabel(value: SubscriptionRenewalMode | "", t: TFunction): string {
  if (!value) {
    return ""
  }
  return t(`subscription.card.renewalMode.${value}`, { defaultValue: value })
}

export function formatSubscriptionEventAmountChange(
  item: EventAmountChange,
  formatAmount: (amount: number, currency?: string) => string
): string {
  if (item.previous_amount !== null && item.new_amount !== null) {
    return `${formatAmount(item.previous_amount, item.previous_currency)} -> ${formatAmount(item.new_amount, item.new_currency)}`
  }
  if (item.new_amount !== null) {
    return formatAmount(item.new_amount, item.new_currency)
  }
  if (item.previous_amount !== null) {
    return formatAmount(item.previous_amount, item.previous_currency)
  }
  return ""
}

export function subscriptionDetailEventChangeRows(
  item: SubscriptionDetailEvent,
  formatAmount: (amount: number, currency?: string) => string,
  language: string,
  t: TFunction
): SubscriptionEventChangeRow[] {
  const fields = item.changed_fields.length > 0 ? item.changed_fields : ["amount"]
  if (item.type === "created") {
    return ["amount", "monthly_amount", "next_billing_date", "status", "renewal_mode", "category", "payment_method"]
      .map((field) => subscriptionDetailEventChangeRow(field, item, formatAmount, language, t, "created"))
      .filter((row): row is SubscriptionEventChangeRow => row !== null)
  }
  if (item.type === "deleted") {
    return ["amount", "monthly_amount", "next_billing_date", "status", "renewal_mode", "category", "payment_method"]
      .map((field) => subscriptionDetailEventChangeRow(field, item, formatAmount, language, t, "deleted"))
      .filter((row): row is SubscriptionEventChangeRow => row !== null)
  }
  return fields
    .map((field) => subscriptionDetailEventChangeRow(field, item, formatAmount, language, t))
    .filter((row): row is SubscriptionEventChangeRow => row !== null)
}

function subscriptionDetailEventChangeRow(
  field: string,
  item: SubscriptionDetailEvent,
  formatAmount: (amount: number, currency?: string) => string,
  language: string,
  t: TFunction,
  mode?: "created" | "deleted"
): SubscriptionEventChangeRow | null {
  const empty = t("subscription.detail.empty.none")
  const label = subscriptionEventFieldLabel(field, t)

  function valueForText(previous: string, next: string): { previous: string, next: string } | null {
    if (!previous && !next) {
      return null
    }
    if (mode === "created") {
      return { previous: empty, next: next || empty }
    }
    if (mode === "deleted") {
      return { previous: previous || empty, next: empty }
    }
    return { previous: previous || empty, next: next || empty }
  }

  if (field === "amount") {
    const values = valueForAmount(
      item.previous_amount,
      item.previous_currency || item.new_currency,
      item.new_amount,
      item.new_currency || item.previous_currency,
      formatAmount,
      mode,
      empty
    )
    return values ? { label, ...values } : null
  }
  if (field === "monthly_amount") {
    const values = valueForAmount(
      item.previous_monthly_amount,
      item.previous_currency || item.new_currency,
      item.new_monthly_amount,
      item.new_currency || item.previous_currency,
      formatAmount,
      mode,
      empty
    )
    return values ? { label, ...values } : null
  }
  if (field === "currency") {
    const values = valueForText(item.previous_currency, item.new_currency)
    return values ? { label, ...values } : null
  }
  if (field === "next_billing_date") {
    const previous = item.previous_next_billing_date
      ? formatDate(item.previous_next_billing_date, language)
      : ""
    const next = item.new_next_billing_date
      ? formatDate(item.new_next_billing_date, language)
      : ""
    const values = valueForText(previous, next)
    return values ? { label, ...values } : null
  }
  if (field === "status") {
    const values = valueForText(
      subscriptionLifecycleStatusLabel(item.previous_status, t),
      subscriptionLifecycleStatusLabel(item.new_status, t)
    )
    return values ? { label, ...values } : null
  }
  if (field === "renewal_mode") {
    const values = valueForText(
      subscriptionRenewalModeLabel(item.previous_renewal_mode, t),
      subscriptionRenewalModeLabel(item.new_renewal_mode, t)
    )
    return values ? { label, ...values } : null
  }
  if (field === "category") {
    const values = valueForText(item.previous_category_name, item.new_category_name)
    return values ? { label, ...values } : null
  }
  if (field === "payment_method") {
    const values = valueForText(item.previous_payment_method_name, item.new_payment_method_name)
    return values ? { label, ...values } : null
  }
  return null
}

function valueForAmount(
  previousAmount: number | null,
  previousCurrency: string,
  newAmount: number | null,
  newCurrency: string,
  formatAmount: (amount: number, currency?: string) => string,
  mode: "created" | "deleted" | undefined,
  empty: string
): { previous: string, next: string } | null {
  if (previousAmount === null && newAmount === null) {
    return null
  }
  const previous = previousAmount !== null ? formatAmount(previousAmount, previousCurrency) : ""
  const next = newAmount !== null ? formatAmount(newAmount, newCurrency) : ""
  if (mode === "created") {
    return { previous: empty, next: next || empty }
  }
  if (mode === "deleted") {
    return { previous: previous || empty, next: empty }
  }
  return { previous: previous || empty, next: next || empty }
}
