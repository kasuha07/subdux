export const DEFAULT_CURRENCY_KEY = "defaultCurrency"

export function getDefaultCurrency(fallback = "USD"): string {
  return localStorage.getItem(DEFAULT_CURRENCY_KEY) || fallback
}

export function setDefaultCurrency(currency: string): void {
  localStorage.setItem(DEFAULT_CURRENCY_KEY, currency)
}
