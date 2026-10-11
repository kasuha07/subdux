import type { Subscription } from "@/types"

// A subscription may carry one scheduled price change: every charge dated on
// or after pending_from is billed at pending_amount instead of amount. The
// form presents it as a free trial (amount 0) or an introductory price; the
// backend switches the amount automatically once pending_from is reached.

export type IntroKind = "trial" | "intro"

export interface ChargeSchedule {
  recurrenceType: string
  intervalCount: number | null
  intervalUnit: string
  monthlyDay: number | null
  yearlyMonth: number | null
  yearlyDay: number | null
}

const DATE_KEY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/
const MAX_SCHEDULE_STEPS = 400

function parseDateKey(value: string): Date | null {
  const match = DATE_KEY_PATTERN.exec(value.trim().slice(0, 10))
  if (!match) {
    return null
  }
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3])))
  return Number.isNaN(date.getTime()) ? null : date
}

function toDateKey(date: Date): string {
  const year = String(date.getUTCFullYear()).padStart(4, "0")
  const month = String(date.getUTCMonth() + 1).padStart(2, "0")
  const day = String(date.getUTCDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

function buildDate(year: number, monthIndex: number, preferredDay: number): Date {
  const normalizedYear = year + Math.floor(monthIndex / 12)
  const normalizedMonth = ((monthIndex % 12) + 12) % 12
  const daysInMonth = new Date(Date.UTC(normalizedYear, normalizedMonth + 1, 0)).getUTCDate()
  const day = Math.min(Math.max(preferredDay, 1), daysInMonth)
  return new Date(Date.UTC(normalizedYear, normalizedMonth, day))
}

export function addDaysToDateKey(value: string, days: number): string {
  const date = parseDateKey(value)
  if (!date) {
    return value
  }
  date.setUTCDate(date.getUTCDate() + days)
  return toDateKey(date)
}

export function daysBetweenDateKeys(from: string, to: string): number {
  const start = parseDateKey(from)
  const end = parseDateKey(to)
  if (!start || !end) {
    return 0
  }
  return Math.round((end.getTime() - start.getTime()) / 86_400_000)
}

// Mirrors the backend's nextRecurringOccurrenceAfter: the charge following
// current, stepping from current itself.
function nextChargeAfter(current: Date, schedule: ChargeSchedule): Date | null {
  switch (schedule.recurrenceType) {
    case "interval": {
      const count = schedule.intervalCount ?? 0
      if (count < 1) {
        return null
      }
      const year = current.getUTCFullYear()
      const month = current.getUTCMonth()
      const day = current.getUTCDate()
      switch (schedule.intervalUnit) {
        case "day":
          return new Date(Date.UTC(year, month, day + count))
        case "week":
          return new Date(Date.UTC(year, month, day + count * 7))
        case "month":
          return buildDate(year, month + count, day)
        case "year":
          return buildDate(year + count, month, day)
        default:
          return null
      }
    }
    case "monthly_date": {
      if (!schedule.monthlyDay) {
        return null
      }
      const from = new Date(current.getTime() + 86_400_000)
      const candidate = buildDate(from.getUTCFullYear(), from.getUTCMonth(), schedule.monthlyDay)
      return candidate < from
        ? buildDate(from.getUTCFullYear(), from.getUTCMonth() + 1, schedule.monthlyDay)
        : candidate
    }
    case "yearly_date": {
      if (!schedule.yearlyMonth || !schedule.yearlyDay) {
        return null
      }
      const from = new Date(current.getTime() + 86_400_000)
      const candidate = buildDate(from.getUTCFullYear(), schedule.yearlyMonth - 1, schedule.yearlyDay)
      return candidate < from
        ? buildDate(from.getUTCFullYear() + 1, schedule.yearlyMonth - 1, schedule.yearlyDay)
        : candidate
    }
    default:
      return null
  }
}

// Returns the first `count` charge dates starting with firstCharge.
export function chargeDatesFrom(firstCharge: string, schedule: ChargeSchedule, count: number): string[] {
  let current = parseDateKey(firstCharge)
  const dates: string[] = []
  while (current && dates.length < count) {
    dates.push(toDateKey(current))
    const next = nextChargeAfter(current, schedule)
    if (!next || next <= current) {
      break
    }
    current = next
  }
  return dates
}

export function chargeScheduleFromSubscription(subscription: Subscription): ChargeSchedule {
  return {
    recurrenceType: subscription.recurrence_type,
    intervalCount: subscription.interval_count,
    intervalUnit: subscription.interval_unit,
    monthlyDay: subscription.monthly_day,
    yearlyMonth: subscription.yearly_month,
    yearlyDay: subscription.yearly_day,
  }
}

export function hasPendingPrice(subscription: Subscription): boolean {
  return subscription.pending_amount !== null && subscription.pending_amount !== undefined &&
    !!subscription.pending_from
}

export function getIntroKind(subscription: Subscription): IntroKind | null {
  if (!hasPendingPrice(subscription)) {
    return null
  }
  return subscription.amount === 0 ? "trial" : "intro"
}

export function chargeAmountOn(subscription: Subscription, dateKey: string): number {
  if (hasPendingPrice(subscription) && dateKey.slice(0, 10) >= (subscription.pending_from ?? "")) {
    return subscription.pending_amount ?? subscription.amount
  }
  return subscription.amount
}

// The regular price: the scheduled price while an introduction runs,
// otherwise the current amount.
export function regularAmount(subscription: Subscription): number {
  return hasPendingPrice(subscription) ? subscription.pending_amount ?? subscription.amount : subscription.amount
}

// Counts the charges still billed at the introductory price, starting with
// the next billing date.
export function remainingIntroCharges(subscription: Subscription): number {
  if (!hasPendingPrice(subscription) || !subscription.next_billing_date || !subscription.pending_from) {
    return 0
  }
  const pendingFrom = subscription.pending_from
  const dates = chargeDatesFrom(
    subscription.next_billing_date,
    chargeScheduleFromSubscription(subscription),
    MAX_SCHEDULE_STEPS
  )
  let count = 0
  for (const date of dates) {
    if (date >= pendingFrom) {
      break
    }
    count++
  }
  return count
}
