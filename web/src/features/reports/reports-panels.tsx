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
  const chartWidth = 760
  const chartHeight = 240
  const chartPadding = { top: 24, right: 24, bottom: 34, left: 24 }
  const chartBottom = chartHeight - chartPadding.bottom
  const chartSpanX = chartWidth - chartPadding.left - chartPadding.right
  const chartSpanY = chartBottom - chartPadding.top
  const yTicks = [0, 0.25, 0.5, 0.75, 1]

  const skeletonNormValues = [0.42, 0.55, 0.48, 0.62, 0.56, 0.70, 0.65, 0.80, 0.72, 0.85, 0.78, 0.88]
  const skeletonPoints = skeletonNormValues.map((norm, index) => ({
    x: chartPadding.left + (chartSpanX * index) / 11,
    y: chartBottom - norm * chartSpanY,
  }))
  const skeletonLinePath = smoothLinePath(skeletonPoints)
  const skeletonAreaPath = `${skeletonLinePath} L ${skeletonPoints[skeletonPoints.length - 1].x.toFixed(1)} ${chartBottom} L ${skeletonPoints[0].x.toFixed(1)} ${chartBottom} Z`

  return (
    <div className="page-loading-enter reports-skeleton space-y-6" role="status" aria-label="Loading reports">
      {/* 1. KPI Cards */}
      <section className="-mx-4 overflow-x-auto pb-1 sm:mx-0 sm:overflow-visible sm:pb-0">
        <div className="flex snap-x gap-2 pl-4 sm:grid sm:grid-cols-2 sm:gap-3 sm:px-0 lg:grid-cols-4">
          {Array.from({ length: 8 }).map((_, index) => (
            <Card
              key={index}
              className="reports-skeleton-shimmer subscription-card-enter w-40 shrink-0 snap-start py-0 sm:w-auto sm:min-w-0 sm:shrink"
              style={{ "--card-delay": `${index * 35}ms` } as React.CSSProperties}
            >
              <CardContent className="px-2.5 py-3 sm:px-4 sm:py-4">
                <div className="flex items-start justify-between gap-2 sm:gap-3">
                  <div className="min-w-0 flex-1 space-y-1.5 sm:space-y-2">
                    <Skeleton className="h-3 w-16 sm:h-3.5 sm:w-20" />
                    <Skeleton className="h-5 w-20 sm:h-7 sm:w-28" />
                  </div>
                  <Skeleton className="size-6 shrink-0 rounded-lg sm:size-8 sm:rounded-xl" />
                </div>
                <Skeleton className="mt-2 h-2.5 w-24 sm:mt-2.5 sm:h-3 sm:w-32" />
              </CardContent>
            </Card>
          ))}
          <div className="w-2 shrink-0 sm:hidden" aria-hidden="true" />
        </div>
      </section>

      {/* 2. Monthly Forecast Chart */}
      <section>
        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-36" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="relative rounded-lg border bg-card p-3 sm:p-4">
              <div className="pointer-events-none absolute inset-0 rounded-lg bg-linear-to-b from-muted/25 to-transparent" />
              <div className="relative h-[240px]">
                <svg
                  viewBox={`0 0 ${chartWidth} ${chartHeight}`}
                  role="img"
                  aria-label="Loading forecast chart"
                  className="absolute inset-0 size-full"
                  preserveAspectRatio="none"
                >
                  <defs>
                    <linearGradient id="forecast-skeleton-area-fill" x1="0" x2="0" y1="0" y2="1">
                      <stop offset="0%" stopColor="var(--primary)" stopOpacity="0.10" />
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
                  <g className="reports-skeleton-wave">
                    <path d={skeletonAreaPath} fill="url(#forecast-skeleton-area-fill)" />
                    <path
                      d={skeletonLinePath}
                      fill="none"
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      className="stroke-primary/35"
                      strokeWidth="2.5"
                      vectorEffect="non-scaling-stroke"
                    />
                  </g>
                </svg>
                {Array.from({ length: 12 }).map((_, index) => {
                  const x = chartPadding.left + (chartSpanX * index) / 11
                  const showOnMobile = index === 0 || index === 11
                  const showOnDesktop = showOnMobile || index % 3 === 0
                  if (!showOnDesktop) return null
                  const labelAlignmentClass = index === 0
                    ? "text-left"
                    : index === 11
                      ? "text-right"
                      : "-translate-x-1/2"
                  const labelPositionStyle = index === 11
                    ? { right: `${((chartWidth - x) / chartWidth) * 100}%`, top: `${((chartHeight - 18) / chartHeight) * 100}%` }
                    : { left: `${(x / chartWidth) * 100}%`, top: `${((chartHeight - 18) / chartHeight) * 100}%` }
                  return (
                    <div
                      key={index}
                      className={`pointer-events-none absolute w-max ${showOnMobile ? "" : "hidden sm:block"} ${labelAlignmentClass}`}
                      style={labelPositionStyle}
                    >
                      <Skeleton className="h-3 w-8" />
                    </div>
                  )
                })}
                {skeletonPoints.map((pt, index) => (
                  <div
                    key={index}
                    className="pointer-events-none absolute -translate-x-1/2 -translate-y-1/2"
                    style={{
                      left: `${(pt.x / chartWidth) * 100}%`,
                      top: `${(pt.y / chartHeight) * 100}%`,
                    }}
                  >
                    <span
                      className="reports-skeleton-wave block size-2 rounded-full border border-primary/40 bg-card"
                      style={{ animationDelay: `${index * 80}ms` }}
                    />
                  </div>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>
      </section>

      {/* 3. Breakdown Panels */}
      <section className="grid gap-6 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, panelIndex) => (
          <Card key={panelIndex} className="reports-skeleton-shimmer">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base">
                <Skeleton className="size-4 rounded" />
                <Skeleton className="h-5 w-28" />
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {[72, 46, 28].map((pct, rowIndex) => (
                  <div key={rowIndex} className="space-y-2">
                    <div className="flex items-center justify-between gap-3">
                      <div className="space-y-1">
                        <Skeleton className={`h-3.5 ${rowIndex === 0 ? "w-24" : rowIndex === 1 ? "w-20" : "w-16"}`} />
                        <Skeleton className="h-3 w-8" />
                      </div>
                      <div className="space-y-1 text-right">
                        <Skeleton className="h-3.5 w-16 ml-auto" />
                        <Skeleton className="h-3 w-10 ml-auto" />
                      </div>
                    </div>
                    <div className="h-2 overflow-hidden rounded-full bg-muted">
                      <div
                        className="h-full rounded-full bg-primary/25 animate-pulse"
                        style={{ width: `${pct}%`, animationDelay: `${(panelIndex * 3 + rowIndex) * 120}ms` }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        ))}
      </section>

      {/* 4. Top Subscriptions & Upcoming Renewals */}
      <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-36" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, index) => (
                <div key={index} className="flex w-full items-center gap-3 rounded-lg border p-3">
                  <Skeleton className="size-9 shrink-0 rounded-lg" />
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Skeleton className={`h-4 ${index === 0 ? "w-24" : index === 1 ? "w-32" : "w-20"}`} />
                      <Skeleton className="h-5 w-16 rounded-full" />
                    </div>
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <div className="space-y-1 text-right">
                    <Skeleton className="h-4 w-16 ml-auto" />
                    <Skeleton className="h-3 w-10 ml-auto" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-32" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {Array.from({ length: 3 }).map((_, index) => (
                <div key={index} className="flex w-full items-center gap-3 rounded-lg border p-3">
                  <Skeleton className="size-9 shrink-0 rounded-lg" />
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <Skeleton className={`h-4 ${index === 0 ? "w-28" : index === 1 ? "w-24" : "w-32"}`} />
                    <Skeleton className="h-3 w-36" />
                  </div>
                  <div className="space-y-1 text-right">
                    <Skeleton className="h-4 w-16 ml-auto" />
                    <Skeleton className="h-5 w-18 rounded-full ml-auto" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </section>

      {/* 5. Price Increases & Annual Growth */}
      <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-28" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {Array.from({ length: 2 }).map((_, index) => (
                <div key={index} className="w-full rounded-lg border p-3 space-y-2.5">
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1">
                      <Skeleton className={`h-4 ${index === 0 ? "w-28" : "w-20"}`} />
                      <Skeleton className="h-3 w-24" />
                    </div>
                    <Skeleton className="h-5 w-14 rounded-full" />
                  </div>
                  <div className="mt-3 grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2">
                    <Skeleton className="h-4 w-16" />
                    <Skeleton className="size-3.5 rounded" />
                    <Skeleton className="h-4 w-16 ml-auto" />
                  </div>
                  <Skeleton className="h-3 w-32" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-28" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {[85, 50].map((pct, index) => (
                <div key={index} className="w-full space-y-2">
                  <div className="flex items-start justify-between gap-3">
                    <div className="space-y-1">
                      <Skeleton className={`h-4 ${index === 0 ? "w-24" : "w-32"}`} />
                      <Skeleton className="h-3 w-36" />
                    </div>
                    <div className="space-y-1 text-right">
                      <Skeleton className="h-4 w-16 ml-auto" />
                      <Skeleton className="h-3 w-12 ml-auto" />
                    </div>
                  </div>
                  <div className="h-2 overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full rounded-full bg-primary/25 animate-pulse"
                      style={{ width: `${pct}%`, animationDelay: `${index * 150}ms` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </section>

      {/* 6. Recent Changes */}
      <section>
        <Card className="reports-skeleton-shimmer">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Skeleton className="size-4 rounded" />
              <Skeleton className="h-5 w-28" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="divide-y rounded-lg border">
              {Array.from({ length: 3 }).map((_, index) => (
                <div key={index} className="grid w-full gap-3 p-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center">
                  <div className="space-y-1.5">
                    <div className="flex items-center gap-2">
                      <Skeleton className={`h-4 ${index === 0 ? "w-28" : index === 1 ? "w-20" : "w-24"}`} />
                      <Skeleton className="h-5 w-16 rounded-full" />
                    </div>
                    <Skeleton className="h-3 w-48" />
                  </div>
                  <Skeleton className="h-4 w-20 sm:ml-auto" />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </section>
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
