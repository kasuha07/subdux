import { type CSSProperties, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import type { TFunction } from "i18next"
import {
  Bell,
  CalendarDays,
  CircleDollarSign,
  ExternalLink,
  History,
  Pencil,
  ReceiptText,
  X,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { formatCurrencyWithSymbol, formatDate } from "@/lib/utils"
import type {
  Subscription,
  SubscriptionDetail,
  SubscriptionDetailEvent,
  SubscriptionDetailPriceHistoryItem,
} from "@/types"
import {
  getSubscriptionEndsAt,
  getSubscriptionRenewalMode,
} from "@/features/subscriptions/subscription-lifecycle"
import {
  getCachedSubscriptionDetail,
  loadSubscriptionDetail,
} from "./subscription-detail-cache"

interface Props {
  open: boolean
  subscription: Subscription | null
  categoryName?: string
  currencySymbol?: string
  paymentMethodName?: string
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
  categoryName,
  currencySymbol,
  paymentMethodName,
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
    const cached = getCachedSubscriptionDetail(subscription.id)
    setDetail(cached)
    setLoading(!cached)
    setError("")
    loadSubscriptionDetail(subscription.id)
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
      <DialogContent
        className="fixed top-0 right-0 left-auto flex h-dvh max-h-dvh w-full max-w-xl translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-r-0 p-0 duration-300 sm:max-w-xl data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right data-[state=open]:zoom-in-100 data-[state=closed]:zoom-out-100"
        showCloseButton={false}
      >
        <DialogHeader className="detail-drawer-stage flex-row items-start justify-between gap-3 border-b px-5 pt-5 pb-4 text-left sm:px-6">
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
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={handleEdit}
              disabled={!activeSubscription}
              className="shrink-0"
            >
              <Pencil className="size-4" />
              {t("subscription.detail.edit")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              className="-mr-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
              asChild
            >
              <DialogClose aria-label={t("common.close")}>
                <X />
              </DialogClose>
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
            ) : !detail && activeSubscription ? (
              <OverviewFallback subscription={activeSubscription} />
            ) : detail && activeSubscription ? (
              <>
                <Overview
                  detail={detail}
                  formatAmount={formatAmount}
                  language={i18n.language}
                />
                <DetailInfoPanel
                  detail={detail}
                  categoryName={categoryName}
                  paymentMethodName={paymentMethodName}
                  formatAmount={formatAmount}
                  language={i18n.language}
                />

                <Tabs defaultValue="timeline" className="detail-drawer-stage gap-4" style={animationDelay(120)}>
                  <div className="w-full overflow-x-auto pb-1">
                    <TabsList className="w-max min-w-max">
                      <TabsTrigger value="timeline" className="flex-none gap-2">
                        <History className="size-4" />
                        {t("subscription.detail.tabs.timeline")}
                        <span className="text-xs text-muted-foreground">{tabsSummary.timeline}</span>
                      </TabsTrigger>
                      <TabsTrigger value="prices" className="flex-none gap-2">
                        <CircleDollarSign className="size-4" />
                        {t("subscription.detail.tabs.prices")}
                        <span className="text-xs text-muted-foreground">{tabsSummary.prices}</span>
                      </TabsTrigger>
                      <TabsTrigger value="notifications" className="flex-none gap-2">
                        <Bell className="size-4" />
                        {t("subscription.detail.tabs.notifications")}
                        <span className="text-xs text-muted-foreground">{tabsSummary.notifications}</span>
                      </TabsTrigger>
                      <TabsTrigger value="charges" className="flex-none gap-2">
                        <ReceiptText className="size-4" />
                        {t("subscription.detail.tabs.charges")}
                        <span className="text-xs text-muted-foreground">{tabsSummary.charges}</span>
                      </TabsTrigger>
                    </TabsList>
                  </div>

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

function OverviewFallback({ subscription }: { subscription: Subscription }) {
  const { t, i18n } = useTranslation()
  const status = subscription.status || "active"
  const renewalMode = getSubscriptionRenewalMode(subscription)
  const periodEndDate = getSubscriptionEndsAt(subscription)
  const isEnding = renewalMode === "cancel_at_period_end" && periodEndDate
  const nextSummaryDate = isEnding ? periodEndDate : subscription.next_billing_date

  return (
    <div className="grid gap-3 sm:grid-cols-3">
      <SummaryTile
        label={isEnding ? t("subscription.detail.summary.periodEnd") : t("subscription.detail.summary.nextCharge")}
        value={nextSummaryDate ? formatDate(nextSummaryDate, i18n.language) : t("subscription.detail.empty.none")}
        detail={isEnding ? t("subscription.detail.summary.endingAtPeriodEnd") : t("subscription.detail.empty.noUpcomingCharges")}
        delay={0}
      />
      <SummaryTile
        label={t("subscription.detail.summary.lifecycle")}
        value={t(`subscription.card.status.${status}`)}
        detail={renewalMode ? t(`subscription.card.renewalMode.${renewalMode}`) : ""}
        delay={40}
      />
      <div
        className="detail-drawer-stage rounded-lg border bg-muted/25 p-3"
        style={animationDelay(80)}
      >
        <p className="text-xs font-medium text-muted-foreground">
          {t("subscription.detail.summary.latestActivity")}
        </p>
        <Skeleton className="mt-2 h-4 w-24 rounded-md" />
        <Skeleton className="mt-2 h-3 w-32 rounded-md" />
      </div>
    </div>
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
  const lifecycleDetail = sub.status === "ended" && sub.ends_at
    ? t("subscription.card.endedOn", { date: formatDate(sub.ends_at, language) })
    : sub.renewal_mode
      ? t(`subscription.card.renewalMode.${sub.renewal_mode}`)
      : ""

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
        detail={lifecycleDetail}
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

function DetailInfoPanel({
  detail,
  categoryName,
  paymentMethodName,
  formatAmount,
  language,
}: {
  detail: SubscriptionDetail
  categoryName?: string
  paymentMethodName?: string
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  const sub = detail.subscription
  const empty = t("subscription.detail.empty.none")
  const resolvedCategory = categoryName?.trim() || sub.category?.trim() || empty
  const resolvedPaymentMethod = paymentMethodName?.trim() || empty
  const formattedUrl = sub.url?.trim()
  const notes = sub.notes?.trim()
  const renewalMode = getSubscriptionRenewalMode(sub)
  const periodEndDate = getSubscriptionEndsAt(sub)
  const isEnding = renewalMode === "cancel_at_period_end" && periodEndDate

  const rows = [
    {
      label: t("subscription.detail.info.amount"),
      value: formatAmount(sub.amount, sub.currency),
    },
    {
      label: t("subscription.detail.info.billingType"),
      value: t(`subscription.form.billingType.${sub.billing_type}`, {
        defaultValue: sub.billing_type || empty,
      }),
    },
    {
      label: t("subscription.detail.info.recurrence"),
      value: formatRecurrenceRule(sub, t),
    },
    {
      label: isEnding
        ? t("subscription.detail.info.periodEndDate")
        : t("subscription.detail.info.nextBillingDate"),
      value: isEnding
        ? formatDate(periodEndDate, language)
        : sub.next_billing_date
          ? formatDate(sub.next_billing_date, language)
          : empty,
    },
    {
      label: t("subscription.detail.info.status"),
      value: t(`subscription.card.status.${sub.status}`),
    },
    {
      label: t("subscription.detail.info.renewalMode"),
      value: t(`subscription.card.renewalMode.${sub.renewal_mode}`),
    },
    {
      label: t("subscription.detail.info.endsAt"),
      value: sub.ends_at ? formatDate(sub.ends_at, language) : empty,
    },
    {
      label: t("subscription.detail.info.category"),
      value: resolvedCategory,
    },
    {
      label: t("subscription.detail.info.paymentMethod"),
      value: resolvedPaymentMethod,
    },
    {
      label: t("subscription.detail.info.notification"),
      value: formatNotificationSetting(sub, t),
    },
    {
      label: t("subscription.detail.info.createdAt"),
      value: sub.created_at ? formatDate(sub.created_at, language) : empty,
    },
    {
      label: t("subscription.detail.info.updatedAt"),
      value: sub.updated_at ? formatDate(sub.updated_at, language) : empty,
    },
  ]

  return (
    <section
      className="detail-drawer-stage overflow-hidden rounded-lg border bg-background"
      style={animationDelay(80)}
    >
      <div className="border-b bg-muted/25 px-3 py-2">
        <p className="text-sm font-medium">{t("subscription.detail.info.title")}</p>
      </div>
      <dl className="grid sm:grid-cols-2">
        {rows.map((row) => (
          <DetailInfoItem key={row.label} label={row.label} value={row.value} />
        ))}
        <DetailInfoItem
          label={t("subscription.detail.info.url")}
          value={formattedUrl || empty}
          href={formattedUrl}
          wide
        />
        <DetailInfoItem
          label={t("subscription.detail.info.notes")}
          value={notes || empty}
          wide
        />
      </dl>
    </section>
  )
}

function DetailInfoItem({
  href,
  label,
  value,
  wide = false,
}: {
  href?: string
  label: string
  value: string
  wide?: boolean
}) {
  return (
    <div className={`min-w-0 border-b px-3 py-2.5 last:border-b-0 ${wide ? "sm:col-span-2" : ""}`}>
      <dt className="text-xs font-medium text-muted-foreground">{label}</dt>
      <dd className="mt-1 min-w-0 text-sm">
        {href ? (
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex max-w-full items-center gap-1 text-primary hover:underline"
            title={value}
          >
            <span className="truncate">{value}</span>
            <ExternalLink className="size-3.5 shrink-0" />
          </a>
        ) : (
          <span className="break-words" title={value}>{value}</span>
        )}
      </dd>
    </div>
  )
}

function SummaryTile({ label, value, detail, delay }: { label: string, value: string, detail: string, delay: number }) {
  return (
    <div
      className="detail-drawer-stage rounded-lg border bg-muted/25 p-3"
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
        <TimelineItem
          key={item.id}
          item={item}
          index={index}
          formatAmount={formatAmount}
          language={language}
        />
      ))}
    </div>
  )
}

function TimelineItem({
  item,
  index,
  formatAmount,
  language,
}: {
  item: SubscriptionDetailEvent
  index: number
  formatAmount: (amount: number, currency?: string) => string
  language: string
}) {
  const { t } = useTranslation()
  const changeRows = eventChangeRows(item, formatAmount, language, t)

  return (
    <div
      className="detail-drawer-row grid gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-start"
      style={rowAnimationDelay(index)}
    >
      <div className="min-w-0">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <Badge variant="outline" className={eventBadgeStyles[item.type] || ""}>
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
      {changeRows.length > 0 ? (
        <div className="grid gap-1.5 rounded-md bg-muted/30 p-2 sm:col-span-2">
          {changeRows.map((row) => (
            <div key={row.label} className="grid gap-1 text-xs sm:grid-cols-[8rem_minmax(0,1fr)]">
              <p className="font-medium text-muted-foreground">{row.label}</p>
              <p className="min-w-0 break-words">
                <span>{row.previous}</span>
                <span className="px-1.5 text-muted-foreground">-&gt;</span>
                <span>{row.next}</span>
              </p>
            </div>
          ))}
        </div>
      ) : null}
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
            className="detail-drawer-row grid gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
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
          className="detail-drawer-row grid gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
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
      <div className="detail-drawer-stage flex flex-wrap items-center justify-between gap-2 rounded-lg border bg-muted/25 p-3">
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
            className="detail-drawer-row flex items-center justify-between gap-3 p-3"
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

function formatRecurrenceRule(sub: Subscription, t: TFunction): string {
  const empty = t("subscription.detail.empty.none")
  if (sub.billing_type !== "recurring") {
    return t(`subscription.form.billingType.${sub.billing_type}`, {
      defaultValue: sub.billing_type || empty,
    })
  }

  if (sub.recurrence_type === "interval" && sub.interval_count && sub.interval_unit) {
    return t(`subscription.card.recurrence.interval.${sub.interval_unit}`, {
      count: sub.interval_count,
    })
  }
  if (sub.recurrence_type === "monthly_date" && sub.monthly_day) {
    return t("subscription.card.recurrence.monthlyDate", { day: sub.monthly_day })
  }
  if (sub.recurrence_type === "yearly_date" && sub.yearly_month && sub.yearly_day) {
    return t("subscription.card.recurrence.yearlyDate", {
      month: sub.yearly_month,
      day: sub.yearly_day,
    })
  }
  return empty
}

function formatNotificationSetting(sub: Subscription, t: TFunction): string {
  if (sub.notify_enabled === null) {
    return t("subscription.detail.info.notificationDefault")
  }
  if (sub.notify_enabled === false) {
    return t("subscription.detail.info.notificationDisabled")
  }
  if (sub.notify_days_before !== null) {
    return t("subscription.detail.info.notificationEnabledWithDays", {
      days: sub.notify_days_before,
    })
  }
  return t("subscription.detail.info.notificationEnabled")
}

function eventChangeRows(
  item: SubscriptionDetailEvent,
  formatAmount: (amount: number, currency?: string) => string,
  language: string,
  t: TFunction
): Array<{ label: string, previous: string, next: string }> {
  const fields = item.changed_fields.length > 0 ? item.changed_fields : ["amount"]
  if (item.type === "created") {
    return ["amount", "monthly_amount", "next_billing_date", "status", "renewal_mode", "category", "payment_method"]
      .map((field) => eventChangeRow(field, item, formatAmount, language, t, "created"))
      .filter((row): row is { label: string, previous: string, next: string } => row !== null)
  }
  if (item.type === "deleted") {
    return ["amount", "monthly_amount", "next_billing_date", "status", "renewal_mode", "category", "payment_method"]
      .map((field) => eventChangeRow(field, item, formatAmount, language, t, "deleted"))
      .filter((row): row is { label: string, previous: string, next: string } => row !== null)
  }
  return fields
    .map((field) => eventChangeRow(field, item, formatAmount, language, t))
    .filter((row): row is { label: string, previous: string, next: string } => row !== null)
}

function eventChangeRow(
  field: string,
  item: SubscriptionDetailEvent,
  formatAmount: (amount: number, currency?: string) => string,
  language: string,
  t: TFunction,
  mode?: "created" | "deleted"
): { label: string, previous: string, next: string } | null {
  const empty = t("subscription.detail.empty.none")
  const label = eventFieldLabel(field, t)

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
      lifecycleStatusLabel(item.previous_status, t),
      lifecycleStatusLabel(item.new_status, t)
    )
    return values ? { label, ...values } : null
  }
  if (field === "renewal_mode") {
    const values = valueForText(
      renewalModeLabel(item.previous_renewal_mode, t),
      renewalModeLabel(item.new_renewal_mode, t)
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

function lifecycleStatusLabel(value: string, t: TFunction): string {
  if (!value) {
    return ""
  }
  return t(`subscription.card.status.${value}`, { defaultValue: value })
}

function renewalModeLabel(value: string, t: TFunction): string {
  if (!value) {
    return ""
  }
  return t(`subscription.card.renewalMode.${value}`, { defaultValue: value })
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
