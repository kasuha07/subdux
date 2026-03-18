import type { Subscription } from "@/types"

export function getAmountInTargetCurrency(
  amount: number,
  sourceCurrency: string,
  targetCurrency: string,
  exchangeRates: Record<string, number>
): number {
  const normalizedSource = sourceCurrency.trim().toUpperCase()
  const normalizedTarget = targetCurrency.trim().toUpperCase()

  if (!normalizedSource || !normalizedTarget || normalizedSource === normalizedTarget) {
    return amount
  }

  const rate = exchangeRates[`${normalizedSource}->${normalizedTarget}`]
  if (typeof rate !== "number" || !Number.isFinite(rate) || rate <= 0) {
    return amount
  }

  return amount * rate
}

export function getMonthlyAmountFactor(subscription: Subscription): number | null {
  if (subscription.billing_type !== "recurring") {
    return null
  }

  if (subscription.recurrence_type === "interval") {
    const intervalCount = subscription.interval_count
    if (!intervalCount || intervalCount <= 0) {
      return null
    }

    switch (subscription.interval_unit) {
      case "day":
        return 30.436875 / intervalCount
      case "week":
        return 4.348125 / intervalCount
      case "month":
        return 1 / intervalCount
      case "year":
        return 1 / (12 * intervalCount)
      default:
        return null
    }
  }

  if (subscription.recurrence_type === "monthly_date") {
    return 1
  }

  if (subscription.recurrence_type === "yearly_date") {
    return 1 / 12
  }

  return null
}

export function getComparableSubscriptionAmount(
  subscription: Subscription,
  preferredCurrency: string,
  exchangeRates: Record<string, number>
): number {
  const amountInTargetCurrency = getAmountInTargetCurrency(
    subscription.amount,
    subscription.currency,
    preferredCurrency,
    exchangeRates
  )
  const monthlyFactor = getMonthlyAmountFactor(subscription)

  return monthlyFactor ? amountInTargetCurrency * monthlyFactor : amountInTargetCurrency
}
