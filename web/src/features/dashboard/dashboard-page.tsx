import { useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Plus, Settings, Shield } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { api, isAdmin } from "@/lib/api"
import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import { toast } from "sonner"
import type {
  Category,
  CreateSubscriptionInput,
  DashboardSummary,
  PaymentMethod,
  Subscription,
  UserCurrency,
  UserPreference,
} from "@/types"

import SubscriptionCard from "@/features/subscriptions/subscription-card"
import SubscriptionForm from "@/features/subscriptions/subscription-form"
import {
  defaultSortDirection,
  defaultSortField,
  type EnabledFilter,
  type SortDirection,
  type SortField,
} from "./dashboard-filter-constants"
import DashboardFiltersToolbar from "./dashboard-filters-toolbar"
import DashboardSummaryCards from "./dashboard-summary-cards"

function toTimestamp(value: string | null): number {
  if (!value) {
    return Number.MAX_SAFE_INTEGER
  }
  const ts = new Date(value).getTime()
  return Number.isNaN(ts) ? Number.MAX_SAFE_INTEGER : ts
}

function DashboardSkeleton() {
  return (
    <>
      <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Card key={i}>
            <CardContent className="p-4">
              <div className="flex items-center gap-2">
                <Skeleton className="size-4 rounded" />
                <Skeleton className="h-3 w-16" />
              </div>
              <Skeleton className="mt-2 h-7 w-20" />
            </CardContent>
          </Card>
        ))}
      </div>

      <Separator className="mb-6" />

      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Card key={i}>
            <CardContent className="flex items-center gap-4 p-4">
              <Skeleton className="size-10 shrink-0 rounded-lg" />
              <div className="min-w-0 flex-1 space-y-2">
                <Skeleton className="h-4 w-32" />
                <Skeleton className="h-3 w-24" />
              </div>
              <div className="flex shrink-0 items-center gap-3">
                <div className="space-y-1.5 text-right">
                  <Skeleton className="ml-auto h-4 w-16" />
                  <Skeleton className="ml-auto h-3 w-12" />
                </div>
                <Skeleton className="h-5 w-14 rounded-full" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
    </>
  )
}

