import type { ReactNode } from "react"
import type { Subscription } from "@/types"
import { useTranslation } from "react-i18next"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { formatCurrencyWithSymbol, daysUntil, formatDate } from "@/lib/utils"
import { Pencil, Trash2, ExternalLink } from "lucide-react"
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
  paymentMethodName?: string
  onEdit: (sub: Subscription) => void
  onDelete: (id: number) => void
}

const statusStyles: Record<string, string> = {
  enabled: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  disabled: "bg-zinc-500/10 text-zinc-500 border-zinc-200",
}

export default function SubscriptionCard({ subscription, categoryName, currencySymbol, paymentMethodName, onEdit, onDelete }: SubscriptionCardProps) {
  const { t, i18n } = useTranslation()
  const days = subscription.next_billing_date ? daysUntil(subscription.next_billing_date) : null
  const isUpcoming = days !== null && days >= 0 && days <= 3
  const categoryLabel = categoryName?.trim() || subscription.category

  function renderBillingLabel(): string {
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

  return (
    <Card className="group transition-all hover:shadow-md">
      <CardContent className="flex items-center gap-4 p-4">
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
                <ExternalLink className="size-3" />
              </a>
            )}
          </div>
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            {categoryLabel && <span>{categoryLabel}</span>}
            {categoryLabel && paymentMethodName && <span>·</span>}
            {paymentMethodName && <span>{paymentMethodName}</span>}
            {(categoryLabel || paymentMethodName) && <span>·</span>}
            <span>
              {isUpcoming ? (
                <span className="text-amber-600 font-medium">{renderDueText()}</span>
              ) : (days ?? 0) < 0 ? (
                <span className="text-destructive font-medium">{renderDueText()}</span>
              ) : (
                renderDueText()
              )}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <div className="text-right">
            <p className="font-semibold tabular-nums">
              {formatCurrencyWithSymbol(
                subscription.amount,
                subscription.currency,
                currencySymbol,
                i18n.language
              )}
            </p>
            <p className="text-xs text-muted-foreground">
              {renderBillingLabel()}
            </p>
          </div>
          <Badge variant="outline" className={statusStyles[subscription.enabled ? "enabled" : "disabled"] || ""}>
            {t(`subscription.card.status.${subscription.enabled ? "enabled" : "disabled"}`)}
          </Badge>
        </div>

        <div className="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
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
