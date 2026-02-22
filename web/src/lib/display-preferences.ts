export const DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY = "displayAllAmountsInPrimaryCurrency"
export const DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY = "displayRecurringAmountsAsMonthlyCost"

export function getDisplayAllAmountsInPrimaryCurrency(): boolean {
  return localStorage.getItem(DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY) === "true"
}

export function setDisplayAllAmountsInPrimaryCurrency(enabled: boolean): void {
  localStorage.setItem(DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY, enabled ? "true" : "false")
}

export function getDisplayRecurringAmountsAsMonthlyCost(): boolean {
  return localStorage.getItem(DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY) === "true"
}

export function setDisplayRecurringAmountsAsMonthlyCost(enabled: boolean): void {
  localStorage.setItem(DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY, enabled ? "true" : "false")
}
