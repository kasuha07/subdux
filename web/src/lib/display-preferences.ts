export const DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY = "displayAllAmountsInPrimaryCurrency"

export function getDisplayAllAmountsInPrimaryCurrency(): boolean {
  return localStorage.getItem(DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY) === "true"
}

export function setDisplayAllAmountsInPrimaryCurrency(enabled: boolean): void {
  localStorage.setItem(DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY, enabled ? "true" : "false")
}
