import type { ReactNode } from "react"
import type { Subscription } from "@/types"
import { useTranslation } from "react-i18next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { formatCurrencyWithSymbol, daysUntil, formatDate } from "@/lib/utils"
import { Pencil, Trash2, ExternalLink, BellOff } from "lucide-react"
import { getBrandIcon } from "@/lib/brand-icons"

function renderIcon(icon: string, name: string): ReactNode {
  const fallbackInitial = (
    <span className="flex size-full items-center justify-center bg-muted text-sm font-bold text-foreground">
      {name.charAt(0).toUpperCase()}
    </span>
  )

  if (!icon) {
    return fallbackInitial
  }

  if (icon.startsWith("si:")) {
    const brand = getBrandIcon(icon.slice(3))
    if (brand) {
      const { Icon } = brand
      return <Icon size={20} color="default" />
    }
    return fallbackInitial
  }

  if (icon.startsWith("http://") || icon.startsWith("https://")) {
    return (
      <img
        src={icon}
        alt={name}
        className="h-6 w-6 object-contain"
      />
    )
  }

  if (icon.startsWith("assets/")) {
    const assetPath = icon.slice("assets/".length)
    return (
      <img
        src={`/uploads/${assetPath}`}
        alt={name}
        className="h-6 w-6 object-contain"
      />
    )
  }

  return <span className="text-lg leading-none">{icon}</span>
}

interface SubscriptionCardProps {
  subscription: Subscription
  categoryName?: string
  currencySymbol?: string
  displayAmount?: number
  displayCurrency?: string
  displayCurrencySymbol?: string
  showMonthlyAmount?: boolean
  paymentMethodName?: string
  paymentMethodIcon?: string
  onEdit: (sub: Subscription) => void
  onDelete: (id: number) => void
}

const statusStyles: Record<string, string> = {
  enabled: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  disabled: "bg-zinc-500/10 text-zinc-500 border-zinc-200",
}

const NOTE_PREVIEW_MAX_LENGTH = 56

function truncateWithEllipsis(value: string, maxLength = NOTE_PREVIEW_MAX_LENGTH): string {
  if (value.length <= maxLength) {
    return value
  }

  return `${value.slice(0, maxLength).trimEnd()}…`
}

function renderInlineIcon(icon: string, name: string): ReactNode {
  if (!icon) {
    return null
  }

  if (icon.startsWith("si:")) {
    const brand = getBrandIcon(icon.slice(3))
    if (brand) {
      const { Icon } = brand
      return <Icon size={12} color="default" />
    }
    return <span className="text-[10px] leading-none">{name.charAt(0).toUpperCase()}</span>
  }

  if (icon.startsWith("http://") || icon.startsWith("https://")) {
    return (
      <img
        src={icon}
        alt={name}
        className="h-3.5 w-3.5 object-contain"
      />
    )
  }

  if (icon.startsWith("assets/")) {
    const assetPath = icon.slice("assets/".length)
    return (
      <img
        src={`/uploads/${assetPath}`}
        alt={name}
        className="h-3.5 w-3.5 object-contain"
      />
    )
  }

  return <span className="text-[10px] leading-none">{icon}</span>
}

