import type { Subscription } from "@/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { formatCurrency, daysUntil, formatDate } from "@/lib/utils"
import { Pencil, Trash2, ExternalLink } from "lucide-react"

interface SubscriptionCardProps {
  subscription: Subscription
  onEdit: (sub: Subscription) => void
  onDelete: (id: number) => void
}

const cycleLabels: Record<string, string> = {
  weekly: "/ week",
  monthly: "/ mo",
  yearly: "/ yr",
}

const statusStyles: Record<string, string> = {
  active: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  paused: "bg-amber-500/10 text-amber-700 border-amber-200",
  cancelled: "bg-zinc-500/10 text-zinc-500 border-zinc-200",
}

export default function SubscriptionCard({ subscription, onEdit, onDelete }: SubscriptionCardProps) {
  const days = daysUntil(subscription.next_billing_date)
  const isUpcoming = days >= 0 && days <= 3

  return (
    <Card className="group transition-all hover:shadow-md">
      <CardContent className="flex items-center gap-4 p-4">
        <div
          className="h-10 w-10 shrink-0 rounded-lg flex items-center justify-center text-lg font-semibold text-white"
          style={{ backgroundColor: subscription.color || "#18181b" }}
        >
          {subscription.icon || subscription.name.charAt(0).toUpperCase()}
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
            {subscription.category && <span>·</span>}
            <span>
              {isUpcoming ? (
                <span className="text-amber-600 font-medium">Due in {days}d</span>
              ) : days < 0 ? (
                <span className="text-destructive font-medium">Overdue</span>
              ) : (
                formatDate(subscription.next_billing_date)
              )}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-3 shrink-0">
          <div className="text-right">
            <p className="font-semibold tabular-nums">
              {formatCurrency(subscription.amount, subscription.currency)}
            </p>
            <p className="text-xs text-muted-foreground">
              {cycleLabels[subscription.billing_cycle] || subscription.billing_cycle}
            </p>
          </div>
          <Badge variant="outline" className={statusStyles[subscription.status] || ""}>
            {subscription.status}
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