export default function DashboardPage() {
  const { t, i18n } = useTranslation()
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [summary, setSummary] = useState<DashboardSummary | null>(null)
  const [loading, setLoading] = useState(true)
  const [formOpen, setFormOpen] = useState(false)
  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [preferredCurrency, setPreferredCurrency] = useState(
    localStorage.getItem("defaultCurrency") || "USD"
  )
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])
  const [searchTerm, setSearchTerm] = useState("")
  const [selectedEnabledStates, setSelectedEnabledStates] = useState<Set<EnabledFilter>>(new Set())
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set())
  const [includeNoCategory, setIncludeNoCategory] = useState(false)
  const [selectedPaymentMethodIDs, setSelectedPaymentMethodIDs] = useState<Set<number>>(new Set())
  const [includeNoPaymentMethod, setIncludeNoPaymentMethod] = useState(false)
  const [sortField, setSortField] = useState<SortField>(defaultSortField)
  const [sortDirection, setSortDirection] = useState<SortDirection>(defaultSortDirection)

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
        localStorage.setItem("defaultCurrency", pref.preferred_currency)
      }
    } catch {
      void 0
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

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
      a.localeCompare(b, i18n.language, { sensitivity: "base" })
    )
  }, [subscriptions, getSubscriptionCategoryName, i18n.language])

  const currencySymbolMap = useMemo(
    () => new Map(userCurrencies.map((item) => [item.code.toUpperCase(), item.symbol.trim()] as const)),
    [userCurrencies]
  )

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

      if (selectedEnabledStates.size > 0) {
        const enabledKey: EnabledFilter = sub.enabled ? "enabled" : "disabled"
        if (!selectedEnabledStates.has(enabledKey)) {
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
      let result = 0

      if (sortField === "name") {
        result = a.name.localeCompare(b.name, i18n.language, { sensitivity: "base" })
      } else if (sortField === "created_at") {
        result = toTimestamp(a.created_at) - toTimestamp(b.created_at)
      } else if (sortField === "amount") {
        result = a.amount - b.amount
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
    selectedEnabledStates,
    selectedCategories,
    includeNoCategory,
    selectedPaymentMethodIDs,
    includeNoPaymentMethod,
    sortField,
    sortDirection,
    getSubscriptionCategoryName,
    i18n.language,
  ])

  const hasActiveFilters =
    searchTerm.trim().length > 0 ||
    selectedEnabledStates.size > 0 ||
    selectedCategories.size > 0 ||
    includeNoCategory ||
    selectedPaymentMethodIDs.size > 0 ||
    includeNoPaymentMethod ||
    sortField !== defaultSortField ||
    sortDirection !== defaultSortDirection

  function handleEdit(sub: Subscription) {
    setEditingSub(sub)
    setFormOpen(true)
  }

  async function handleDelete(id: number) {
    if (!confirm(t("dashboard.deleteConfirm"))) return
    try {
      await api.delete(`/subscriptions/${id}`)
      toast.success(t("dashboard.deleteSuccess"))
      await fetchData()
    } catch {
      void 0
    }
  }

  async function handleFormSubmit(data: CreateSubscriptionInput) {
    if (editingSub) {
      const updatePayload = {
        ...data,
        payment_method_id: data.payment_method_id ?? 0,
      }
      const updated = await api.put<Subscription>(`/subscriptions/${editingSub.id}`, updatePayload)
      toast.success(t("dashboard.updateSuccess"))
      setEditingSub(null)
      setFormOpen(false)
      await fetchData()
      return updated
    }

    const created = await api.post<Subscription>("/subscriptions", data)
    toast.success(t("dashboard.createSuccess"))
    setEditingSub(null)
    setFormOpen(false)
    await fetchData()
    return created
  }

  function openNewForm() {
    setEditingSub(null)
    setFormOpen(true)
  }

  function resetFiltersAndSorting() {
    setSearchTerm("")
    setSelectedEnabledStates(new Set())
    setSelectedCategories(new Set())
    setIncludeNoCategory(false)
    setSelectedPaymentMethodIDs(new Set())
    setIncludeNoPaymentMethod(false)
    setSortField(defaultSortField)
    setSortDirection(defaultSortDirection)
  }

  function getSortFieldLabel(field: SortField): string {
    if (field === "name") return t("dashboard.filters.sortFields.name")
    if (field === "created_at") return t("dashboard.filters.sortFields.createdAt")
    if (field === "amount") return t("dashboard.filters.sortFields.amount")
    return t("dashboard.filters.sortFields.nextBillingDate")
  }

  function handleSortFieldSelect(field: SortField) {
    if (sortField === field) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"))
      return
    }

    setSortField(field)
    setSortDirection(defaultSortDirection)
  }

  function handleToggleEnabledState(status: EnabledFilter, checked: boolean) {
    setSelectedEnabledStates((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(status)
      } else {
        next.delete(status)
      }
      return next
    })
  }

  function handleToggleCategory(category: string, checked: boolean) {
    setSelectedCategories((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(category)
      } else {
        next.delete(category)
      }
      return next
    })
  }

  function handleTogglePaymentMethod(paymentMethodID: number, checked: boolean) {
    setSelectedPaymentMethodIDs((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(paymentMethodID)
      } else {
        next.delete(paymentMethodID)
      }
      return next
    })
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-4xl items-center justify-between px-4">
          <h1 className="text-lg font-bold tracking-tight">{t("dashboard.title")}</h1>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={openNewForm} disabled={loading}>
              <Plus className="size-4" />
              {t("dashboard.add")}
            </Button>
            {isAdmin() && (
              <Button variant="ghost" size="icon-sm" asChild>
                <Link to="/admin">
                  <Shield className="size-4" />
                </Link>
              </Button>
            )}
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/settings">
                <Settings className="size-4" />
              </Link>
            </Button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-6">
        {loading ? (
          <DashboardSkeleton />
        ) : (
          <>
            {summary && (
              <DashboardSummaryCards
                summary={summary}
                preferredCurrency={preferredCurrency}
                language={i18n.language}
              />
            )}

            <Separator className="mb-6" />

            <DashboardFiltersToolbar
              searchTerm={searchTerm}
              onSearchTermChange={setSearchTerm}
              selectedEnabledStates={selectedEnabledStates}
              selectedCategories={selectedCategories}
              includeNoCategory={includeNoCategory}
              selectedPaymentMethodIDs={selectedPaymentMethodIDs}
              includeNoPaymentMethod={includeNoPaymentMethod}
              categoryOptions={categoryOptions}
              paymentMethods={paymentMethods}
              paymentMethodLabelMap={paymentMethodLabelMap}
              sortField={sortField}
              sortDirection={sortDirection}
              onSortFieldSelect={handleSortFieldSelect}
              getSortFieldLabel={getSortFieldLabel}
              hasActiveFilters={hasActiveFilters}
              onResetFiltersAndSorting={resetFiltersAndSorting}
              onToggleEnabledState={handleToggleEnabledState}
              onToggleCategory={handleToggleCategory}
              onToggleNoCategory={setIncludeNoCategory}
              onTogglePaymentMethod={handleTogglePaymentMethod}
              onToggleNoPaymentMethod={setIncludeNoPaymentMethod}
            />

            <div className="space-y-2">
              {subscriptions.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <div className="mb-4 rounded-full bg-muted p-4">
                    <Plus className="size-6 text-muted-foreground" />
                  </div>
                  <h3 className="font-medium">{t("dashboard.empty.title")}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t("dashboard.empty.description")}
                  </p>
                  <Button className="mt-4" onClick={openNewForm}>
                    <Plus className="size-4" />
                    {t("dashboard.empty.addButton")}
                  </Button>
                </div>
              ) : filteredSubscriptions.length === 0 ? (
                <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12 text-center">
                  <h3 className="font-medium">{t("dashboard.filters.empty.title")}</h3>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {t("dashboard.filters.empty.description")}
                  </p>
                </div>
              ) : (
                filteredSubscriptions.map((sub) => (
                  <SubscriptionCard
                    key={sub.id}
                    subscription={sub}
                    categoryName={getSubscriptionCategoryName(sub)}
                    currencySymbol={currencySymbolMap.get(sub.currency.toUpperCase())}
                    paymentMethodName={
                      sub.payment_method_id
                        ? paymentMethodLabelMap.get(sub.payment_method_id)
                        : undefined
                    }
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                  />
                ))
              )}
            </div>
          </>
        )}
      </main>

      <SubscriptionForm
        key={editingSub?.id ?? "new"}
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditingSub(null)
        }}
        subscription={editingSub}
        onSubmit={handleFormSubmit}
        userCurrencies={userCurrencies}
        categories={categories}
        paymentMethods={paymentMethods}
      />
    </div>
  )
}
