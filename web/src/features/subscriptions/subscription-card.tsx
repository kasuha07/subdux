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
  if (!icon) {
    return (
      <span className="text-sm font-bold text-white">
        {name.charAt(0).toUpperCase()}
      </span>
    )
  }

  if (icon.startsWith("si:")) {
    const brand = getBrandIcon(icon.slice(3))
    if (brand) {
      const { Icon } = brand
      return <Icon size={20} color="default" />
    }
    return (
      <span className="text-sm font-bold text-white">
        {name.charAt(0).toUpperCase()}
      </span>
    )
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
  currencySymbol?: string
  paymentMethodName?: string
  onEdit: (sub: Subscription) => void
  onDelete: (id: number) => void
}

const statusStyles: Record<string, string> = {
  active: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  paused: "bg-amber-500/10 text-amber-700 border-amber-200",
  cancelled: "bg-zinc-500/10 text-zinc-500 border-zinc-200",
}

export default function SubscriptionCard({ subscription, currencySymbol, paymentMethodName, onEdit, onDelete }: SubscriptionCardProps) {
  const { t, i18n } = useTranslation()
  const days = daysUntil(subscription.next_billing_date)
  const isUpcoming = days >= 0 && days <= 3

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
            {subscription.category && <span>{subscription.category}</span>}
            {subscription.category && paymentMethodName && <span>·</span>}
            {paymentMethodName && <span>{paymentMethodName}</span>}
            {(subscription.category || paymentMethodName) && <span>·</span>}
            <span>
              {isUpcoming ? (
                <span className="text-amber-600 font-medium">{t("subscription.card.dueIn", { count: days })}</span>
              ) : days < 0 ? (
                <span className="text-destructive font-medium">{t("subscription.card.overdue")}</span>
              ) : (
                formatDate(subscription.next_billing_date, i18n.language)
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
              {t(`subscription.card.cycle.${subscription.billing_cycle}`, subscription.billing_cycle)}
            </p>
          </div>
          <Badge variant="outline" className={statusStyles[subscription.status] || ""}>
            {t(`subscription.card.status.${subscription.status}`, subscription.status)}
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
