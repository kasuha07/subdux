import { useState } from "react"
import { useTranslation } from "react-i18next"
import {
  ArrowUpRight,
  CalendarDays,
  History,
  ReceiptText,
  TrendingUp,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { SubscriptionIcon } from "@/features/subscriptions/subscription-icon"
import {
  formatSubscriptionEventAmountChange,
  reportRenewalModeLabel,
  subscriptionEventFieldLabel,
  subscriptionEventTypeLabel,
} from "@/lib/subscription-event-formatters"
import { formatDate } from "@/lib/utils"
import type {
  AnalyticsReport,
  ReportAnnualGrowthItem,
  ReportBreakdownItem,
  ReportPriceIncrease,
  ReportSubscriptionEvent,
  ReportSubscriptionSpend,
  ReportUpcomingRenewal,
} from "@/types"

export function ReportsSkeleton() {
  return (
    <div className="page-loading-enter space-y-6">
      <div className="grid grid-cols-2 gap-2 sm:gap-3 lg:grid-cols-4">
        {Array.from({ length: 8 }).map((_, index) => (
          <Card key={index}>
            <CardContent className="space-y-1.5 px-3 py-2 sm:space-y-2.5 sm:px-4 sm:py-3">
              <Skeleton className="size-7 rounded-lg sm:size-9 sm:rounded-xl" />
              <Skeleton className="h-3 w-20 sm:h-4 sm:w-24" />
              <Skeleton className="h-6 w-24 sm:h-8 sm:w-32" />
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardContent className="space-y-3 p-5">
          {Array.from({ length: 8 }).map((_, index) => (
            <Skeleton key={index} className="h-8 w-full" />
          ))}
        </CardContent>
      </Card>
    </div>
  )
}

export function EmptyState({ title, description }: { title: string, description: string }) {
  return (
    <div className="rounded-lg border border-dashed p-8 text-center">
      <p className="font-medium">{title}</p>
      <p className="mt-1 text-sm text-muted-foreground">{description}</p>
    </div>
  )
}

export function KpiCard({
  detail,
  icon: Icon,
  label,
  value,
}: {
  detail: string
  icon: LucideIcon
  label: string
  value: string
}) {
  return (
    <Card className="w-40 shrink-0 snap-start py-0 sm:w-auto sm:min-w-0 sm:shrink">
      <CardContent className="px-2.5 py-3 sm:px-4 sm:py-4">
        <div className="flex items-start justify-between gap-2 sm:gap-3">
          <div className="min-w-0">
            <p className="truncate text-xs font-medium text-muted-foreground sm:text-sm">{label}</p>
            <p className="mt-0.5 truncate text-base font-semibold tabular-nums sm:mt-2 sm:text-2xl">{value}</p>
          </div>
          <div className="rounded-lg bg-primary/10 p-1 text-primary sm:rounded-xl sm:p-2">
            <Icon className="size-3 sm:size-4" />
          </div>
        </div>
        <p className="mt-1 truncate text-[11px] text-muted-foreground sm:mt-2 sm:text-xs">{detail}</p>
      </CardContent>
    </Card>
  )
}

export function MonthlyForecastPanel({
  formatAmount,
  items,
  maxAmount,
}: {
  formatAmount: (amount: number) => string
  items: AnalyticsReport["monthly_forecast"]
  maxAmount: number
}) {
  const { t, i18n } = useTranslation()
  const [activeIndex, setActiveIndex] = useState<number | null>(null)
  const [focusedIndex, setFocusedIndex] = useState<number | null>(null)
  const chartWidth = 760
  const chartHeight = 240
  const chartPadding = { top: 24, right: 24, bottom: 34, left: 24 }
  const chartBottom = chartHeight - chartPadding.bottom
  const chartSpanX = chartWidth - chartPadding.left - chartPadding.right
  const chartSpanY = chartBottom - chartPadding.top
  const pointCount = Math.max(1, items.length - 1)
  const points = items.map((item, index) => {
    const x = chartPadding.left + (chartSpanX * index) / pointCount
    const y = chartBottom - (item.amount_due / maxAmount) * chartSpanY
    return { item, x, y }
  })
  const linePath = smoothLinePath(points)
  const areaPath = points.length > 1
    ? `${linePath} L ${points[points.length - 1].x.toFixed(1)} ${chartBottom} L ${points[0].x.toFixed(1)} ${chartBottom} Z`
    : ""
  const yTicks = [0, 0.25, 0.5, 0.75, 1]
  const selectedIndex = activeIndex ?? focusedIndex
  const activePoint = selectedIndex === null ? null : points[selectedIndex]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <CalendarDays className="size-4" />
          {t("reports.forecast.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.empty.title")} description={t("reports.empty.description")} />
        ) : (
          <div className="relative rounded-lg border bg-card p-3 sm:p-4">
            <div className="pointer-events-none absolute inset-0 rounded-lg bg-linear-to-b from-muted/25 to-transparent" />
            <div
              className="relative h-[240px]"
              onPointerMove={(event) => {
                if (event.pointerType === "touch") return
                const bounds = event.currentTarget.getBoundingClientRect()
                if (bounds.width === 0) return
                const x = ((event.clientX - bounds.left) / bounds.width) * chartWidth
                const index = Math.round(((x - chartPadding.left) / chartSpanX) * (items.length - 1))
                setActiveIndex(Math.max(0, Math.min(items.length - 1, index)))
              }}
              onPointerLeave={() => setActiveIndex(null)}
              onPointerCancel={() => setActiveIndex(null)}
              onKeyDown={(event) => {
                if (event.key === "Escape") {
                  setActiveIndex(null)
                  setFocusedIndex(null)
                }
              }}
            >
              {activePoint ? (
                <div
                  role="status"
                  className="reports-forecast-tooltip pointer-events-none absolute z-20 w-44 max-w-full rounded-md border bg-popover px-3 py-2 text-xs text-popover-foreground shadow-lg"
                  style={{
                    left: `clamp(0px, calc(${(activePoint.x / chartWidth) * 100}% - 5.5rem), max(0px, calc(100% - 11rem)))`,
                    top: `${(activePoint.y / chartHeight) * 100}%`,
                    transform: activePoint.y < chartHeight / 2
                      ? "translateY(16px)"
                      : "translateY(calc(-100% - 16px))",
                  }}
                >
                  <p className="font-medium">{formatMonth(activePoint.item.month, i18n.language)}</p>
                  <p className="mt-1 break-words font-semibold tabular-nums">{formatAmount(activePoint.item.amount_due)}</p>
                  <p className="mt-1 text-muted-foreground">
                    {t("reports.forecast.occurrences", { count: activePoint.item.occurrence_count })}
                  </p>
                </div>
              ) : null}
              <svg
                viewBox={`0 0 ${chartWidth} ${chartHeight}`}
                role="img"
                aria-label={t("reports.forecast.title")}
                className="absolute inset-0 size-full"
                preserveAspectRatio="none"
              >
                <defs>
                  <linearGradient id="forecast-area-fill" x1="0" x2="0" y1="0" y2="1">
                    <stop offset="0%" stopColor="var(--primary)" stopOpacity="0.18" />
                    <stop offset="100%" stopColor="var(--primary)" stopOpacity="0" />
                  </linearGradient>
                </defs>
                {yTicks.map((tick) => {
                  const y = chartBottom - tick * chartSpanY
                  return (
                    <line
                      key={tick}
                      x1={chartPadding.left}
                      y1={y}
                      x2={chartWidth - chartPadding.right}
                      y2={y}
                      className="stroke-border"
                      strokeDasharray={tick === 0 ? "0" : "3 8"}
                      vectorEffect="non-scaling-stroke"
                    />
                  )
                })}
                <g className="reports-forecast-reveal">
                  {areaPath ? (
                    <path d={areaPath} fill="url(#forecast-area-fill)" />
                  ) : null}
                  {linePath ? (
                    <path
                      d={linePath}
                      fill="none"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="stroke-primary"
                      strokeWidth="3"
                      vectorEffect="non-scaling-stroke"
                    />
                  ) : null}
                </g>
                {activePoint ? (
                  <line
                    x1="0"
                    y1={chartPadding.top}
                    x2="0"
                    y2={chartBottom}
                    className="reports-forecast-guide stroke-primary/30"
                    style={{ transform: `translateX(${activePoint.x}px)` }}
                    strokeDasharray="4 6"
                    vectorEffect="non-scaling-stroke"
                  />
                ) : null}
              </svg>
              {points.map(({ item, x, y }, index) => {
                const isActive = selectedIndex === index
                const showLabel = index === 0 || index === points.length - 1 || index % 3 === 0
                const isEdgeLabel = index === 0 || index === points.length - 1
                const labelAlignmentClass = index === 0
                  ? "text-left"
                  : index === points.length - 1
                    ? "text-right"
                    : "-translate-x-1/2 text-center"
                const labelPositionStyle = index === points.length - 1
                  ? {
                      right: `${((chartWidth - x) / chartWidth) * 100}%`,
                      top: `${((chartHeight - 18) / chartHeight) * 100}%`,
                    }
                  : {
                      left: `${(x / chartWidth) * 100}%`,
                      top: `${((chartHeight - 18) / chartHeight) * 100}%`,
                    }
                return (
                  <div key={item.month}>
                    {showLabel ? (
                      <span
                        className={`pointer-events-none absolute w-max whitespace-nowrap text-[11px] text-muted-foreground ${isEdgeLabel ? "" : "hidden sm:block"} ${labelAlignmentClass}`}
                        style={labelPositionStyle}
                      >
                        {formatMonth(item.month, i18n.language)}
                      </span>
                    ) : null}
                    <button
                      type="button"
                      className="absolute size-8 -translate-x-1/2 -translate-y-1/2 rounded-full border-0 bg-transparent p-0 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                      style={{
                        left: `${(x / chartWidth) * 100}%`,
                        top: `${(y / chartHeight) * 100}%`,
                      }}
                      aria-label={`${formatMonth(item.month, i18n.language)} ${formatAmount(item.amount_due)} ${t("reports.forecast.occurrences", { count: item.occurrence_count })}`}
                      onClick={() => setActiveIndex(index)}
                      onFocus={() => {
                        setActiveIndex(null)
                        setFocusedIndex(index)
                      }}
                      onBlur={() => {
                        setActiveIndex(null)
                        setFocusedIndex(null)
                      }}
                    >
                      <span
                        style={{ animationDelay: `${160 + (x / chartWidth) * 1000}ms` }}
                        className={[
                          "reports-forecast-point absolute left-1/2 top-1/2 block -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-primary bg-card transition-[width,height,box-shadow] duration-150 ease-out motion-reduce:transition-none",
                          isActive ? "size-3.5 shadow-sm ring-4 ring-primary/15" : "size-2.5",
                        ].join(" ")}
                      />
                    </button>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function BreakdownPanel({
  emptyDescription,
  emptyTitle,
  formatAmount,
  icon: Icon,
  items,
  labelForKey,
  title,
}: {
  emptyDescription: string
  emptyTitle: string
  formatAmount: (amount: number) => string
  icon: LucideIcon
  items: ReportBreakdownItem[]
  labelForKey: (item: ReportBreakdownItem) => string
  title: string
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <Icon className="size-4" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={emptyTitle} description={emptyDescription} />
        ) : (
          <div className="space-y-4">
            {items.map((item) => (
              <div key={item.key} className="space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{labelForKey(item)}</p>
                    <p className="text-xs text-muted-foreground">{item.count}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold tabular-nums">{formatAmount(item.monthly_amount)}</p>
                    <p className="text-xs text-muted-foreground">{item.percentage.toFixed(1)}%</p>
                  </div>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-muted">
                  <div
                    className="cycle-progress-fill h-full rounded-full bg-primary"
                    style={{ width: `${Math.max(2, Math.min(100, item.percentage))}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function TopSubscriptionsPanel({
  formatAmount,
  items,
  onOpenSubscription,
}: {
  formatAmount: (amount: number) => string
  items: ReportSubscriptionSpend[]
  onOpenSubscription?: (id: number) => void
}) {
  const { t, i18n } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <TrendingUp className="size-4" />
          {t("reports.topSubscriptions.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.empty.title")} description={t("reports.empty.description")} />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <button
                key={item.id}
                type="button"
                className="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => onOpenSubscription?.(item.id)}
                disabled={!onOpenSubscription}
              >
                <div className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted">
                  <SubscriptionIcon icon={item.icon} name={item.name} size={22} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex min-w-0 items-center gap-2">
                    <p className="truncate text-sm font-medium">{item.name}</p>
                    <Badge variant="outline">{reportRenewalModeLabel(item.renewal_mode, t)}</Badge>
                  </div>
                  <p className="mt-1 truncate text-xs text-muted-foreground">
                    {item.category || t("reports.categories.none")}
                    {item.next_billing_date ? ` / ${formatDate(item.next_billing_date, i18n.language)}` : ""}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold tabular-nums">{formatAmount(item.monthly_amount)}</p>
                  <p className="text-xs text-muted-foreground">{t("reports.topSubscriptions.monthly")}</p>
                </div>
              </button>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function UpcomingRenewalsPanel({
  formatAmount,
  items,
  language,
  onOpenSubscription,
}: {
  formatAmount: (amount: number) => string
  items: ReportUpcomingRenewal[]
  language: string
  onOpenSubscription?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ReceiptText className="size-4" />
          {t("reports.upcoming.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.upcoming.emptyTitle")} description={t("reports.upcoming.emptyDescription")} />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <button
                key={`${item.id}-${item.billing_date}`}
                type="button"
                className="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => onOpenSubscription?.(item.id)}
                disabled={!onOpenSubscription}
              >
                <div className="flex size-9 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted">
                  <SubscriptionIcon icon={item.icon} name={item.name} size={22} />
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium">{item.name}</p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {formatDate(item.billing_date, language)} / {t("reports.upcoming.daysUntil", { count: item.days_until })}
                  </p>
                </div>
                <div className="text-right">
                  <p className="text-sm font-semibold tabular-nums">{formatAmount(item.amount)}</p>
                  <Badge variant="outline" className="mt-1">
                    {reportRenewalModeLabel(item.renewal_mode, t)}
                  </Badge>
                </div>
              </button>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function PriceIncreasesPanel({
  formatAmount,
  items,
  language,
  onOpenSubscription,
}: {
  formatAmount: (amount: number) => string
  items: ReportPriceIncrease[]
  language: string
  onOpenSubscription?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <ArrowUpRight className="size-4" />
          {t("reports.priceIncreases.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.priceIncreases.emptyTitle")} description={t("reports.priceIncreases.emptyDescription")} />
        ) : (
          <div className="space-y-3">
            {items.map((item) => (
              <button
                key={`${item.subscription_id}-${item.changed_at}`}
                type="button"
                className="w-full rounded-lg border p-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => onOpenSubscription?.(item.subscription_id)}
                disabled={!onOpenSubscription}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{item.name}</p>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {formatDate(item.changed_at, language)}
                    </p>
                  </div>
                  <Badge variant="outline" className="shrink-0 tabular-nums">
                    +{item.delta_percentage.toFixed(1)}%
                  </Badge>
                </div>
                <div className="mt-3 grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2 text-sm">
                  <p className="truncate tabular-nums text-muted-foreground">{formatAmount(item.previous_monthly_amount)}</p>
                  <ArrowUpRight className="size-3.5 text-muted-foreground" />
                  <p className="truncate text-right font-semibold tabular-nums">{formatAmount(item.new_monthly_amount)}</p>
                </div>
                <p className="mt-1 text-xs text-muted-foreground">
                  {t("reports.priceIncreases.delta", { amount: formatAmount(item.delta_monthly_amount) })}
                </p>
              </button>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function AnnualGrowthPanel({
  formatAmount,
  items,
  onOpenSubscription,
}: {
  formatAmount: (amount: number) => string
  items: ReportAnnualGrowthItem[]
  onOpenSubscription?: (id: number) => void
}) {
  const { t } = useTranslation()
  const maxDelta = Math.max(1, ...items.map((item) => item.delta_monthly_amount))
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <TrendingUp className="size-4" />
          {t("reports.annualGrowth.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.annualGrowth.emptyTitle")} description={t("reports.annualGrowth.emptyDescription")} />
        ) : (
          <div className="space-y-4">
            {items.map((item) => (
              <button
                key={item.subscription_id}
                type="button"
                className="w-full space-y-2 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                onClick={() => onOpenSubscription?.(item.subscription_id)}
                disabled={!onOpenSubscription}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{item.name}</p>
                    <p className="mt-1 truncate text-xs text-muted-foreground">
                      {t("reports.annualGrowth.fromTo", {
                        from: formatAmount(item.baseline_monthly_amount),
                        to: formatAmount(item.current_monthly_amount),
                      })}
                    </p>
                  </div>
                  <div className="text-right">
                    <p className="text-sm font-semibold tabular-nums">{formatAmount(item.delta_monthly_amount)}</p>
                    <p className="text-xs text-muted-foreground">+{item.delta_percentage.toFixed(1)}%</p>
                  </div>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-muted">
                  <div
                    className="cycle-progress-fill h-full rounded-full bg-primary"
                    style={{ width: `${Math.max(4, Math.min(100, (item.delta_monthly_amount / maxDelta) * 100))}%` }}
                  />
                </div>
              </button>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

export function RecentChangesPanel({
  formatAmount,
  items,
  language,
  onOpenSubscription,
}: {
  formatAmount: (amount: number) => string
  items: ReportSubscriptionEvent[]
  language: string
  onOpenSubscription?: (id: number) => void
}) {
  const { t } = useTranslation()
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2 text-base">
          <History className="size-4" />
          {t("reports.recentChanges.title")}
        </CardTitle>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <EmptyState title={t("reports.recentChanges.emptyTitle")} description={t("reports.recentChanges.emptyDescription")} />
        ) : (
          <div className="divide-y rounded-lg border">
            {items.map((item) => (
              <button
                key={item.id}
                type="button"
                className="grid w-full gap-3 p-3 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center"
                onClick={() => item.subscription_id !== null && onOpenSubscription?.(item.subscription_id)}
                disabled={item.subscription_id === null || !onOpenSubscription}
              >
                <div className="min-w-0">
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <p className="truncate text-sm font-medium">{item.name}</p>
                    <Badge variant="outline">{subscriptionEventTypeLabel(item.type, t)}</Badge>
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {formatDate(item.changed_at, language)}
                    {item.changed_fields.length > 0 ? ` / ${item.changed_fields.map((field) => subscriptionEventFieldLabel(field, t)).join(", ")}` : ""}
                  </p>
                </div>
                <p className="text-left text-sm tabular-nums sm:text-right">
                  {formatSubscriptionEventAmountChange(item, formatAmount)}
                </p>
              </button>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function formatMonth(value: string, locale: string): string {
  const [year, month] = value.split("-").map((part) => Number.parseInt(part, 10))
  if (!year || !month) {
    return value
  }
  return new Date(year, month - 1, 1).toLocaleDateString(locale, {
    month: "short",
    year: "numeric",
  })
}

function smoothLinePath(points: Array<{ x: number, y: number }>): string {
  if (points.length === 0) {
    return ""
  }
  if (points.length === 1) {
    const point = points[0]
    return `M ${point.x.toFixed(1)} ${point.y.toFixed(1)}`
  }

  return points.reduce((path, point, index) => {
    if (index === 0) {
      return `M ${point.x.toFixed(1)} ${point.y.toFixed(1)}`
    }

    const previous = points[index - 1]
    const controlOffset = (point.x - previous.x) * 0.45
    const controlX1 = previous.x + controlOffset
    const controlX2 = point.x - controlOffset

    return [
      path,
      `C ${controlX1.toFixed(1)} ${previous.y.toFixed(1)}`,
      `${controlX2.toFixed(1)} ${point.y.toFixed(1)}`,
      `${point.x.toFixed(1)} ${point.y.toFixed(1)}`,
    ].join(" ")
  }, "")
}
