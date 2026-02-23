import { useCallback, useEffect, useState } from "react"

import { api } from "@/lib/api"
import { getDefaultCurrency, setDefaultCurrency } from "@/lib/default-currency"
import type {
  Category,
  DashboardSummary,
  PaymentMethod,
  Subscription,
  UserCurrency,
  UserPreference,
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
      const [subs, sum, pref, currencies, categoryList, methods] = await Promise.all([
        api.get<Subscription[]>("/subscriptions"),
        api.get<DashboardSummary>("/dashboard/summary"),
        api.get<UserPreference>("/preferences/currency"),
        api.get<UserCurrency[]>("/currencies"),
        api.get<Category[]>("/categories"),
        api.get<PaymentMethod[]>("/payment-methods"),
      ])
      setSubscriptions(subs || [])
      setSummary(sum)
      setUserCurrencies(currencies || [])
      setCategories(categoryList || [])
      setPaymentMethods(methods || [])
      if (pref?.preferred_currency) {
        setPreferredCurrency(pref.preferred_currency)
        setDefaultCurrency(pref.preferred_currency)
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
