import { useCallback, useEffect, useState } from "react"

import { api } from "@/lib/api"
import { getDefaultCurrency, setDefaultCurrency } from "@/lib/default-currency"
import type {
  Category,
  DashboardBootstrap,
  DashboardSummary,
  PaymentMethod,
  Subscription,
  UserCurrency,
} from "@/types"

interface UseDashboardDataResult {
  categories: Category[]
  error: unknown
  fetchData: () => Promise<void>
  loading: boolean
  paymentMethods: PaymentMethod[]
  preferredCurrency: string
  subscriptions: Subscription[]
  summary: DashboardSummary | null
  userCurrencies: UserCurrency[]
}

export function requestDashboardBootstrap(
  get: typeof api.get = api.get
): Promise<DashboardBootstrap> {
  return get<DashboardBootstrap>("/dashboard/bootstrap")
}

export function useDashboardData(): UseDashboardDataResult {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<unknown>(null)
  const [preferredCurrency, setPreferredCurrency] = useState(getDefaultCurrency())
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])

  const applyData = useCallback((data: DashboardBootstrap) => {
    setSubscriptions(data.subscriptions || [])
    setSummary(data.summary)
    setUserCurrencies(data.currencies || [])
    setCategories(data.categories || [])
    setPaymentMethods(data.payment_methods || [])
    if (data.preferred_currency) {
      setPreferredCurrency(data.preferred_currency)
      setDefaultCurrency(data.preferred_currency)
    }
  }, [])

  const fetchData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      applyData(await requestDashboardBootstrap())
    } catch (error) {
      setError(error)
    } finally {
      setLoading(false)
    }
  }, [applyData])

  // Initial load: state starts in the loading state, so only settle it from
  // the request callbacks instead of resetting it synchronously in the effect.
  useEffect(() => {
    let cancelled = false
    requestDashboardBootstrap()
      .then((data) => {
        if (!cancelled) applyData(data)
      })
      .catch((error: unknown) => {
        if (!cancelled) setError(error)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [applyData])

  return {
    categories,
    error,
    fetchData,
    loading,
    paymentMethods,
    preferredCurrency,
    subscriptions,
    summary,
    userCurrencies,
  }
}
