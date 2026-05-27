import { type CSSProperties, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  Bell,
  CalendarDays,
  CircleDollarSign,
  History,
  Pencil,
  ReceiptText,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api } from "@/lib/api"
import { formatCurrencyWithSymbol, formatDate } from "@/lib/utils"
import type {
  Subscription,
  SubscriptionDetail,
  SubscriptionDetailEvent,
  SubscriptionDetailPriceHistoryItem,
} from "@/types"

interface Props {
  open: boolean
  subscription: Subscription | null
  currencySymbol?: string
  onOpenChange: (open: boolean) => void
  onEdit: (subscription: Subscription) => void
}

const eventBadgeStyles: Record<string, string> = {
  created: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  updated: "bg-sky-500/10 text-sky-700 border-sky-200",
  manual_renewed: "bg-amber-500/10 text-amber-700 border-amber-200",
  deleted: "bg-destructive/10 text-destructive border-destructive/30",
  system_change: "bg-violet-500/10 text-violet-700 border-violet-200",
}

const logStatusStyles: Record<string, string> = {
  sent: "bg-emerald-500/10 text-emerald-700 border-emerald-200",
  failed: "bg-destructive/10 text-destructive border-destructive/30",
}

type DetailAnimationStyle = CSSProperties & {
  "--detail-delay"?: string
  "--detail-row-delay"?: string
}

function animationDelay(delay: number): DetailAnimationStyle {
  return { "--detail-delay": `${delay}ms` }
}

function rowAnimationDelay(index: number): DetailAnimationStyle {
  return { "--detail-row-delay": `${Math.min(index * 32, 192)}ms` }
}

