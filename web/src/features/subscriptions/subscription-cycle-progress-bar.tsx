import { useTranslation } from "react-i18next"

import { cn } from "@/lib/utils"
import type { Subscription } from "@/types"

const DAY_IN_MS = 24 * 60 * 60 * 1000

function toUTCDate(value: string): Date | null {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return new Date(Date.UTC(parsed.getUTCFullYear(), parsed.getUTCMonth(), parsed.getUTCDate()))
}

function daysInUTCMonth(year: number, monthIndex: number): number {
  return new Date(Date.UTC(year, monthIndex + 1, 0)).getUTCDate()
}

function shiftUTCDate(date: Date, diff: { years?: number, months?: number, days?: number }): Date {
  const years = diff.years ?? 0
  const months = diff.months ?? 0
  const days = diff.days ?? 0
  const baseYear = date.getUTCFullYear() + years
  const baseMonth = date.getUTCMonth() + months
  const normalizedMonthDate = new Date(Date.UTC(baseYear, baseMonth, 1))
  const normalizedYear = normalizedMonthDate.getUTCFullYear()
  const normalizedMonth = normalizedMonthDate.getUTCMonth()
  const day = Math.min(date.getUTCDate(), daysInUTCMonth(normalizedYear, normalizedMonth))

  return new Date(Date.UTC(normalizedYear, normalizedMonth, day) + days * DAY_IN_MS)
}

function getPreviousCycleDate(subscription: Subscription, nextDate: Date): Date | null {
  if (subscription.billing_type === "one_time") {
    return toUTCDate(subscription.created_at)
  }

  if (subscription.recurrence_type === "monthly_date") {
    return shiftUTCDate(nextDate, { months: -1 })
  }

  if (subscription.recurrence_type === "yearly_date") {
    return shiftUTCDate(nextDate, { years: -1 })
  }

  if (subscription.recurrence_type === "interval") {
    const intervalCount = subscription.interval_count && subscription.interval_count > 0
      ? subscription.interval_count
      : 1

    switch (subscription.interval_unit) {
      case "day":
        return shiftUTCDate(nextDate, { days: -intervalCount })
      case "week":
        return shiftUTCDate(nextDate, { days: -intervalCount * 7 })
      case "month":
        return shiftUTCDate(nextDate, { months: -intervalCount })
      case "year":
        return shiftUTCDate(nextDate, { years: -intervalCount })
      default:
        return null
    }
  }

  return null
}

function getCycleProgressPercent(subscription: Subscription): number | null {
  if (!subscription.next_billing_date) {
    return null
  }

  const nextDate = toUTCDate(subscription.next_billing_date)
  if (!nextDate) {
    return null
  }

  const previousCycleDate = getPreviousCycleDate(subscription, nextDate)
  if (!previousCycleDate) {
    return null
  }

  const cycleDuration = nextDate.getTime() - previousCycleDate.getTime()
  if (cycleDuration <= 0) {
    return null
  }

  const now = new Date()
  const today = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()))
  const elapsed = today.getTime() - previousCycleDate.getTime()
  const ratio = elapsed / cycleDuration

  if (ratio <= 0) {
    return 0
  }

  if (ratio >= 1) {
    return 100
  }

  return ratio * 100
}

function getProgressColor(progress: number): string {
  if (progress < 25) {
    return "bg-emerald-500"
  }
  if (progress < 50) {
    return "bg-sky-500"
  }
  if (progress < 75) {
    return "bg-amber-500"
  }
  if (progress < 90) {
    return "bg-orange-500"
  }
  return "bg-rose-500"
}

interface SubscriptionCycleProgressBarProps {
  subscription: Subscription
  className?: string
}

export default function SubscriptionCycleProgressBar({
  subscription,
  className,
}: SubscriptionCycleProgressBarProps) {
  const { t } = useTranslation()
  const progress = getCycleProgressPercent(subscription)

  if (progress === null) {
    return null
  }

  const progressRounded = Math.round(progress)
  const ariaLabel = t("subscription.card.cycleProgressAria", { progress: progressRounded })

  return (
    <div
      className={cn("pointer-events-none absolute inset-x-0 bottom-0 h-1 overflow-hidden bg-border/60", className)}
      role="progressbar"
      aria-label={ariaLabel}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={progressRounded}
      title={ariaLabel}
    >
      <div
        className={cn("h-full transition-[width,background-color] duration-300", getProgressColor(progress))}
        style={{ width: `${progress}%` }}
      />
    </div>
  )
}
