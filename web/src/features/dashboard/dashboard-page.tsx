import { useState, useEffect, useCallback, useMemo } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { api, isAdmin } from "@/lib/api"
import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import { formatCurrency } from "@/lib/utils"
import { toast } from "sonner"
import type {
  Subscription,
  DashboardSummary,
  CreateSubscriptionInput,
  UserPreference,
  UserCurrency,
  Category,
  PaymentMethod,
} from "@/types"
import SubscriptionCard from "@/features/subscriptions/subscription-card"
import SubscriptionForm from "@/features/subscriptions/subscription-form"
import {
  Plus,
  Settings,
  DollarSign,
  CalendarDays,
  Repeat,
  TrendingUp,
  Shield,
  Search,
  Filter,
  FilterX,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
} from "lucide-react"

type SortField = "next_billing_date" | "name" | "created_at" | "amount"
type SortDirection = "asc" | "desc"
type EnabledFilter = "enabled" | "disabled"

const defaultSortField: SortField = "next_billing_date"
const defaultSortDirection: SortDirection = "asc"
const enabledOptions: EnabledFilter[] = ["enabled", "disabled"]
const sortFieldOptions: SortField[] = ["next_billing_date", "name", "created_at", "amount"]

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
              <div className="flex items-center gap-3 shrink-0">
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
  const [selectedPaymentMethodIDs, setSelectedPaymentMethodIDs] = useState<Set<number>>(new Set())
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

  const getSubscriptionCategoryName = useCallback((sub: Subscription): string => {
    if (sub.category_id != null) {
      const category = categoryMap.get(sub.category_id)
      if (category) {
        return getCategoryLabel(category, t)
      }
    }
    return sub.category.trim()
  }, [categoryMap, t])

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
    () =>
      new Map(
        userCurrencies.map((item) => [item.code.toUpperCase(), item.symbol.trim()] as const)
      ),
    [userCurrencies]
  )

  const paymentMethodLabelMap = useMemo(
    () =>
      new Map(
        paymentMethods.map((item) => [item.id, getPaymentMethodLabel(item, t)] as const)
      ),
    [paymentMethods, t]
  )

  const filteredSubscriptions = useMemo(() => {
    const normalizedSearchTerm = searchTerm.trim().toLowerCase()

    const filtered = subscriptions.filter((sub) => {
      const categoryName = getSubscriptionCategoryName(sub)

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

      if (selectedCategories.size > 0 && !selectedCategories.has(categoryName)) {
        return false
      }

      if (selectedPaymentMethodIDs.size > 0) {
        if (sub.payment_method_id == null || !selectedPaymentMethodIDs.has(sub.payment_method_id)) {
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
  }, [subscriptions, searchTerm, selectedEnabledStates, selectedCategories, selectedPaymentMethodIDs, sortField, sortDirection, getSubscriptionCategoryName, i18n.language])

  const hasActiveFilters = searchTerm.trim().length > 0 ||
    selectedEnabledStates.size > 0 ||
    selectedCategories.size > 0 ||
    selectedPaymentMethodIDs.size > 0 ||
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
    } else {
      const created = await api.post<Subscription>("/subscriptions", data)
      toast.success(t("dashboard.createSuccess"))
      setEditingSub(null)
      setFormOpen(false)
      await fetchData()
      return created
    }
  }

  function openNewForm() {
    setEditingSub(null)
    setFormOpen(true)
  }

  function resetFiltersAndSorting() {
    setSearchTerm("")
    setSelectedEnabledStates(new Set())
    setSelectedCategories(new Set())
    setSelectedPaymentMethodIDs(new Set())
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
              <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <DollarSign className="size-4" />
                      <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.monthly")}</span>
                    </div>
                    <p className="mt-1 text-2xl font-bold tabular-nums">
                      {formatCurrency(summary.total_monthly, summary.currency || preferredCurrency, i18n.language)}
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <TrendingUp className="size-4" />
                      <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.yearly")}</span>
                    </div>
                    <p className="mt-1 text-2xl font-bold tabular-nums">
                      {formatCurrency(summary.total_yearly, summary.currency || preferredCurrency, i18n.language)}
                    </p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <Repeat className="size-4" />
                      <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.enabled")}</span>
                    </div>
                    <p className="mt-1 text-2xl font-bold tabular-nums">{summary.enabled_count}</p>
                  </CardContent>
                </Card>
                <Card>
                  <CardContent className="p-4">
                    <div className="flex items-center gap-2 text-muted-foreground">
                      <CalendarDays className="size-4" />
                      <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.upcoming")}</span>
                    </div>
                    <p className="mt-1 text-2xl font-bold tabular-nums">
                      {summary.upcoming_renewal_count ?? 0}
                    </p>
                  </CardContent>
                </Card>
              </div>
            )}

            <Separator className="mb-6" />

            <div className="mb-4 flex flex-wrap items-center justify-end gap-2 lg:flex-nowrap">
              <div className="relative w-56 shrink-0 lg:w-72">
                <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
                <Input
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  placeholder={t("dashboard.filters.searchPlaceholder")}
                  className="pl-9"
                />
              </div>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="shrink-0">
                    <Filter className="size-4" />
                    {t("dashboard.filters.filter")}
                    {(selectedEnabledStates.size > 0 || selectedCategories.size > 0 || selectedPaymentMethodIDs.size > 0)
                      ? ` (${selectedEnabledStates.size + selectedCategories.size + selectedPaymentMethodIDs.size})`
                      : ""}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-52">
                  <DropdownMenuLabel>{t("dashboard.filters.filter")}</DropdownMenuLabel>
                  <DropdownMenuSeparator />

                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>{t("dashboard.filters.status")}</DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-48">
                      {enabledOptions.map((status) => (
                        <DropdownMenuCheckboxItem
                          key={status}
                          checked={selectedEnabledStates.has(status)}
                          onSelect={(event) => event.preventDefault()}
                          onCheckedChange={(checked) => {
                            setSelectedEnabledStates((prev) => {
                              const next = new Set(prev)
                              if (checked === true) {
                                next.add(status)
                              } else {
                                next.delete(status)
                              }
                              return next
                            })
                          }}
                        >
                          {t(`subscription.card.status.${status}`)}
                        </DropdownMenuCheckboxItem>
                      ))}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>

                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>{t("dashboard.filters.category")}</DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-56">
                      {categoryOptions.length > 0 ? (
                        categoryOptions.map((category) => (
                          <DropdownMenuCheckboxItem
                            key={category}
                            checked={selectedCategories.has(category)}
                            onSelect={(event) => event.preventDefault()}
                            onCheckedChange={(checked) => {
                              setSelectedCategories((prev) => {
                                const next = new Set(prev)
                                if (checked === true) {
                                  next.add(category)
                                } else {
                                  next.delete(category)
                                }
                                return next
                              })
                            }}
                          >
                            {category}
                          </DropdownMenuCheckboxItem>
                        ))
                      ) : (
                        <div className="text-muted-foreground px-2 py-1.5 text-sm">
                          {t("dashboard.filters.noCategories")}
                        </div>
                      )}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>

                  <DropdownMenuSub>
                    <DropdownMenuSubTrigger>{t("dashboard.filters.paymentMethod")}</DropdownMenuSubTrigger>
                    <DropdownMenuSubContent className="w-56">
                      {paymentMethods.length > 0 ? (
                        paymentMethods.map((method) => (
                          <DropdownMenuCheckboxItem
                            key={method.id}
                            checked={selectedPaymentMethodIDs.has(method.id)}
                            onSelect={(event) => event.preventDefault()}
                            onCheckedChange={(checked) => {
                              setSelectedPaymentMethodIDs((prev) => {
                                const next = new Set(prev)
                                if (checked === true) {
                                  next.add(method.id)
                                } else {
                                  next.delete(method.id)
                                }
                                return next
                              })
                            }}
                          >
                            {paymentMethodLabelMap.get(method.id) ?? method.name}
                          </DropdownMenuCheckboxItem>
                        ))
                      ) : (
                        <div className="text-muted-foreground px-2 py-1.5 text-sm">
                          {t("dashboard.filters.noPaymentMethods")}
                        </div>
                      )}
                    </DropdownMenuSubContent>
                  </DropdownMenuSub>

                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onSelect={(event) => {
                      event.preventDefault()
                      resetFiltersAndSorting()
                    }}
                    disabled={!hasActiveFilters}
                  >
                    <FilterX className="size-4" />
                    {t("dashboard.filters.clearFilters")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm" className="shrink-0">
                    <ArrowUpDown className="size-4" />
                    {getSortFieldLabel(sortField)}
                    {sortDirection === "asc" ? <ArrowUp className="size-3.5" /> : <ArrowDown className="size-3.5" />}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-56">
                  <DropdownMenuLabel>{t("dashboard.filters.sort")}</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {sortFieldOptions.map((field) => (
                    <DropdownMenuItem
                      key={field}
                      onSelect={(event) => {
                        event.preventDefault()
                        handleSortFieldSelect(field)
                      }}
                    >
                      {getSortFieldLabel(field)}
                      {sortField === field ? (
                        sortDirection === "asc" ? <ArrowUp className="ml-auto size-3.5" /> : <ArrowDown className="ml-auto size-3.5" />
                      ) : null}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>

            </div>

            <div className="space-y-2">
              {subscriptions.length === 0 ? (
                <div className="flex flex-col items-center justify-center py-16 text-center">
                  <div className="rounded-full bg-muted p-4 mb-4">
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
                  <p className="text-muted-foreground mt-1 text-sm">
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
                    paymentMethodName={sub.payment_method_id ? paymentMethodLabelMap.get(sub.payment_method_id) : undefined}
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