export default function SubscriptionDetailDrawer({
  open,
  subscription,
  currencySymbol,
  onOpenChange,
  onEdit,
}: Props) {
  const { t, i18n } = useTranslation()
  const [detail, setDetail] = useState<SubscriptionDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    if (!open || !subscription) {
      setDetail(null)
      setError("")
      return
    }

    let active = true
    setLoading(true)
    setError("")
    api.get<SubscriptionDetail>(`/subscriptions/${subscription.id}/detail`)
      .then((data) => {
        if (active) {
          setDetail(data)
        }
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : t("subscription.detail.error"))
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [open, subscription, t])

  const activeSubscription = detail?.subscription ?? subscription
  const tabsSummary = useMemo(() => {
    if (!detail) {
      return {
        timeline: 0,
        prices: 0,
        notifications: 0,
        charges: 0,
      }
    }
    return {
      timeline: detail.timeline.length,
      prices: detail.price_history.length,
      notifications: detail.notification_logs.length,
      charges: detail.upcoming_charges.length,
    }
  }, [detail])

  function formatAmount(amount: number, currency = activeSubscription?.currency ?? "USD") {
    const symbol = currency.toUpperCase() === activeSubscription?.currency.toUpperCase()
      ? currencySymbol
      : undefined
    return formatCurrencyWithSymbol(amount, currency, symbol, i18n.language)
  }

  function handleEdit() {
    if (!activeSubscription) {
      return
    }
    onEdit(activeSubscription)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="fixed top-0 right-0 left-auto flex h-dvh max-h-dvh w-full max-w-xl translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-r-0 p-0 duration-300 sm:max-w-xl data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right data-[state=open]:zoom-in-100 data-[state=closed]:zoom-out-100">
        <DialogHeader className="detail-drawer-stage border-b px-5 pt-5 pb-4 sm:px-6">
          <div className="flex items-start justify-between gap-4 pr-8">
            <div className="min-w-0">
              <DialogTitle className="truncate">
                {activeSubscription?.name ?? t("subscription.detail.titleFallback")}
              </DialogTitle>
              <DialogDescription className="sr-only">
                {t("subscription.detail.description")}
              </DialogDescription>
              {activeSubscription ? (
                <p className="mt-1 text-sm text-muted-foreground">
                  {formatAmount(activeSubscription.amount, activeSubscription.currency)}
                </p>
              ) : null}
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={handleEdit}
              disabled={!activeSubscription}
              className="shrink-0 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm"
            >
              <Pencil className="size-4" />
              {t("subscription.detail.edit")}
            </Button>
          </div>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-5 px-5 py-5 sm:px-6">
            {loading ? (
              <DetailSkeleton />
            ) : error ? (
              <EmptyPanel
                title={t("subscription.detail.errorTitle")}
                description={error}
              />
            ) : detail && activeSubscription ? (
              <>
                <Overview
                  detail={detail}
                  formatAmount={formatAmount}
                  language={i18n.language}
                />

                <Tabs defaultValue="timeline" className="detail-drawer-stage gap-4" style={animationDelay(120)}>
                  <TabsList className="grid h-auto w-full grid-cols-2 gap-1 sm:grid-cols-4">
                    <TabsTrigger value="timeline" className="h-9 transition-all duration-200 data-[state=active]:scale-[1.02]">
                      <History className="size-4" />
                      {t("subscription.detail.tabs.timeline")}
                      <span className="text-xs text-muted-foreground">{tabsSummary.timeline}</span>
                    </TabsTrigger>
                    <TabsTrigger value="prices" className="h-9 transition-all duration-200 data-[state=active]:scale-[1.02]">
                      <CircleDollarSign className="size-4" />
                      {t("subscription.detail.tabs.prices")}
                      <span className="text-xs text-muted-foreground">{tabsSummary.prices}</span>
                    </TabsTrigger>
                    <TabsTrigger value="notifications" className="h-9 transition-all duration-200 data-[state=active]:scale-[1.02]">
                      <Bell className="size-4" />
                      {t("subscription.detail.tabs.notifications")}
                      <span className="text-xs text-muted-foreground">{tabsSummary.notifications}</span>
                    </TabsTrigger>
                    <TabsTrigger value="charges" className="h-9 transition-all duration-200 data-[state=active]:scale-[1.02]">
                      <ReceiptText className="size-4" />
                      {t("subscription.detail.tabs.charges")}
                      <span className="text-xs text-muted-foreground">{tabsSummary.charges}</span>
                    </TabsTrigger>
                  </TabsList>

                  <TabsContent value="timeline" className="detail-tabs-content">
                    <TimelinePanel items={detail.timeline} formatAmount={formatAmount} language={i18n.language} />
                  </TabsContent>
                  <TabsContent value="prices" className="detail-tabs-content">
                    <PriceHistoryPanel items={detail.price_history} formatAmount={formatAmount} language={i18n.language} />
                  </TabsContent>
                  <TabsContent value="notifications" className="detail-tabs-content">
                    <NotificationPanel detail={detail} language={i18n.language} />
                  </TabsContent>
                  <TabsContent value="charges" className="detail-tabs-content">
                    <UpcomingChargesPanel detail={detail} formatAmount={formatAmount} language={i18n.language} />
                  </TabsContent>
                </Tabs>
              </>
            ) : null}
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, index) => (
          <Skeleton
            key={index}
            className="detail-drawer-stage h-20 rounded-lg"
            style={animationDelay(index * 40)}
          />
        ))}
      </div>
      <Skeleton className="detail-drawer-stage h-10 rounded-lg" style={animationDelay(120)} />
      <Skeleton className="detail-drawer-stage h-64 rounded-lg" style={animationDelay(160)} />
    </div>
  )
}

