import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Plus, Settings, Shield } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { useDashboardData } from "@/features/dashboard/hooks/use-dashboard-data"
import { useDashboardFilters } from "@/features/dashboard/hooks/use-dashboard-filters"
import { api, isAdmin } from "@/lib/api"
import {
  DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY,
  DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY,
  DISPLAY_SUBSCRIPTION_CYCLE_PROGRESS_KEY,
  getDisplayAllAmountsInPrimaryCurrency,
  getDisplayRecurringAmountsAsMonthlyCost,
  getDisplaySubscriptionCycleProgress,
} from "@/lib/display-preferences"
import { getExchangeRatesToTarget } from "@/lib/exchange-rate-cache"
import { toast } from "sonner"
import type { CreateSubscriptionInput, Subscription } from "@/types"

import SubscriptionCard from "@/features/subscriptions/subscription-card"
import SubscriptionSquareCard from "@/features/subscriptions/subscription-square-card"
import SubscriptionForm from "@/features/subscriptions/subscription-form"
import DashboardFiltersToolbar from "./dashboard-filters-toolbar"
import DashboardSummaryCards from "./dashboard-summary-cards"

function getMonthlyAmountFactor(subscription: Subscription): number | null {
  if (subscription.billing_type !== "recurring") {
    return null
  }

  if (subscription.recurrence_type === "interval") {
    const intervalCount = subscription.interval_count
    if (!intervalCount || intervalCount <= 0) {
      return null
    }

    switch (subscription.interval_unit) {
      case "day":
        return 30.436875 / intervalCount
      case "week":
        return 4.348125 / intervalCount
      case "month":
        return 1 / intervalCount
      case "year":
        return 1 / (12 * intervalCount)
      default:
        return null
    }
  }

  if (subscription.recurrence_type === "monthly_date") {
    return 1
  }

  if (subscription.recurrence_type === "yearly_date") {
    return 1 / 12
  }

  return null
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
              <Skeleton className="size-11 shrink-0 rounded-lg" />
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
  const [subscriptionView, setSubscriptionView] = useState<"list" | "cards">("list")
  const [formOpen, setFormOpen] = useState(false)
  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [displayAllAmountsInPrimaryCurrency, setDisplayAllAmountsInPrimaryCurrency] = useState(
    getDisplayAllAmountsInPrimaryCurrency()
  )
  const [displayRecurringAmountsAsMonthlyCost, setDisplayRecurringAmountsAsMonthlyCost] = useState(
    getDisplayRecurringAmountsAsMonthlyCost()
  )
  const [displaySubscriptionCycleProgress, setDisplaySubscriptionCycleProgress] = useState(
    getDisplaySubscriptionCycleProgress()
  )
  const [exchangeRates, setExchangeRates] = useState<Record<string, number>>({})

  const {
    categories,
    fetchData,
    loading,
    paymentMethods,
    preferredCurrency,
    subscriptions,
    summary,
    userCurrencies,
  } = useDashboardData()

  const {
    categoryOptions,
    filteredSubscriptions,
    getSortFieldLabel,
    getSubscriptionCategoryName,
    handleSortFieldSelect,
    handleToggleCategory,
    handleToggleEnabledState,
    handleTogglePaymentMethod,
    hasActiveFilters,
    includeNoCategory,
    includeNoPaymentMethod,
    onToggleNoCategory,
    onToggleNoPaymentMethod,
    paymentMethodLabelMap,
    resetFiltersAndSorting,
    searchTerm,
    selectedCategories,
    selectedEnabledStates,
    selectedPaymentMethodIDs,
    setSearchTerm,
    sortDirection,
    sortField,
  } = useDashboardFilters({
    categories,
    language: i18n.language,
    paymentMethods,
    subscriptions,
    t,
  })

  const currencySymbolMap = useMemo(
    () => new Map(userCurrencies.map((item) => [item.code.toUpperCase(), item.symbol.trim()] as const)),
    [userCurrencies]
  )
  const paymentMethodIconMap = useMemo(
    () => new Map(paymentMethods.map((item) => [item.id, item.icon] as const)),
    [paymentMethods]
  )

  useEffect(() => {
    const handleStorage = (event: StorageEvent) => {
      if (event.key === DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY) {
        setDisplayAllAmountsInPrimaryCurrency(getDisplayAllAmountsInPrimaryCurrency())
      }
      if (event.key === DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY) {
        setDisplayRecurringAmountsAsMonthlyCost(getDisplayRecurringAmountsAsMonthlyCost())
      }
      if (event.key === DISPLAY_SUBSCRIPTION_CYCLE_PROGRESS_KEY) {
        setDisplaySubscriptionCycleProgress(getDisplaySubscriptionCycleProgress())
      }
    }
    window.addEventListener("storage", handleStorage)
    return () => window.removeEventListener("storage", handleStorage)
  }, [])

  useEffect(() => {
    if (!displayAllAmountsInPrimaryCurrency) {
      return
    }

    const targetCurrency = preferredCurrency.toUpperCase()
    const sourceCurrencies = Array.from(
      new Set(
        subscriptions
          .map((sub) => sub.currency.toUpperCase())
          .filter((currency) => currency && currency !== targetCurrency)
      )
    )

    if (sourceCurrencies.length === 0) {
      return
    }

    let active = true
    getExchangeRatesToTarget(sourceCurrencies, targetCurrency)
      .then((ratesBySource) => {
        if (!active) {
          return
        }
        const nextRates: Record<string, number> = {}
        for (const [sourceCurrency, rate] of Object.entries(ratesBySource)) {
          nextRates[`${sourceCurrency}->${targetCurrency}`] = rate
        }
        setExchangeRates((prev) => ({ ...prev, ...nextRates }))
      })
      .catch(() => {
        void 0
      })

    return () => {
      active = false
    }
  }, [displayAllAmountsInPrimaryCurrency, preferredCurrency, subscriptions])

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
                currencySymbol={currencySymbolMap.get((summary.currency || preferredCurrency).toUpperCase())}
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
              onToggleNoCategory={onToggleNoCategory}
              onTogglePaymentMethod={handleTogglePaymentMethod}
              onToggleNoPaymentMethod={onToggleNoPaymentMethod}
              subscriptionView={subscriptionView}
              onToggleSubscriptionView={() =>
                setSubscriptionView((current) => (current === "list" ? "cards" : "list"))
              }
              viewToggleDisabled={subscriptions.length === 0}
            />

            <div
              className={
                subscriptionView === "list"
                  ? "space-y-1"
                  : "grid auto-rows-min items-start grid-cols-1 gap-1.5 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4"
              }
            >
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
                filteredSubscriptions.map((sub) => {
                  const targetCurrency = preferredCurrency.toUpperCase()
                  const sourceCurrency = sub.currency.toUpperCase()
                  const shouldConvert =
                    displayAllAmountsInPrimaryCurrency && sourceCurrency !== targetCurrency
                  const conversionRate = shouldConvert
                    ? exchangeRates[`${sourceCurrency}->${targetCurrency}`]
                    : undefined
                  const displayAmount = conversionRate ? sub.amount * conversionRate : undefined
                  const displayCurrency = conversionRate ? targetCurrency : undefined
                  const displayCurrencySymbol = displayCurrency
                    ? currencySymbolMap.get(displayCurrency)
                    : undefined
                  const monthlyFactor = displayRecurringAmountsAsMonthlyCost
                    ? getMonthlyAmountFactor(sub)
                    : null
                  const monthlyDisplayAmount = monthlyFactor
                    ? (displayAmount ?? sub.amount) * monthlyFactor
                    : undefined

                  if (subscriptionView === "list") {
                    return (
                      <SubscriptionCard
                        key={sub.id}
                        subscription={sub}
                        categoryName={getSubscriptionCategoryName(sub)}
                        currencySymbol={currencySymbolMap.get(sub.currency.toUpperCase())}
                        displayAmount={monthlyDisplayAmount ?? displayAmount}
                        displayCurrency={displayCurrency}
                        displayCurrencySymbol={displayCurrencySymbol}
                        showMonthlyAmount={monthlyFactor !== null}
                        showCycleProgress={displaySubscriptionCycleProgress}
                        paymentMethodName={
                          sub.payment_method_id
                            ? paymentMethodLabelMap.get(sub.payment_method_id)
                            : undefined
                        }
                        paymentMethodIcon={
                          sub.payment_method_id
                            ? paymentMethodIconMap.get(sub.payment_method_id)
                            : undefined
                        }
                        onEdit={handleEdit}
                        onDelete={handleDelete}
                      />
                    )
                  }

                  return (
                    <SubscriptionSquareCard
                      key={sub.id}
                      subscription={sub}
                      categoryName={getSubscriptionCategoryName(sub)}
                      currencySymbol={currencySymbolMap.get(sub.currency.toUpperCase())}
                      displayAmount={monthlyDisplayAmount ?? displayAmount}
                      displayCurrency={displayCurrency}
                      displayCurrencySymbol={displayCurrencySymbol}
                      showMonthlyAmount={monthlyFactor !== null}
                      showCycleProgress={displaySubscriptionCycleProgress}
                      paymentMethodName={
                        sub.payment_method_id
                          ? paymentMethodLabelMap.get(sub.payment_method_id)
                          : undefined
                      }
                      onEdit={handleEdit}
                    />
                  )
                })
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