export default function SubscriptionCard({
  subscription,
  categoryName,
  currencySymbol,
  displayAmount,
  displayCurrency,
  displayCurrencySymbol,
  showMonthlyAmount = false,
  paymentMethodName,
  paymentMethodIcon,
  onEdit,
  onDelete,
}: SubscriptionCardProps) {
  const { t, i18n } = useTranslation()
  const amountToDisplay = displayAmount ?? subscription.amount
  const currencyToDisplay = displayCurrency ?? subscription.currency
  const symbolToDisplay = displayCurrencySymbol ?? currencySymbol
  const days = subscription.next_billing_date ? daysUntil(subscription.next_billing_date) : null
  const isUpcoming = days !== null && days >= 0 && days < 7
  const trialDays = subscription.trial_enabled && subscription.trial_end_date
    ? daysUntil(subscription.trial_end_date)
    : null
  const trialStartDays = subscription.trial_enabled && subscription.trial_start_date
    ? daysUntil(subscription.trial_start_date)
    : null
  const categoryLabel = categoryName?.trim() || subscription.category
  const rawNotes = subscription.notes.trim()
  const notesPreview = rawNotes ? truncateWithEllipsis(rawNotes) : ""
  const showAnchorDate = Boolean(subscription.billing_anchor_date)
    && (subscription.billing_type === "one_time" || !subscription.next_billing_date)

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
    if (isUpcoming) {
      return t("subscription.card.dueIn", { count: days ?? 0 })
    }
    if ((days ?? 0) < 0) {
      return t("subscription.card.overdue")
    }
    return formatDate(subscription.next_billing_date, i18n.language)
  }

  function renderTrialText(): string | null {
    if (!subscription.trial_enabled) {
      return null
    }
    if ((trialStartDays ?? 0) > 0) {
      return t("subscription.card.trial.startsIn", { count: trialStartDays ?? 0 })
    }
    if (!subscription.trial_end_date) {
      return t("subscription.card.trial.active")
    }
    if ((trialDays ?? 0) >= 0) {
      return t("subscription.card.trial.endsIn", { count: trialDays ?? 0 })
    }
    return t("subscription.card.trial.endedOn", {
      date: formatDate(subscription.trial_end_date, i18n.language),
    })
  }

  function renderReminderText(): string {
    if (!subscription.notify_enabled) {
      return t("subscription.card.reminder.off")
    }
    return t("subscription.card.reminder.on", {
      days: subscription.notify_days_before ?? 0,
    })
  }

  const trialText = renderTrialText()
  const reminderDisabledText = subscription.notify_enabled === false ? renderReminderText() : null
  const dueText = renderDueText()
  const isOverdue = (days ?? 0) < 0
  const dueBadgeClass = isUpcoming
    ? "bg-amber-500/10 text-amber-700 border-amber-200"
    : isOverdue
      ? "bg-destructive/10 text-destructive border-destructive/30"
      : "bg-zinc-500/10 text-zinc-600 border-zinc-200"
  const holdingDays = subscription.billing_type === "one_time" && subscription.billing_anchor_date
    ? (() => {
      const anchorDate = new Date(subscription.billing_anchor_date)
      if (Number.isNaN(anchorDate.getTime())) {
        return null
      }
      return Math.max(1, -daysUntil(subscription.billing_anchor_date))
    })()
    : null
  const holdingCostText = holdingDays
    ? t("subscription.card.holdingCost", {
      amount: formatCurrencyWithSymbol(
        amountToDisplay / holdingDays,
        currencyToDisplay,
        symbolToDisplay,
        i18n.language
      ),
    })
    : null
  const showHoldingCostBadge = subscription.billing_type === "one_time" && Boolean(holdingCostText)
  const secondaryBadgeText = showHoldingCostBadge && holdingCostText ? holdingCostText : dueText
  const secondaryBadgeClass = showHoldingCostBadge
    ? "bg-zinc-500/10 text-zinc-600 border-zinc-200"
    : dueBadgeClass
  const secondaryBadgeTitle = secondaryBadgeText
  const anchorDateText = showAnchorDate && subscription.billing_anchor_date
    ? t("subscription.card.anchorDate", {
      date: formatDate(subscription.billing_anchor_date, i18n.language),
    })
    : null

  return (
    <Card className="group py-3 transition-all hover:shadow-md">
      <CardContent className="flex items-start gap-3 px-4 py-1.5">
        <div
          className="h-10 w-10 shrink-0 rounded-lg flex items-center justify-center overflow-hidden"
        >
          {renderIcon(subscription.icon, subscription.name)}
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h3 className="truncate font-medium">{subscription.name}</h3>
            {subscription.url && (
              <a
                href={subscription.url}
                target="_blank"
                rel="noopener noreferrer"
                className="text-muted-foreground hover:text-foreground"
              >
                <ExternalLink className="size-4" />
              </a>
            )}
          </div>
          <div className="mt-0.5 flex items-center gap-2 text-sm text-muted-foreground">
            {categoryLabel && <span>{categoryLabel}</span>}
            {categoryLabel && paymentMethodName && <span>·</span>}
            {paymentMethodName && (
              <span className="inline-flex items-center gap-1 min-w-0">
                {paymentMethodIcon && (
                  <span className="inline-flex size-4 shrink-0 items-center justify-center rounded-sm bg-muted/50">
                    {renderInlineIcon(paymentMethodIcon, paymentMethodName)}
                  </span>
                )}
                <span className="truncate">{paymentMethodName}</span>
              </span>
            )}
            {reminderDisabledText && (categoryLabel || paymentMethodName) && <span>·</span>}
            {reminderDisabledText && (
              <span
                className="inline-flex size-4 shrink-0 items-center justify-center text-muted-foreground"
                title={reminderDisabledText}
                aria-label={reminderDisabledText}
              >
                <BellOff className="size-3.5" />
              </span>
            )}
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-1">
            {trialText && (
              <Badge variant="outline" className="bg-sky-500/10 text-sky-700 border-sky-200">
                {trialText}
              </Badge>
            )}
            {anchorDateText && (
              <Badge variant="outline" className="bg-zinc-500/10 text-zinc-600 border-zinc-200">
                {anchorDateText}
              </Badge>
            )}
          </div>
          {notesPreview && (
            <p className="mt-1 truncate text-xs text-muted-foreground" title={rawNotes}>
              {t("subscription.card.notes", { content: notesPreview })}
            </p>
          )}
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1">
          <div className="flex max-w-[14rem] items-baseline gap-1 text-right">
            <p className="font-semibold tabular-nums whitespace-nowrap">
              {formatCurrencyWithSymbol(
                amountToDisplay,
                currencyToDisplay,
                symbolToDisplay,
                i18n.language
              )}
            </p>
            <span className="text-xs text-muted-foreground">/</span>
            <p className="truncate text-xs text-muted-foreground" title={renderBillingLabel()}>
              {renderBillingLabel()}
            </p>
          </div>
          <div className="mt-0.5 flex max-w-[14rem] justify-end">
            <Badge
              variant="outline"
              className={`max-w-[12rem] truncate ${secondaryBadgeClass}`}
              title={secondaryBadgeTitle}
            >
              {secondaryBadgeText}
            </Badge>
          </div>
          <div className="flex max-w-[14rem] flex-wrap justify-end gap-1">
            <Badge variant="outline" className={statusStyles[subscription.enabled ? "enabled" : "disabled"] || ""}>
              {t(`subscription.card.status.${subscription.enabled ? "enabled" : "disabled"}`)}
            </Badge>
          </div>
        </div>

        <div className="flex self-center flex-col items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
          <Button variant="ghost" size="icon-sm" onClick={() => onEdit(subscription)}>
            <Pencil className="size-3.5" />
          </Button>
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-destructive hover:text-destructive"
            onClick={() => onDelete(subscription.id)}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
