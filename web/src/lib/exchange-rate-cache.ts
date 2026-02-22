import { api } from "@/lib/api"
import type { ExchangeRateInfo } from "@/types"

interface ExchangeRateCacheEntry {
  expiresAt: number
  rate: number
}

interface ExchangeRateCacheStore {
  rates: Record<string, ExchangeRateCacheEntry>
}

const EXCHANGE_RATE_CACHE_KEY = "exchangeRateCacheV1"
// Cache exchange rates for 6 hours to avoid refetching on every dashboard load.
const EXCHANGE_RATE_CACHE_TTL_MS = 6 * 60 * 60 * 1000

function getCacheKey(baseCurrency: string, targetCurrency: string): string {
  return `${baseCurrency.toUpperCase()}->${targetCurrency.toUpperCase()}`
}

function readExchangeRateCacheStore(): ExchangeRateCacheStore {
  const raw = localStorage.getItem(EXCHANGE_RATE_CACHE_KEY)
  if (!raw) {
    return { rates: {} }
  }

  try {
    const parsed: unknown = JSON.parse(raw)
    if (
      typeof parsed !== "object" ||
      parsed === null ||
      !("rates" in parsed) ||
      typeof parsed.rates !== "object" ||
      parsed.rates === null
    ) {
      return { rates: {} }
    }
    return parsed as ExchangeRateCacheStore
  } catch {
    return { rates: {} }
  }
}

function writeExchangeRateCacheStore(store: ExchangeRateCacheStore): void {
  localStorage.setItem(EXCHANGE_RATE_CACHE_KEY, JSON.stringify(store))
}

function getCachedExchangeRate(baseCurrency: string, targetCurrency: string): number | null {
  const store = readExchangeRateCacheStore()
  const cacheKey = getCacheKey(baseCurrency, targetCurrency)
  const entry = store.rates[cacheKey]
  if (!entry) {
    return null
  }

  if (entry.expiresAt <= Date.now()) {
    delete store.rates[cacheKey]
    writeExchangeRateCacheStore(store)
    return null
  }

  return entry.rate
}

function setCachedExchangeRate(baseCurrency: string, targetCurrency: string, rate: number): void {
  const store = readExchangeRateCacheStore()
  const cacheKey = getCacheKey(baseCurrency, targetCurrency)
  store.rates[cacheKey] = {
    expiresAt: Date.now() + EXCHANGE_RATE_CACHE_TTL_MS,
    rate,
  }
  writeExchangeRateCacheStore(store)
}

function setCachedExchangeRates(entries: Array<{ baseCurrency: string; targetCurrency: string; rate: number }>): void {
  if (entries.length === 0) {
    return
  }

  const store = readExchangeRateCacheStore()
  const expiresAt = Date.now() + EXCHANGE_RATE_CACHE_TTL_MS

  for (const entry of entries) {
    const cacheKey = getCacheKey(entry.baseCurrency, entry.targetCurrency)
    store.rates[cacheKey] = {
      expiresAt,
      rate: entry.rate,
    }
  }

  writeExchangeRateCacheStore(store)
}

export async function getExchangeRatesToTarget(
  baseCurrencies: string[],
  targetCurrency: string
): Promise<Record<string, number>> {
  const normalizedTarget = targetCurrency.toUpperCase()
  const normalizedSources = Array.from(
    new Set(
      baseCurrencies
        .map((currency) => currency.toUpperCase())
        .filter((currency) => currency && currency !== normalizedTarget)
    )
  )

  if (normalizedSources.length === 0) {
    return {}
  }

  const rates: Record<string, number> = {}
  const missingSources = new Set<string>()

  for (const sourceCurrency of normalizedSources) {
    const cachedRate = getCachedExchangeRate(sourceCurrency, normalizedTarget)
    if (cachedRate !== null) {
      rates[sourceCurrency] = cachedRate
    } else {
      missingSources.add(sourceCurrency)
    }
  }

  if (missingSources.size === 0) {
    return rates
  }

  let primaryBaseRates: ExchangeRateInfo[] = []
  try {
    // Use the primary currency as base, then invert to get source -> primary conversion.
    primaryBaseRates = await api.get<ExchangeRateInfo[]>(
      `/exchange-rates?base=${encodeURIComponent(normalizedTarget)}`
    )
  } catch {
    return rates
  }

  const sourcePerPrimaryMap = new Map(
    (primaryBaseRates ?? []).map((item) => [item.target_currency.toUpperCase(), item.rate] as const)
  )

  const cacheEntries: Array<{ baseCurrency: string; targetCurrency: string; rate: number }> = []
  for (const sourceCurrency of missingSources) {
    const sourcePerPrimary = sourcePerPrimaryMap.get(sourceCurrency)
    if (!sourcePerPrimary || sourcePerPrimary <= 0) {
      continue
    }

    const primaryPerSource = 1 / sourcePerPrimary
    rates[sourceCurrency] = primaryPerSource
    cacheEntries.push({
      baseCurrency: sourceCurrency,
      targetCurrency: normalizedTarget,
      rate: primaryPerSource,
    })
  }

  setCachedExchangeRates(cacheEntries)
  return rates
}

export async function getExchangeRate(baseCurrency: string, targetCurrency: string): Promise<number> {
  const normalizedBase = baseCurrency.toUpperCase()
  const normalizedTarget = targetCurrency.toUpperCase()

  if (normalizedBase === normalizedTarget) {
    return 1
  }

  const cachedRate = getCachedExchangeRate(normalizedBase, normalizedTarget)
  if (cachedRate !== null) {
    return cachedRate
  }

  const response = await api.get<ExchangeRateInfo>(
    `/exchange-rates/${normalizedBase}/${normalizedTarget}`
  )
  setCachedExchangeRate(normalizedBase, normalizedTarget, response.rate)
  return response.rate
}
