import { useTranslation } from "react-i18next"

import { isSubscriptionActive } from "@/features/subscriptions/subscription-lifecycle"
import { daysUntil, formatCurrency, formatDate } from "@/lib/utils"
import type { Subscription } from "@/types"

import { getIntroKind, regularAmount, remainingIntroCharges } from "../subscription-pending-price"

// Short label ("Trial · 5d left") and follow-up ("$30 from Mar 8") for an
// active subscription running a free trial or intro price; null otherwise.
export function useSubscriptionIntroText(subscription: Subscription) {
  const { t, i18n } = useTranslation()
  const kind = isSubscriptionActive(subscription) ? getIntroKind(subscription) : null
  if (!kind || !subscription.pending_from) {
    return null
  }

  const label = kind === "trial"
    ? t("subscription.card.intro.trial", { count: Math.max(0, daysUntil(subscription.pending_from)) })
    : t("subscription.card.intro.intro", { count: remainingIntroCharges(subscription) })
  const after = t("subscription.card.intro.after", {
    date: formatDate(subscription.pending_from, i18n.language),
    amount: formatCurrency(regularAmount(subscription), subscription.currency, i18n.language),
  })
  return { kind, label, after }
}
