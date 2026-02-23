import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

const DAY_IN_MS = 24 * 60 * 60 * 1000
const DATE_ONLY_PATTERN = /^(\d{4})-(\d{2})-(\d{2})/

interface DateParts {
  year: number
  month: number
  day: number
}

function toDateParts(value: string): DateParts | null {
  const match = DATE_ONLY_PATTERN.exec(value.trim())
  if (match) {
    const year = Number.parseInt(match[1], 10)
    const month = Number.parseInt(match[2], 10)
    const day = Number.parseInt(match[3], 10)
    const candidate = new Date(Date.UTC(year, month - 1, day))
    if (
      candidate.getUTCFullYear() === year
      && candidate.getUTCMonth() === month - 1
      && candidate.getUTCDate() === day
    ) {
      return { year, month, day }
    }
    return null
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }

  return {
    year: parsed.getUTCFullYear(),
    month: parsed.getUTCMonth() + 1,
    day: parsed.getUTCDate(),
  }
}

function toLocalDate(value: string): Date | null {
  const parts = toDateParts(value)
  if (!parts) {
    return null
  }
  return new Date(parts.year, parts.month - 1, parts.day)
}

export function formatDateKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatCurrency(amount: number, currency: string = "USD", locale: string = "en-US"): string {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(amount)
}

export function formatCurrencyWithSymbol(
  amount: number,
  currency: string = "USD",
  symbol?: string,
  locale: string = "en-US"
): string {
  const normalizedSymbol = symbol?.trim()
  if (!normalizedSymbol) {
    return formatCurrency(amount, currency, locale)
  }

  const parts = new Intl.NumberFormat(locale, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).formatToParts(amount)

  return parts
    .map((part) => (part.type === "currency" ? normalizedSymbol : part.value))
    .join("")
}

export function formatDate(date: string, locale: string = "en-US"): string {
  const target = toLocalDate(date)
  if (!target) {
    return date
  }

  return target.toLocaleDateString(locale, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

export function daysUntil(date: string): number {
  const target = toLocalDate(date)
  if (!target) {
    return Number.NaN
  }

  const now = new Date()
  // Compare date-only values in user's local calendar day.
  const targetUTC = Date.UTC(target.getFullYear(), target.getMonth(), target.getDate())
  const nowUTC = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())
  const diff = targetUTC - nowUTC
  return Math.round(diff / DAY_IN_MS)
}
