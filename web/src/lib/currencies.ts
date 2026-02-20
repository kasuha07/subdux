export interface PresetCurrency {
  code: string
  symbol: string
  alias: string
}

const PRESET_CURRENCY_CODES = [
  "USD", "EUR", "GBP", "JPY", "CNY", "CAD", "AUD", "CHF", "HKD", "SGD", "KRW",
  "INR", "BRL", "MXN", "TWD", "THB", "TRY", "NZD", "SEK", "NOK", "DKK", "PLN",
] as const

function getIntlCurrencyAlias(code: string, locale: string): string {
  if (typeof Intl.DisplayNames === "function") {
    const displayNames = new Intl.DisplayNames([locale], { type: "currency" })
    const alias = displayNames.of(code)
    if (alias) {
      return alias
    }
  }
  return code
}

function getIntlCurrencySymbol(code: string, locale: string): string {
  const parts = new Intl.NumberFormat(locale, {
    style: "currency",
    currency: code,
    currencyDisplay: "symbol",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).formatToParts(0)

  const symbol = parts.find((part) => part.type === "currency")?.value
  return symbol || code
}

const presetCurrencyCache = new Map<string, PresetCurrency[]>()
const presetCurrencyMapCache = new Map<string, Map<string, PresetCurrency>>()

function normalizeLocale(locale: string): string {
  const normalized = locale.trim()
  return normalized || "en"
}

export function getPresetCurrencies(locale: string = "en"): PresetCurrency[] {
  const normalizedLocale = normalizeLocale(locale)
  const cached = presetCurrencyCache.get(normalizedLocale)
  if (cached) {
    return cached
  }

  const generated = PRESET_CURRENCY_CODES.map((code) => ({
    code,
    symbol: getIntlCurrencySymbol(code, normalizedLocale),
    alias: getIntlCurrencyAlias(code, normalizedLocale),
  }))
  presetCurrencyCache.set(normalizedLocale, generated)
  return generated
}

export const PRESET_CURRENCIES: PresetCurrency[] = getPresetCurrencies("en")

export const DEFAULT_CURRENCY_FALLBACK = ["USD", "EUR", "GBP", "CNY", "JPY"]

function getPresetCurrencyMap(locale: string): Map<string, PresetCurrency> {
  const normalizedLocale = normalizeLocale(locale)
  const cached = presetCurrencyMapCache.get(normalizedLocale)
  if (cached) {
    return cached
  }

  const currencyMap = new Map(
    getPresetCurrencies(normalizedLocale).map((item) => [item.code, item] as const)
  )
  presetCurrencyMapCache.set(normalizedLocale, currencyMap)
  return currencyMap
}

export function getPresetCurrencyMeta(code: string, locale: string = "en"): PresetCurrency | undefined {
  return getPresetCurrencyMap(locale).get(code.toUpperCase())
}

export function formatCurrencyDisplay(code: string, alias?: string, symbol?: string): string {
  const normalizedAlias = alias?.trim() ?? ""
  const normalizedSymbol = symbol?.trim() ?? ""

  if (normalizedAlias && normalizedSymbol) {
    return `${code} - ${normalizedAlias} (${normalizedSymbol})`
  }
  if (normalizedAlias) {
    return `${code} - ${normalizedAlias}`
  }
  if (normalizedSymbol) {
    return `${code} (${normalizedSymbol})`
  }
  return code
}
