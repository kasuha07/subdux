import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"
import { BellOff, ExternalLink, Pencil } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { getBrandIconFromValue } from "@/lib/brand-icons"
import { daysUntil, formatCurrencyWithSymbol, formatDate } from "@/lib/utils"
import type { Subscription } from "@/types"
import SubscriptionCycleProgressBar from "./subscription-cycle-progress-bar"

function renderIcon(icon: string, name: string): ReactNode {
  const fallbackInitial = (
    <span className="flex size-full items-center justify-center text-sm font-bold text-foreground">
      {name.charAt(0).toUpperCase()}
    </span>
  )

  if (!icon) {
    return fallbackInitial
  }

  const brand = getBrandIconFromValue(icon)
  if (brand) {
    const { Icon } = brand
    return <Icon size={22} color="default" />
  }

  if (icon.startsWith("http://") || icon.startsWith("https://")) {
    return <img src={icon} alt={name} className="h-7 w-7 object-contain" />
  }

  if (icon.startsWith("file:")) {
    const filename = icon.slice("file:".length)
    if (filename && !filename.includes("/") && !filename.includes("\\")) {
      return <img src={`/uploads/icons/${filename}`} alt={name} className="h-7 w-7 object-contain" />
    }
  }

  if (icon.includes(":")) {
    return fallbackInitial
  }

  return <span className="text-lg leading-none">{icon}</span>
}

interface SubscriptionSquareCardProps {
  subscription: Subscription
  categoryName?: string
  currencySymbol?: string
  displayAmount?: number
  displayCurrency?: string
  displayCurrencySymbol?: string
  showMonthlyAmount?: boolean
  showCycleProgress?: boolean
  paymentMethodName?: string
  onEdit: (sub: Subscription) => void
}

const statusStyles: Record<string, string> = {
  enabled: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  disabled: "bg-zinc-500/10 text-zinc-500 border-zinc-200",
}

export default function SubscriptionSquareCard({
  subscription,
  categoryName,
  currencySymbol,
  displayAmount,
  displayCurrency,
  displayCurrencySymbol,
  showMonthlyAmount = false,
  showCycleProgress = false,
  paymentMethodName,
  onEdit,
}: SubscriptionSquareCardProps) {
  const { t, i18n } = useTranslation()
  const amountToDisplay = displayAmount ?? subscription.amount
  const currencyToDisplay = displayCurrency ?? subscription.currency
  const symbolToDisplay = displayCurrencySymbol ?? currencySymbol
  const categoryLabel = categoryName?.trim() || subscription.category
  const days = subscription.next_billing_date ? daysUntil(subscription.next_billing_date) : null
  const isUpcoming = days !== null && days >= 0 && days < 7
  const isOverdue = (days ?? 0) < 0

  function renderBillingLabel(): string {
    if (showMonthlyAmount && subscription.billing_type === "recurring") {
      return t("subscription.card.recurrence.monthlyCost")
    }

    if (subscription.billing_type === "one_time") {
      return t("subscription.card.billingType.one_time")
    }

    if (subscription.recurrence_type === "monthly_date") {
      return t("subscription.card.recurrence.monthlyDate", { day: subscription.monthly_day ?? 1 })
    }
    if (subscription.recurrence_type === "yearly_date") {
      return t("subscription.card.recurrence.yearlyDate", {
        month: subscription.yearly_month ?? 1,
        day: subscription.yearly_day ?? 1,
      })
    }
    return t(`subscription.card.recurrence.interval.${subscription.interval_unit}`, {
      count: subscription.interval_count ?? 1,
    })
  }

  function renderDueText(): string {
    if (!subscription.next_billing_date) {
      return t("subscription.card.noNextBilling")
    }
    if (days === 0) {
      return t("subscription.card.dueToday")
    }
    if ((days ?? 0) < 0) {
      return t("subscription.card.overdue")
    }
    return formatDate(subscription.next_billing_date, i18n.language)
  }

  const details = [categoryLabel, paymentMethodName].filter(Boolean).join(" · ")
  const dueBadgeClass = isUpcoming
    ? "bg-amber-500/10 text-amber-700 border-amber-200"
    : isOverdue
      ? "bg-destructive/10 text-destructive border-destructive/30"
      : "bg-zinc-500/10 text-zinc-600 border-zinc-200"
  const reminderOff = subscription.notify_enabled === false

  return (
    <Card className={`group relative h-auto w-full self-start gap-0 overflow-hidden py-2 transition-all hover:shadow-md${subscription.enabled ? "" : " grayscale opacity-60"}`}>
      <CardContent className="flex flex-col gap-2 px-3.5 py-2.5">
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 items-center gap-2">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-lg">
              {renderIcon(subscription.icon, subscription.name)}
            </div>
            <div className="min-w-0">
              <h3 className="truncate text-sm font-medium leading-tight">{subscription.name}</h3>
              {details && <p className="mt-0.5 truncate text-xs text-muted-foreground">{details}</p>}
            </div>
          </div>
          {subscription.url && (
            <a
              href={subscription.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex size-7 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-muted/60 hover:text-foreground"
            >
              <ExternalLink className="size-4" />
            </a>
          )}
        </div>

        <div className="flex items-center justify-between gap-2 rounded-lg bg-muted/35 px-3 py-2">
          <div className="min-w-0 self-center">
            <p className="text-sm tabular-nums leading-tight">
              {formatCurrencyWithSymbol(amountToDisplay, currencyToDisplay, symbolToDisplay, i18n.language)}
            </p>
            <p className="mt-1 truncate text-[11px] text-muted-foreground">{renderBillingLabel()}</p>
          </div>

          <div className="flex shrink-0 flex-col items-end gap-1">
            <Badge variant="outline" className={statusStyles[subscription.enabled ? "enabled" : "disabled"] || ""}>
              {t(`subscription.card.status.${subscription.enabled ? "enabled" : "disabled"}`)}
            </Badge>
            <Badge
              variant="outline"
              className={`bg-zinc-500/10 text-zinc-600 border-zinc-200 px-1.5 ${reminderOff ? "" : "invisible"}`}
              title={reminderOff ? t("subscription.card.reminder.off") : undefined}
              aria-label={reminderOff ? t("subscription.card.reminder.off") : undefined}
              aria-hidden={!reminderOff}
            >
              <BellOff className="size-3" />
            </Badge>
          </div>
        </div>

        <div className="flex items-center justify-between gap-2">
          <Badge variant="outline" className={`max-w-[10rem] truncate ${dueBadgeClass}`}>
            {renderDueText()}
          </Badge>

          <Button
            variant="ghost"
            size="icon"
            className="size-9 shrink-0 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            onClick={() => onEdit(subscription)}
          >
            <Pencil className="size-3.5" />
          </Button>
        </div>
      </CardContent>
      {showCycleProgress ? <SubscriptionCycleProgressBar subscription={subscription} /> : null}
    </Card>
  )
}
