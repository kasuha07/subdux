import { useCallback, useMemo, useState } from "react"
import type { TFunction } from "i18next"

import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import type {
  Category,
  PaymentMethod,
  Subscription,
  SubscriptionRenewalMode,
} from "@/types"

import {
  defaultSortDirection,
  defaultSortField,
  type SortDirection,
  type SortField,
  type StatusFilter,
} from "@/features/dashboard/dashboard-filter-constants"
import { getComparableSubscriptionAmount } from "@/features/dashboard/dashboard-amount-utils"
import {
  getSubscriptionRenewalMode,
  getSubscriptionStatus,
} from "@/features/subscriptions/subscription-lifecycle"

interface UseDashboardFiltersOptions {
  categories: Category[]
  displayDisabledSubscriptionsLast: boolean
  exchangeRates: Record<string, number>
  language: string
  paymentMethods: PaymentMethod[]
  preferredCurrency: string
  subscriptions: Subscription[]
  t: TFunction<"translation", undefined>
}

interface UseDashboardFiltersResult {
  categoryOptions: string[]
  filteredSubscriptions: Subscription[]
  getSortFieldLabel: (field: SortField) => string
  getSubscriptionCategoryName: (sub: Subscription) => string
  handleSortFieldSelect: (field: SortField) => void
  handleToggleCategory: (category: string, checked: boolean) => void
  handleToggleRenewalMode: (mode: SubscriptionRenewalMode, checked: boolean) => void
  handleToggleStatus: (status: StatusFilter, checked: boolean) => void
  handleTogglePaymentMethod: (paymentMethodID: number, checked: boolean) => void
  hasActiveFilters: boolean
  includeNoCategory: boolean
  includeNoPaymentMethod: boolean
  onToggleNoCategory: (checked: boolean) => void
  onToggleNoPaymentMethod: (checked: boolean) => void
  paymentMethodLabelMap: Map<number, string>
  resetFiltersAndSorting: () => void
  searchTerm: string
  selectedCategories: Set<string>
  selectedPaymentMethodIDs: Set<number>
  selectedRenewalModes: Set<SubscriptionRenewalMode>
  selectedStatuses: Set<StatusFilter>
  setSearchTerm: (value: string) => void
  sortDirection: SortDirection
  sortField: SortField
}

function toTimestamp(value: string | null): number {
  if (!value) {
    return Number.MAX_SAFE_INTEGER
  }
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? Number.MAX_SAFE_INTEGER : ts
}

