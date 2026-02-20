export interface PresetCurrency {
  code: string
  symbol: string
  alias: string
}

export const PRESET_CURRENCIES: PresetCurrency[] = [
  { code: "USD", symbol: "$", alias: "US Dollar" },
  { code: "EUR", symbol: "€", alias: "Euro" },
  { code: "GBP", symbol: "£", alias: "British Pound" },
  { code: "JPY", symbol: "¥", alias: "Japanese Yen" },
  { code: "CNY", symbol: "￥", alias: "Chinese Yuan" },
  { code: "CAD", symbol: "C$", alias: "Canadian Dollar" },
  { code: "AUD", symbol: "A$", alias: "Australian Dollar" },
  { code: "CHF", symbol: "CHF", alias: "Swiss Franc" },
  { code: "HKD", symbol: "HK$", alias: "Hong Kong Dollar" },
  { code: "SGD", symbol: "S$", alias: "Singapore Dollar" },
  { code: "KRW", symbol: "₩", alias: "South Korean Won" },
  { code: "INR", symbol: "₹", alias: "Indian Rupee" },
  { code: "BRL", symbol: "R$", alias: "Brazilian Real" },
  { code: "MXN", symbol: "MX$", alias: "Mexican Peso" },
  { code: "TWD", symbol: "NT$", alias: "New Taiwan Dollar" },
  { code: "THB", symbol: "฿", alias: "Thai Baht" },
  { code: "TRY", symbol: "₺", alias: "Turkish Lira" },
  { code: "NZD", symbol: "NZ$", alias: "New Zealand Dollar" },
  { code: "SEK", symbol: "kr", alias: "Swedish Krona" },
  { code: "NOK", symbol: "kr", alias: "Norwegian Krone" },
  { code: "DKK", symbol: "kr", alias: "Danish Krone" },
  { code: "PLN", symbol: "zł", alias: "Polish Zloty" },
]

export const DEFAULT_CURRENCY_FALLBACK = ["USD", "EUR", "GBP", "CNY", "JPY"]

const presetCurrencyMap = new Map(
  PRESET_CURRENCIES.map((item) => [item.code, item] as const)
)

export function getPresetCurrencyMeta(code: string): PresetCurrency | undefined {
  return presetCurrencyMap.get(code.toUpperCase())
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