function Overview({
  detail,
  formatAmount,
  language,
}: {
  detail: SubscriptionDetail
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  const sub = detail.subscription
  const nextCharge = detail.upcoming_charges[0]
  const latestEvent = detail.timeline[0]
  const latestLog = detail.notification_logs[0]

  return (
    <div className="grid gap-3 sm:grid-cols-3">
      <SummaryTile
        label={t("subscription.detail.summary.nextCharge")}
        value={nextCharge ? formatDate(nextCharge.date, language) : t("subscription.detail.empty.none")}
        detail={nextCharge ? formatAmount(nextCharge.amount, nextCharge.currency) : t("subscription.detail.empty.noUpcomingCharges")}
        delay={0}
      />
      <SummaryTile
        label={t("subscription.detail.summary.lifecycle")}
        value={t(`subscription.card.status.${sub.status}`)}
        detail={sub.renewal_mode ? t(`subscription.card.renewalMode.${sub.renewal_mode}`) : ""}
        delay={40}
      />
      <SummaryTile
        label={t("subscription.detail.summary.latestActivity")}
        value={latestEvent ? formatDate(latestEvent.changed_at, language) : t("subscription.detail.empty.none")}
        detail={latestLog ? t("subscription.detail.summary.lastNotification", { channel: latestLog.channel_type }) : t("subscription.detail.empty.noNotifications")}
        delay={80}
      />
    </div>
  )
}

function SummaryTile({ label, value, detail, delay }: { label: string, value: string, detail: string, delay: number }) {
  return (
    <div
      className="detail-drawer-stage rounded-lg border bg-muted/25 p-3 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/20 hover:bg-muted/35 hover:shadow-sm"
      style={animationDelay(delay)}
    >
      <p className="text-xs font-medium text-muted-foreground">{label}</p>
      <p className="mt-2 truncate text-sm font-semibold" title={value}>{value}</p>
      <p className="mt-1 truncate text-xs text-muted-foreground" title={detail}>{detail}</p>
    </div>
  )
}

function TimelinePanel({
  items,
  formatAmount,
  language,
}: {
  items: SubscriptionDetailEvent[]
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  if (items.length === 0) {
    return <EmptyPanel title={t("subscription.detail.timeline.emptyTitle")} description={t("subscription.detail.timeline.emptyDescription")} />
  }

  return (
    <div className="divide-y rounded-lg border">
      {items.map((item, index) => (
        <div
          key={item.id}
          className="group detail-drawer-row grid gap-3 p-3 transition-colors duration-200 hover:bg-muted/35 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
          style={rowAnimationDelay(index)}
        >
          <div className="min-w-0">
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <Badge variant="outline" className={`transition-transform duration-200 group-hover:scale-[1.03] ${eventBadgeStyles[item.type] || ""}`}>
                {eventTypeLabel(item.type, t)}
              </Badge>
              <p className="truncate text-sm text-muted-foreground">
                {formatDate(item.changed_at, language)}
              </p>
            </div>
            {item.changed_fields.length > 0 ? (
              <p className="mt-1 text-xs text-muted-foreground">
                {item.changed_fields.map((field) => eventFieldLabel(field, t)).join(", ")}
              </p>
            ) : null}
          </div>
          <p className="text-left text-sm tabular-nums sm:text-right">
            {formatEventAmountChange(item, formatAmount)}
          </p>
        </div>
      ))}
    </div>
  )
}

function PriceHistoryPanel({
  items,
  formatAmount,
  language,
}: {
  items: SubscriptionDetailPriceHistoryItem[]
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  if (items.length === 0) {
    return <EmptyPanel title={t("subscription.detail.prices.emptyTitle")} description={t("subscription.detail.prices.emptyDescription")} />
  }

  return (
    <div className="divide-y rounded-lg border">
      {items.map((item, index) => {
        const amounts = priceHistoryDisplayAmounts(item)
        return (
          <div
            key={item.event_id}
            className="detail-drawer-row grid gap-3 p-3 transition-colors duration-200 hover:bg-muted/35 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
            style={rowAnimationDelay(index)}
          >
            <div className="min-w-0">
              <p className="text-sm font-medium">
                {formatAmount(amounts.amount, amounts.currency)}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {formatDate(item.changed_at, language)}
              </p>
            </div>
            <p className="text-left text-sm text-muted-foreground sm:text-right">
              {amounts.previousAmount !== null
                ? t("subscription.detail.prices.from", {
                    amount: formatAmount(amounts.previousAmount, amounts.previousCurrency),
                  })
                : eventTypeLabel(item.type, t)}
            </p>
          </div>
        )
      })}
    </div>
  )
}

function NotificationPanel({ detail, language }: { detail: SubscriptionDetail, language: string }) {
  const { t } = useTranslation()
  if (detail.notification_logs.length === 0) {
    return <EmptyPanel title={t("subscription.detail.notifications.emptyTitle")} description={t("subscription.detail.notifications.emptyDescription")} />
  }

  return (
    <div className="divide-y rounded-lg border">
      {detail.notification_logs.map((log, index) => (
        <div
          key={log.id}
          className="detail-drawer-row grid gap-3 p-3 transition-colors duration-200 hover:bg-muted/35 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
          style={rowAnimationDelay(index)}
        >
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <p className="text-sm font-medium">{channelTypeLabel(log.channel_type)}</p>
              <Badge variant="outline" className={logStatusStyles[log.status] || ""}>
                {log.status === "sent"
                  ? t("subscription.detail.notifications.statusSent")
                  : t("subscription.detail.notifications.statusFailed")}
              </Badge>
            </div>
            {log.error ? <p className="mt-1 truncate text-xs text-destructive" title={log.error}>{log.error}</p> : null}
          </div>
          <div className="text-left text-xs text-muted-foreground sm:text-right">
            <p>{formatDate(log.notify_date, language)}</p>
            <p>{formatDate(log.sent_at, language)}</p>
          </div>
        </div>
      ))}
    </div>
  )
}

function UpcomingChargesPanel({
  detail,
  formatAmount,
  language,
}: {
  detail: SubscriptionDetail
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  if (detail.upcoming_charges.length === 0) {
    return <EmptyPanel title={t("subscription.detail.charges.emptyTitle")} description={t("subscription.detail.charges.emptyDescription")} />
  }

  return (
    <div className="space-y-4">
      <div className="detail-drawer-stage flex flex-wrap items-center justify-between gap-2 rounded-lg border bg-muted/25 p-3 transition-all duration-200 hover:-translate-y-0.5 hover:border-primary/20 hover:bg-muted/35 hover:shadow-sm">
        <div className="min-w-0">
          <p className="text-sm font-medium">{t("subscription.detail.calendar.title")}</p>
          <p className="mt-1 text-xs text-muted-foreground">
            {detail.calendar.next_event_date
              ? t("subscription.detail.calendar.next", { date: formatDate(detail.calendar.next_event_date, language) })
              : t("subscription.detail.calendar.noEvent")}
          </p>
        </div>
        <Button variant="outline" size="sm" asChild>
          <Link to={detail.calendar.path}>
            <CalendarDays className="size-4" />
            {t("subscription.detail.calendar.open")}
          </Link>
        </Button>
      </div>

      <div className="divide-y rounded-lg border">
        {detail.upcoming_charges.map((charge, index) => (
          <div
            key={charge.date}
            className="detail-drawer-row flex items-center justify-between gap-3 p-3 transition-colors duration-200 hover:bg-muted/35"
            style={rowAnimationDelay(index)}
          >
            <div className="min-w-0">
              <p className="text-sm font-medium">{formatDate(charge.date, language)}</p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t(`subscription.card.renewalMode.${charge.renewal_mode}`)}
              </p>
            </div>
            <p className="shrink-0 text-sm font-medium tabular-nums">
              {formatAmount(charge.amount, charge.currency)}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}

function EmptyPanel({ title, description }: { title: string, description: string }) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
    </div>
  )
}

function eventTypeLabel(type: string, t: (key: string) => string): string {
  return t(`reports.recentChanges.types.${type}`)
}

function eventFieldLabel(field: string, t: (key: string) => string): string {
  const translated = t(`reports.recentChanges.fields.${field}`)
  if (translated === `reports.recentChanges.fields.${field}`) {
    return field
  }
  return translated
}

function formatEventAmountChange(
  item: Pick<SubscriptionDetailEvent, "previous_amount" | "new_amount" | "previous_currency" | "new_currency">,
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

function priceHistoryDisplayAmounts(item: SubscriptionDetailPriceHistoryItem): {
  amount: number
  currency: string
  previousAmount: number | null
  previousCurrency: string
} {
  const monthlyAmount = item.monthly_amount
  const previousMonthlyAmount = item.previous_monthly_amount
  const hasMonthlyOnlyPriceChange = monthlyAmount !== null &&
    previousMonthlyAmount !== null &&
    monthlyAmount !== previousMonthlyAmount &&
    (item.previous_amount === null || item.previous_amount === item.amount)

  if (hasMonthlyOnlyPriceChange) {
    return {
      amount: monthlyAmount,
      currency: item.currency,
      previousAmount: previousMonthlyAmount,
      previousCurrency: item.previous_currency || item.currency,
    }
  }

  return {
    amount: item.amount,
    currency: item.currency,
    previousAmount: item.previous_amount,
    previousCurrency: item.previous_currency || item.currency,
  }
}

function channelTypeLabel(value: string): string {
  if (!value) {
    return ""
  }
  return value
    .split(/[_-]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ")
}