export function useDashboardFilters({
  categories,
  displayDisabledSubscriptionsLast,
  exchangeRates,
  language,
  paymentMethods,
  preferredCurrency,
  subscriptions,
  t,
}: UseDashboardFiltersOptions): UseDashboardFiltersResult {
  const [searchTerm, setSearchTerm] = useState("")
  const [selectedStatuses, setSelectedStatuses] = useState<Set<StatusFilter>>(new Set(["active"]))
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set())
  const [includeNoCategory, setIncludeNoCategory] = useState(false)
  const [selectedPaymentMethodIDs, setSelectedPaymentMethodIDs] = useState<Set<number>>(new Set())
  const [selectedRenewalModes, setSelectedRenewalModes] = useState<Set<SubscriptionRenewalMode>>(new Set())
  const [includeNoPaymentMethod, setIncludeNoPaymentMethod] = useState(false)
  const [sortField, setSortField] = useState<SortField>(defaultSortField)
  const [sortDirection, setSortDirection] = useState<SortDirection>(defaultSortDirection)

  const categoryMap = useMemo(
    () => new Map(categories.map((item) => [item.id, item] as const)),
    [categories]
  )

  const getSubscriptionCategoryName = useCallback(
    (sub: Subscription): string => {
      if (sub.category_id != null) {
        const category = categoryMap.get(sub.category_id)
        if (category) {
          return getCategoryLabel(category, t)
        }
      }
      return sub.category.trim()
    },
    [categoryMap, t]
  )

  const categoryOptions = useMemo(() => {
    const uniqueCategories = new Set<string>()

    for (const sub of subscriptions) {
      const normalizedCategory = getSubscriptionCategoryName(sub)
      if (normalizedCategory) {
        uniqueCategories.add(normalizedCategory)
      }
    }

    return Array.from(uniqueCategories).sort((a, b) =>
      a.localeCompare(b, language, { sensitivity: "base" })
    )
  }, [subscriptions, getSubscriptionCategoryName, language])

  const paymentMethodLabelMap = useMemo(
    () => new Map(paymentMethods.map((item) => [item.id, getPaymentMethodLabel(item, t)] as const)),
    [paymentMethods, t]
  )

  const filteredSubscriptions = useMemo(() => {
    const normalizedSearchTerm = searchTerm.trim().toLowerCase()

    const filtered = subscriptions.filter((sub) => {
      const categoryName = getSubscriptionCategoryName(sub)
      const hasCategory = categoryName.trim().length > 0

      if (normalizedSearchTerm) {
        const searchableContent = [sub.name, categoryName, sub.notes].join(" ").toLowerCase()
        if (!searchableContent.includes(normalizedSearchTerm)) {
          return false
        }
      }

      const status = getSubscriptionStatus(sub)
      if (!selectedStatuses.has(status)) {
        return false
      }

      if (selectedRenewalModes.size > 0 && sub.billing_type === "recurring") {
        const renewalMode = getSubscriptionRenewalMode(sub)
        if (!selectedRenewalModes.has(renewalMode)) {
          return false
        }
      }

      if (selectedCategories.size > 0 || includeNoCategory) {
        if (hasCategory) {
          if (!selectedCategories.has(categoryName)) {
            return false
          }
        } else if (!includeNoCategory) {
          return false
        }
      }

      if (selectedPaymentMethodIDs.size > 0 || includeNoPaymentMethod) {
        if (sub.payment_method_id == null) {
          if (!includeNoPaymentMethod) {
            return false
          }
        } else if (!selectedPaymentMethodIDs.has(sub.payment_method_id)) {
          return false
        }
      }

      return true
    })

    return [...filtered].sort((a, b) => {
      if (displayDisabledSubscriptionsLast) {
        const aStatus = getSubscriptionStatus(a)
        const bStatus = getSubscriptionStatus(b)
        if (aStatus !== bStatus) {
          return aStatus === "active" ? -1 : 1
        }
      }

      let result = 0

      if (sortField === "name") {
        result = a.name.localeCompare(b.name, language, { sensitivity: "base" })
      } else if (sortField === "created_at") {
        result = toTimestamp(a.created_at) - toTimestamp(b.created_at)
      } else if (sortField === "amount") {
        result =
          getComparableSubscriptionAmount(a, preferredCurrency, exchangeRates) -
          getComparableSubscriptionAmount(b, preferredCurrency, exchangeRates)
      } else {
        result = toTimestamp(a.next_billing_date) - toTimestamp(b.next_billing_date)
      }

      if (result === 0) {
        result = a.id - b.id
      }

      return sortDirection === "asc" ? result : -result
    })
  }, [
    subscriptions,
    searchTerm,
    selectedStatuses,
    selectedCategories,
    includeNoCategory,
    selectedPaymentMethodIDs,
    selectedRenewalModes,
    includeNoPaymentMethod,
    sortField,
    sortDirection,
    displayDisabledSubscriptionsLast,
    exchangeRates,
    getSubscriptionCategoryName,
    language,
    preferredCurrency,
  ])

  const hasActiveFilters =
    searchTerm.trim().length > 0 ||
    selectedStatuses.size !== 1 ||
    !selectedStatuses.has("active") ||
    selectedCategories.size > 0 ||
    includeNoCategory ||
    selectedPaymentMethodIDs.size > 0 ||
    selectedRenewalModes.size > 0 ||
    includeNoPaymentMethod ||
    sortField !== defaultSortField ||
    sortDirection !== defaultSortDirection

  const resetFiltersAndSorting = useCallback(() => {
    setSearchTerm("")
    setSelectedStatuses(new Set(["active"]))
    setSelectedCategories(new Set())
    setIncludeNoCategory(false)
    setSelectedPaymentMethodIDs(new Set())
    setSelectedRenewalModes(new Set())
    setIncludeNoPaymentMethod(false)
    setSortField(defaultSortField)
    setSortDirection(defaultSortDirection)
  }, [])

  const getSortFieldLabel = useCallback(
    (field: SortField): string => {
      if (field === "name") return t("dashboard.filters.sortFields.name")
      if (field === "created_at") return t("dashboard.filters.sortFields.createdAt")
      if (field === "amount") return t("dashboard.filters.sortFields.amount")
      return t("dashboard.filters.sortFields.nextBillingDate")
    },
    [t]
  )

  const handleSortFieldSelect = useCallback(
    (field: SortField) => {
      if (sortField === field) {
        setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"))
        return
      }

      setSortField(field)
      setSortDirection(defaultSortDirection)
    },
    [sortField]
  )

  const handleToggleStatus = useCallback((status: StatusFilter, checked: boolean) => {
    setSelectedStatuses((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(status)
      } else {
        next.delete(status)
      }
      if (next.size === 0) {
        next.add("active")
      }
      return next
    })
  }, [])

  const handleToggleRenewalMode = useCallback((mode: SubscriptionRenewalMode, checked: boolean) => {
    setSelectedRenewalModes((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(mode)
      } else {
        next.delete(mode)
      }
      return next
    })
  }, [])

  const handleToggleCategory = useCallback((category: string, checked: boolean) => {
    setSelectedCategories((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(category)
      } else {
        next.delete(category)
      }
      return next
    })
  }, [])

  const handleTogglePaymentMethod = useCallback((paymentMethodID: number, checked: boolean) => {
    setSelectedPaymentMethodIDs((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(paymentMethodID)
      } else {
        next.delete(paymentMethodID)
      }
      return next
    })
  }, [])

  return {
    categoryOptions,
    filteredSubscriptions,
    getSortFieldLabel,
    getSubscriptionCategoryName,
    handleSortFieldSelect,
    handleToggleCategory,
    handleToggleRenewalMode,
    handleToggleStatus,
    handleTogglePaymentMethod,
    hasActiveFilters,
    includeNoCategory,
    includeNoPaymentMethod,
    onToggleNoCategory: setIncludeNoCategory,
    onToggleNoPaymentMethod: setIncludeNoPaymentMethod,
    paymentMethodLabelMap,
    resetFiltersAndSorting,
    searchTerm,
    selectedCategories,
    selectedPaymentMethodIDs,
    selectedRenewalModes,
    selectedStatuses,
    setSearchTerm,
    sortDirection,
    sortField,
  }
}
