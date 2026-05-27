import { api } from "@/lib/api"
import type { SubscriptionDetail } from "@/types"

const DETAIL_CACHE_TTL_MS = 120_000

interface CachedSubscriptionDetail {
  cachedAt: number
  detail: SubscriptionDetail
}

const detailCache = new Map<number, CachedSubscriptionDetail>()
const detailRequests = new Map<number, Promise<SubscriptionDetail>>()
const detailVersions = new Map<number, number>()

function detailVersion(id: number): number {
  return detailVersions.get(id) ?? 0
}

function bumpDetailVersion(id: number): void {
  detailVersions.set(id, detailVersion(id) + 1)
}

export function getCachedSubscriptionDetail(id: number): SubscriptionDetail | null {
  const cached = detailCache.get(id)
  if (!cached) {
    return null
  }
  if (Date.now() - cached.cachedAt > DETAIL_CACHE_TTL_MS) {
    detailCache.delete(id)
    return null
  }
  return cached.detail
}

export function loadSubscriptionDetail(id: number): Promise<SubscriptionDetail> {
  const cached = getCachedSubscriptionDetail(id)
  if (cached) {
    return Promise.resolve(cached)
  }

  const existing = detailRequests.get(id)
  if (existing) {
    return existing
  }

  const version = detailVersion(id)
  const request = api.get<SubscriptionDetail>(`/subscriptions/${id}/detail`)
    .then((detail) => {
      if (detailVersion(id) === version) {
        detailCache.set(id, {
          cachedAt: Date.now(),
          detail,
        })
      }
      return detail
    })
    .finally(() => {
      if (detailVersion(id) === version) {
        detailRequests.delete(id)
      }
    })

  detailRequests.set(id, request)
  return request
}

export function preloadSubscriptionDetail(id: number): void {
  void loadSubscriptionDetail(id).catch(() => undefined)
}

export function invalidateSubscriptionDetail(id: number): void {
  bumpDetailVersion(id)
  detailCache.delete(id)
  detailRequests.delete(id)
}
