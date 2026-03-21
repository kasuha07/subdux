import type { Subscription, SubscriptionRenewalMode, SubscriptionStatus } from "@/types"

export function getSubscriptionStatus(subscription: Subscription): SubscriptionStatus {
  return subscription.status || "active"
}

export function isSubscriptionActive(subscription: Subscription): boolean {
  return getSubscriptionStatus(subscription) === "active"
}

export function isSubscriptionEnded(subscription: Subscription): boolean {
  return getSubscriptionStatus(subscription) === "ended"
}

export function getSubscriptionRenewalMode(subscription: Subscription): SubscriptionRenewalMode {
  if (subscription.renewal_mode) {
    return subscription.renewal_mode
  }

  if (subscription.billing_type === "recurring" && isSubscriptionActive(subscription)) {
    return "auto_renew"
  }

  if (isSubscriptionEnded(subscription)) {
    return "cancel_at_period_end"
  }

  return "manual_renew"
}

export function getSubscriptionEndsAt(subscription: Subscription): string | null {
  if (subscription.ends_at) {
    return subscription.ends_at
  }
  return isSubscriptionEnded(subscription) ? subscription.next_billing_date : null
}

export function hasFutureRecurringSchedule(subscription: Subscription): boolean {
  return (
    subscription.billing_type === "recurring" &&
    isSubscriptionActive(subscription) &&
    getSubscriptionRenewalMode(subscription) === "auto_renew"
  )
}
