import { Suspense, lazy, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { BarChart3, CalendarDays, ListChecks, Plus, Settings, Shield } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { useDashboardData } from "@/features/dashboard/hooks/use-dashboard-data"
import { useDashboardFilters } from "@/features/dashboard/hooks/use-dashboard-filters"
import { getMonthlyAmountFactor } from "@/features/dashboard/dashboard-amount-utils"
import {
  getSubscriptionEndsAt,
  getSubscriptionRenewalMode,
  isSubscriptionActive,
} from "@/features/subscriptions/subscription-lifecycle"
import {
  invalidateSubscriptionDetail,
  preloadSubscriptionDetail,
} from "@/features/subscriptions/subscription-detail-cache"
import { api, isAdmin } from "@/lib/api"
import { formatCurrencyWithSymbol, formatDate } from "@/lib/utils"
import {
  DISPLAY_ALL_AMOUNTS_IN_PRIMARY_CURRENCY_KEY,
  DISPLAY_DISABLED_SUBSCRIPTIONS_LAST_KEY,
  DISPLAY_RECURRING_AMOUNTS_AS_MONTHLY_COST_KEY,
  DISPLAY_SUBSCRIPTION_CYCLE_PROGRESS_KEY,
  getDisplayAllAmountsInPrimaryCurrency,
  getDisplayDisabledSubscriptionsLast,
  getDisplayRecurringAmountsAsMonthlyCost,
  getDisplaySubscriptionCycleProgress,
} from "@/lib/display-preferences"
import { getExchangeRatesToTarget } from "@/lib/exchange-rate-cache"
import { toast } from "sonner"
import type { CreateSubscriptionInput, Subscription } from "@/types"

import SubscriptionCard from "@/features/subscriptions/subscription-card"
import SubscriptionSquareCard from "@/features/subscriptions/subscription-square-card"
import DashboardFiltersToolbar from "./dashboard-filters-toolbar"
import DashboardSummaryCards from "./dashboard-summary-cards"

const loadSubscriptionForm = () => import("@/features/subscriptions/subscription-form")
const SubscriptionForm = lazy(loadSubscriptionForm)
const loadSubscriptionDetailDrawer = () => import("@/features/subscriptions/subscription-detail-drawer")
const SubscriptionDetailDrawer = lazy(loadSubscriptionDetailDrawer)

function preloadSubscriptionForm() {
  void loadSubscriptionForm()
}

function preloadSubscriptionDetailDrawer() {
  void loadSubscriptionDetailDrawer()
}

function DashboardSkeleton() {
  return (
    <>
      <Card className="mb-6 overflow-hidden py-0">
        <CardContent className="grid gap-px p-0 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
          <div className="space-y-4 p-6 sm:p-7">
            <div className="flex items-center gap-3">
              <Skeleton className="size-11 rounded-2xl" />
              <Skeleton className="h-4 w-32" />
            </div>
            <Skeleton className="h-11 w-56" />
            <div className="flex flex-wrap gap-3">
              <Skeleton className="h-10 w-32 rounded-full" />
              <Skeleton className="h-10 w-32 rounded-full" />
            </div>
          </div>

          <div className="grid gap-px sm:grid-cols-2 lg:grid-cols-1 xl:grid-cols-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="space-y-3 p-4 sm:p-5">
                <div className="flex items-center gap-3">
                  <Skeleton className="size-8 rounded-xl" />
                  <div className="space-y-2">
                    <Skeleton className="h-3 w-20" />
                    <Skeleton className="h-5 w-24" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>

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

function SubscriptionFormFallbackDialog({
  description,
  onOpenChange,
  open,
  title,
}: {
  description: string
  onOpenChange: (open: boolean) => void
  open: boolean
  title: string
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="flex max-h-[calc(100vh-1.5rem)] max-w-2xl flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]"
        onInteractOutside={(event) => event.preventDefault()}
        onPointerDownOutside={(event) => event.preventDefault()}
      >
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription className="sr-only">{description}</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-5 py-4 sm:px-6">
          <div className="grid grid-cols-[auto_minmax(0,1fr)] items-end gap-3">
            <div className="space-y-2">
              <Skeleton className="h-4 w-14" />
              <Skeleton className="size-9 rounded-md" />
            </div>
            <div className="min-w-0 space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
          </div>
          <div className="grid grid-cols-[minmax(0,1fr)_minmax(7rem,0.9fr)] gap-3 sm:grid-cols-2">
            <div className="space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-14" />
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
          </div>
          <div className="grid grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] gap-3 sm:grid-cols-2">
            <div className="space-y-2">
              <Skeleton className="h-4 w-16" />
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
            <div className="space-y-2">
              <Skeleton className="h-4 w-20" />
              <Skeleton className="h-9 w-full rounded-md" />
            </div>
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-9 w-full rounded-md" />
          </div>
          <div className="grid gap-3 sm:grid-cols-2">
            <Skeleton className="h-9 w-full rounded-md" />
            <Skeleton className="h-9 w-full rounded-md" />
          </div>
        </div>
        <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <Skeleton className="h-9 w-full rounded-md sm:w-24" />
            <Skeleton className="h-9 w-full rounded-md sm:ml-auto sm:w-28" />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function SubscriptionDetailFallbackDrawer({
  currencySymbol,
  onOpenChange,
  open,
  subscription,
}: {
  currencySymbol?: string
  onOpenChange: (open: boolean) => void
  open: boolean
  subscription: Subscription
}) {
  const { t, i18n } = useTranslation()
  const amount = formatCurrencyWithSymbol(
    subscription.amount,
    subscription.currency,
    currencySymbol,
    i18n.language
  )
  const status = subscription.status || "active"
  const renewalMode = getSubscriptionRenewalMode(subscription)
  const periodEndDate = getSubscriptionEndsAt(subscription)
  const isEnding = renewalMode === "cancel_at_period_end" && periodEndDate
  const nextSummaryDate = isEnding ? periodEndDate : subscription.next_billing_date

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="fixed top-0 right-0 left-auto flex h-dvh max-h-dvh w-full max-w-full translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-r-0 p-0 duration-300 sm:max-w-full md:max-w-xl data-[state=open]:slide-in-from-right data-[state=closed]:slide-out-to-right data-[state=open]:zoom-in-100 data-[state=closed]:zoom-out-100">
        <DialogHeader className="detail-drawer-stage border-b px-5 pt-5 pb-4 sm:px-6">
          <div className="flex items-start justify-between gap-4 pr-8">
            <div className="min-w-0">
              <DialogTitle className="truncate">
                {subscription.name || t("subscription.detail.titleFallback")}
              </DialogTitle>
              <DialogDescription className="sr-only">
                {t("subscription.detail.description")}
              </DialogDescription>
              <p className="mt-1 text-sm text-muted-foreground">{amount}</p>
            </div>
            <Skeleton className="h-9 w-20 shrink-0 rounded-md" />
          </div>
        </DialogHeader>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <div className="space-y-5 px-5 py-5 sm:px-6">
            <div className="grid gap-3 sm:grid-cols-3">
              <div className="detail-drawer-stage rounded-lg border bg-muted/25 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  {isEnding ? t("subscription.detail.summary.periodEnd") : t("subscription.detail.summary.nextCharge")}
                </p>
                <p className="mt-2 truncate text-sm font-semibold">
                  {nextSummaryDate
                    ? formatDate(nextSummaryDate, i18n.language)
                    : t("subscription.detail.empty.none")}
                </p>
                <Skeleton className="mt-2 h-3 w-28 rounded-md" />
              </div>
              <div className="detail-drawer-stage rounded-lg border bg-muted/25 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  {t("subscription.detail.summary.lifecycle")}
                </p>
                <p className="mt-2 truncate text-sm font-semibold">
                  {t(`subscription.card.status.${status}`)}
                </p>
                <p className="mt-1 truncate text-xs text-muted-foreground">
                  {t(`subscription.card.renewalMode.${renewalMode}`)}
                </p>
              </div>
              <div className="detail-drawer-stage rounded-lg border bg-muted/25 p-3">
                <p className="text-xs font-medium text-muted-foreground">
                  {t("subscription.detail.summary.latestActivity")}
                </p>
                <Skeleton className="mt-2 h-4 w-24 rounded-md" />
                <Skeleton className="mt-2 h-3 w-32 rounded-md" />
              </div>
            </div>
            <Skeleton className="detail-drawer-stage h-10 rounded-lg" />
            <Skeleton className="detail-drawer-stage h-64 rounded-lg" />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export default function DashboardPage() {
  const { t, i18n } = useTranslation()
  const [subscriptionView, setSubscriptionView] = useState<"list" | "cards">("list")
  const [formOpen, setFormOpen] = useState(false)
  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [detailSub, setDetailSub] = useState<Subscription | null>(null)
  const [displayAllAmountsInPrimaryCurrency, setDisplayAllAmountsInPrimaryCurrency] = useState(
    getDisplayAllAmountsInPrimaryCurrency()
  )
  const [displayRecurringAmountsAsMonthlyCost, setDisplayRecurringAmountsAsMonthlyCost] = useState(
    getDisplayRecurringAmountsAsMonthlyCost()
  )
  const [displaySubscriptionCycleProgress, setDisplaySubscriptionCycleProgress] = useState(
    getDisplaySubscriptionCycleProgress()
  )
  const [displayDisabledSubscriptionsLast, setDisplayDisabledSubscriptionsLast] = useState(
    getDisplayDisabledSubscriptionsLast()
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
    handleToggleRenewalMode,
    handleToggleStatus,
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
    selectedPaymentMethodIDs,
    selectedRenewalModes,
    selectedStatuses,
    setSearchTerm,
    sortDirection,
    sortField,
  } = useDashboardFilters({
    categories,
    displayDisabledSubscriptionsLast,
    exchangeRates,
    language: i18n.language,
    paymentMethods,
    preferredCurrency,
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
      if (event.key === DISPLAY_DISABLED_SUBSCRIPTIONS_LAST_KEY) {
        setDisplayDisabledSubscriptionsLast(getDisplayDisabledSubscriptionsLast())
      }
    }
    window.addEventListener("storage", handleStorage)
    return () => window.removeEventListener("storage", handleStorage)
  }, [])

  useEffect(() => {
    const timeout = window.setTimeout(preloadSubscriptionForm, 250)
    return () => window.clearTimeout(timeout)
  }, [])

  useEffect(() => {
    const timeout = window.setTimeout(preloadSubscriptionDetailDrawer, 400)
    return () => window.clearTimeout(timeout)
  }, [])

  useEffect(() => {
    if (!displayAllAmountsInPrimaryCurrency && sortField !== "amount") {
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
  }, [displayAllAmountsInPrimaryCurrency, preferredCurrency, sortField, subscriptions])

  function handleEdit(sub: Subscription) {
    preloadSubscriptionForm()
    setEditingSub(sub)
    setFormOpen(true)
  }

  function handleOpenDetail(sub: Subscription) {
    preloadSubscriptionDetailDrawer()
    preloadSubscriptionDetail(sub.id)
    setDetailSub(sub)
  }

  function handlePreloadDetail(sub: Subscription) {
    preloadSubscriptionDetailDrawer()
    preloadSubscriptionDetail(sub.id)
  }

  async function handleDelete(id: number) {
    if (!confirm(t("dashboard.deleteConfirm"))) return
    try {
      await api.delete(`/subscriptions/${id}`)
      invalidateSubscriptionDetail(id)
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
      invalidateSubscriptionDetail(editingSub.id)
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

  async function handleMarkRenewed(sub: Subscription) {
    const renewed = await api.post<Subscription>(`/subscriptions/${sub.id}/mark-renewed`, {})
    toast.success(t("dashboard.updateSuccess"))
    invalidateSubscriptionDetail(sub.id)
    setEditingSub(null)
    setFormOpen(false)
    await fetchData()
    return renewed
  }

  function openNewForm() {
    preloadSubscriptionForm()
    setEditingSub(null)
    setFormOpen(true)
  }

  function handleFormOpenChange(open: boolean) {
    setFormOpen(open)
    if (!open) setEditingSub(null)
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
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/actions" aria-label={t("actions.title")} title={t("actions.title")}>
                <ListChecks className="size-4" />
              </Link>
            </Button>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/calendar">
                <CalendarDays className="size-4" />
              </Link>
            </Button>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/reports">
                <BarChart3 className="size-4" />
              </Link>
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


            <DashboardFiltersToolbar
              searchTerm={searchTerm}
              onSearchTermChange={setSearchTerm}
              selectedStatuses={selectedStatuses}
              selectedCategories={selectedCategories}
              includeNoCategory={includeNoCategory}
              selectedPaymentMethodIDs={selectedPaymentMethodIDs}
              selectedRenewalModes={selectedRenewalModes}
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
              onToggleStatus={handleToggleStatus}
              onToggleCategory={handleToggleCategory}
              onToggleNoCategory={onToggleNoCategory}
              onTogglePaymentMethod={handleTogglePaymentMethod}
              onToggleRenewalMode={handleToggleRenewalMode}
              onToggleNoPaymentMethod={onToggleNoPaymentMethod}
              subscriptionView={subscriptionView}
              shownCount={filteredSubscriptions.length}
              totalCount={subscriptions.length}
              onToggleSubscriptionView={() =>
                setSubscriptionView((current) => (current === "list" ? "cards" : "list"))
              }
              viewToggleDisabled={subscriptions.length === 0}
            />

            <div
              className={
                subscriptionView === "list"
                  ? "space-y-3"
                  : "grid auto-rows-min items-start grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4"
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
                        showCycleProgress={displaySubscriptionCycleProgress && isSubscriptionActive(sub)}
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
                        onOpenDetail={handleOpenDetail}
                        onPreloadDetail={handlePreloadDetail}
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
                      showCycleProgress={displaySubscriptionCycleProgress && isSubscriptionActive(sub)}
                      paymentMethodName={
                        sub.payment_method_id
                          ? paymentMethodLabelMap.get(sub.payment_method_id)
                          : undefined
                      }
                      onOpenDetail={handleOpenDetail}
                      onPreloadDetail={handlePreloadDetail}
                    />
                  )
                })
              )}
            </div>
          </>
        )}
      </main>

      {formOpen && (
        <Suspense
          fallback={
            <SubscriptionFormFallbackDialog
              open={formOpen}
              onOpenChange={handleFormOpenChange}
              title={editingSub ? t("subscription.form.editTitle") : t("subscription.form.addTitle")}
              description={
                editingSub
                  ? t("subscription.form.editDescription")
                  : t("subscription.form.addDescription")
              }
            />
          }
        >
          <SubscriptionForm
            key={editingSub?.id ?? "new"}
            open={formOpen}
            onOpenChange={handleFormOpenChange}
            subscription={editingSub}
            onSubmit={handleFormSubmit}
            onMarkRenewed={handleMarkRenewed}
            userCurrencies={userCurrencies}
            categories={categories}
            paymentMethods={paymentMethods}
          />
        </Suspense>
      )}

      {detailSub && (
        <Suspense
          fallback={
            <SubscriptionDetailFallbackDrawer
              open={!!detailSub}
              subscription={detailSub}
              currencySymbol={currencySymbolMap.get(detailSub.currency.toUpperCase())}
              onOpenChange={(open) => {
                if (!open) setDetailSub(null)
              }}
            />
          }
        >
          <SubscriptionDetailDrawer
            open={!!detailSub}
            subscription={detailSub}
            categoryName={getSubscriptionCategoryName(detailSub)}
            currencySymbol={currencySymbolMap.get(detailSub.currency.toUpperCase())}
            paymentMethodName={
              detailSub.payment_method_id
                ? paymentMethodLabelMap.get(detailSub.payment_method_id)
                : undefined
            }
            onOpenChange={(open) => {
              if (!open) setDetailSub(null)
            }}
            onEdit={handleEdit}
          />
        </Suspense>
      )}
    </div>
  )
}
