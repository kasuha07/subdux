import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

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
  return new Date(date).toLocaleDateString(locale, {
    month: "short",
    day: "numeric",
    year: "numeric",
  })
}

export function daysUntil(date: string): number {
  const target = new Date(date)
  const now = new Date()
  // Use UTC dates for consistent calculation regardless of local timezone
  const targetUTC = Date.UTC(target.getUTCFullYear(), target.getUTCMonth(), target.getUTCDate())
  const nowUTC = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate())
  const diff = targetUTC - nowUTC
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}
