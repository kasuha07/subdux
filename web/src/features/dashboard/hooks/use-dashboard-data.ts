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
  fetchData: () => Promise<void>
  loading: boolean
  paymentMethods: PaymentMethod[]
  preferredCurrency: string
  subscriptions: Subscription[]
  summary: DashboardSummary | null
  userCurrencies: UserCurrency[]
}

export function useDashboardData(): UseDashboardDataResult {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [preferredCurrency, setPreferredCurrency] = useState(getDefaultCurrency())
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])

  const fetchData = useCallback(async () => {
    try {
      const data = await api.get<DashboardBootstrap>("/dashboard/bootstrap")
      setSubscriptions(data.subscriptions || [])
      setSummary(data.summary)
      setUserCurrencies(data.currencies || [])
      setCategories(data.categories || [])
      setPaymentMethods(data.payment_methods || [])
      if (data.preferred_currency) {
        setPreferredCurrency(data.preferred_currency)
        setDefaultCurrency(data.preferred_currency)
      }
    } catch {
      void 0
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void fetchData()
  }, [fetchData])

  return {
    categories,
    fetchData,
    loading,
    paymentMethods,
    preferredCurrency,
    subscriptions,
    summary,
    userCurrencies,
  }
}
