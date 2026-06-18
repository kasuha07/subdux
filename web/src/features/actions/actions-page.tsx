import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import type { TFunction } from "i18next"
import {
  AlertTriangle,
  ArrowLeft,
  BellOff,
  CalendarClock,
  CheckCircle2,
  ChevronRight,
  Clock3,
  CreditCard,
  Eye,
  History,
  Pencil,
  RefreshCw,
  TrendingUp,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api"
import { getBrandIconFromValue } from "@/lib/brand-icons"
import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import { cn, formatCurrencyWithSymbol, formatDate } from "@/lib/utils"
import {
  invalidateSubscriptionDetail,
  preloadSubscriptionDetail,
} from "@/features/subscriptions/subscription-detail-cache"
import type {
  ActionCenter,
  Category,
  CreateSubscriptionInput,
  PaymentMethod,
  Subscription,
  SubscriptionAction,
  SubscriptionActionSeverity,
  SubscriptionActionType,
  UserCurrency,
} from "@/types"

const loadSubscriptionForm = () => import("@/features/subscriptions/subscription-form")
const SubscriptionForm = lazy(loadSubscriptionForm)
const loadSubscriptionDetailDrawer = () => import("@/features/subscriptions/subscription-detail-drawer")
const SubscriptionDetailDrawer = lazy(loadSubscriptionDetailDrawer)

const actionIconMap: Record<SubscriptionActionType, LucideIcon> = {
  upcoming_renewal: CalendarClock,
  manual_renewal_due: CheckCircle2,
  ending_soon: Clock3,
  notification_failed: BellOff,
  missing_next_billing: AlertTriangle,
  price_increase: TrendingUp,
}

const severityStyles: Record<SubscriptionActionSeverity, string> = {
  critical: "border-red-200 bg-red-500/10 text-red-700",
  high: "border-orange-200 bg-orange-500/10 text-orange-700",
  medium: "border-amber-200 bg-amber-500/10 text-amber-700",
  low: "border-sky-200 bg-sky-500/10 text-sky-700",
}

interface SubscriptionActionGroup {
  key: string
  primary: SubscriptionAction
  actions: SubscriptionAction[]
}

function ActionsSkeleton() {
  return (
    <div className="space-y-5">
      <div className="grid gap-3 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, index) => (
          <Card key={index}>
            <CardContent className="space-y-3 p-4">
              <Skeleton className="size-8 rounded-lg" />
              <Skeleton className="h-5 w-20" />
              <Skeleton className="h-3 w-28" />
            </CardContent>
          </Card>
        ))}
      </div>
      {Array.from({ length: 4 }).map((_, index) => (
        <Card key={index}>
          <CardContent className="flex items-center gap-4 p-4">
            <Skeleton className="size-11 rounded-lg" />
            <div className="min-w-0 flex-1 space-y-2">
              <Skeleton className="h-4 w-36" />
              <Skeleton className="h-3 w-64 max-w-full" />
            </div>
            <Skeleton className="h-8 w-24 rounded-md" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

function renderActionIcon(icon: string, name: string) {
  const fallback = (
    <span className="flex size-full items-center justify-center bg-muted text-sm font-semibold text-foreground">
      {name.charAt(0).toUpperCase()}
    </span>
  )

  if (!icon) {
    return fallback
  }

  const brand = getBrandIconFromValue(icon)
  if (brand) {
    const { Icon } = brand
    return <Icon size={22} color="default" />
  }

  if (icon.startsWith("http://") || icon.startsWith("https://") || icon.startsWith("/api/icon-proxy/")) {
    return <img src={icon} alt={name} className="h-7 w-7 object-contain" />
  }

  if (icon.startsWith("file:")) {
    const filename = icon.slice("file:".length)
    if (filename && !filename.includes("/") && !filename.includes("\\")) {
      return <img src={`/uploads/icons/${filename}`} alt={name} className="h-7 w-7 object-contain" />
    }
  }

  if (icon.includes(":")) {
    return fallback
  }

  return <span className="text-lg leading-none">{icon}</span>
}

export default function ActionsPage() {
  const { t, i18n } = useTranslation()
  const [center, setCenter] = useState<ActionCenter | null>(null)
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [busyKey, setBusyKey] = useState<string | null>(null)
  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [detailSub, setDetailSub] = useState<Subscription | null>(null)

  const currencySymbolMap = useMemo(
    () => new Map(userCurrencies.map((item) => [item.code.toUpperCase(), item.symbol.trim()] as const)),
    [userCurrencies]
  )

  const categoryMap = useMemo(
    () => new Map(categories.map((item) => [item.id, item] as const)),
    [categories]
  )

  const paymentMethodLabelMap = useMemo(
    () => new Map(paymentMethods.map((item) => [item.id, getPaymentMethodLabel(item, t)] as const)),
    [paymentMethods, t]
  )

  const subscriptionMap = useMemo(
    () => new Map(subscriptions.map((item) => [item.id, item] as const)),
    [subscriptions]
  )

  const fetchData = useCallback(async () => {
    const [actionCenter, subs, currencies, categoryList, methods] = await Promise.all([
      api.get<ActionCenter>("/actions"),
      api.get<Subscription[]>("/subscriptions"),
      api.get<UserCurrency[]>("/currencies"),
      api.get<Category[]>("/categories"),
      api.get<PaymentMethod[]>("/payment-methods"),
    ])
    setCenter(actionCenter)
    setSubscriptions(subs ?? [])
    setUserCurrencies(currencies ?? [])
    setCategories(categoryList ?? [])
    setPaymentMethods(methods ?? [])
  }, [])

  useEffect(() => {
    fetchData()
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [fetchData])

  async function handleRefresh() {
    setRefreshing(true)
    try {
      await fetchData()
    } catch {
      toast.error(t("actions.error.description"))
    } finally {
      setRefreshing(false)
    }
  }

  function subscriptionForAction(action: SubscriptionAction): Subscription | null {
    return subscriptionMap.get(action.subscription_id) ?? null
  }

  function getSubscriptionCategoryName(sub: Subscription): string {
    if (sub.category_id != null) {
      const category = categoryMap.get(sub.category_id)
      if (category) {
        return getCategoryLabel(category, t)
      }
    }
    return sub.category.trim()
  }

  function openEdit(action: SubscriptionAction) {
    const sub = subscriptionForAction(action)
    if (!sub) return
    void loadSubscriptionForm()
    setEditingSub(sub)
  }

  function openDetail(action: SubscriptionAction) {
    const sub = subscriptionForAction(action)
    if (!sub) return
    void loadSubscriptionDetailDrawer()
    preloadSubscriptionDetail(sub.id)
    setDetailSub(sub)
  }

  async function refreshAfterAction(subscriptionID?: number) {
    if (subscriptionID) {
      invalidateSubscriptionDetail(subscriptionID)
    }
    await fetchData()
  }

  async function handleMarkRenewed(action: SubscriptionAction) {
    const sub = subscriptionForAction(action)
    if (!sub) {
      toast.error(t("actions.error.subscriptionMissing"))
      return
    }
    setBusyKey(action.key)
    try {
      await api.post<Subscription>(`/subscriptions/${action.subscription_id}/mark-renewed`, {})
      toast.success(t("actions.toast.markRenewed"))
      await refreshAfterAction(action.subscription_id)
    } catch {
      toast.error(t("actions.error.actionFailed"))
    } finally {
      setBusyKey(null)
    }
  }

  async function handleCancelAtPeriodEnd(action: SubscriptionAction) {
    const sub = subscriptionForAction(action)
    if (!sub) {
      toast.error(t("actions.error.subscriptionMissing"))
      return
    }
    if (!sub.next_billing_date) {
      toast.error(t("actions.error.missingNextBilling"))
      return
    }
    setBusyKey(action.key)
    try {
      await api.put<Subscription>(`/subscriptions/${action.subscription_id}`, {
        renewal_mode: "cancel_at_period_end",
        ends_at: sub.next_billing_date,
      })
      toast.success(t("actions.toast.cancelAtPeriodEnd"))
      await refreshAfterAction(action.subscription_id)
    } catch {
      toast.error(t("actions.error.actionFailed"))
    } finally {
      setBusyKey(null)
    }
  }

  async function handleKeepSubscription(action: SubscriptionAction) {
    const sub = subscriptionForAction(action)
    if (!sub) {
      toast.error(t("actions.error.subscriptionMissing"))
      return
    }
    setBusyKey(action.key)
    try {
      await api.put<Subscription>(`/subscriptions/${action.subscription_id}`, {
        renewal_mode: "auto_renew",
      })
      toast.success(t("actions.toast.keepSubscription"))
      await refreshAfterAction(action.subscription_id)
    } catch {
      toast.error(t("actions.error.actionFailed"))
    } finally {
      setBusyKey(null)
    }
  }

  async function handleSnooze(group: SubscriptionActionGroup) {
    setBusyKey(group.key)
    try {
      await Promise.all(group.actions.map((action) => api.post("/actions/snooze", { key: action.key, days: 7 })))
      toast.success(t("actions.toast.snoozed"))
      await fetchData()
    } catch {
      toast.error(t("actions.error.actionFailed"))
    } finally {
      setBusyKey(null)
    }
  }

  async function handleFormSubmit(data: CreateSubscriptionInput) {
    if (!editingSub) {
      throw new Error("missing subscription")
    }

    const updatePayload = {
      ...data,
      payment_method_id: data.payment_method_id ?? 0,
    }
    const updated = await api.put<Subscription>(`/subscriptions/${editingSub.id}`, updatePayload)
    toast.success(t("dashboard.updateSuccess"))
    setEditingSub(null)
    await refreshAfterAction(editingSub.id)
    return updated
  }

  async function handleFormMarkRenewed(sub: Subscription) {
    const renewed = await api.post<Subscription>(`/subscriptions/${sub.id}/mark-renewed`, {})
    toast.success(t("actions.toast.markRenewed"))
    setEditingSub(null)
    await refreshAfterAction(sub.id)
    return renewed
  }

  const counts = center?.counts
  const actionGroups = useMemo(() => groupSubscriptionActions(center?.items ?? []), [center?.items])

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/" aria-label={t("actions.nav.back")} title={t("actions.nav.back")}>
                <ArrowLeft className="size-4" />
              </Link>
            </Button>
            <h1 className="text-lg font-bold tracking-tight">{t("actions.title")}</h1>
          </div>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => void handleRefresh()}
            disabled={loading || refreshing}
            aria-label={t("actions.nav.refresh")}
            title={t("actions.nav.refresh")}
          >
            <RefreshCw className={cn("size-4", refreshing && "animate-spin")} />
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-5xl space-y-4 px-4 py-4 sm:space-y-6 sm:py-6">
        {loading ? (
          <ActionsSkeleton />
        ) : !center ? (
          <EmptyState
            title={t("actions.error.title")}
            description={t("actions.error.description")}
          />
        ) : (
          <>
            <section className="grid grid-cols-3 gap-2 sm:gap-3">
              <SummaryCard
                icon={AlertTriangle}
                label={t("actions.summary.total")}
                value={String(actionGroups.length)}
                detail={t("actions.summary.grouped", {
                  actions: counts?.total ?? 0,
                  snoozed: counts?.snoozed ?? 0,
                })}
              />
              <SummaryCard
                icon={CreditCard}
                label={t("actions.summary.upcoming")}
                value={String(counts?.upcoming_charge ?? 0)}
                detail={t("actions.summary.window", { urgent: center.urgent_days, days: center.window_days })}
              />
              <SummaryCard
                icon={History}
                label={t("actions.summary.repair")}
                value={String(counts?.needs_repair ?? 0)}
                detail={t("actions.summary.decision", { count: counts?.needs_decision ?? 0 })}
              />
            </section>

            {actionGroups.length === 0 ? (
              <EmptyState
                title={t("actions.empty.title")}
                description={t("actions.empty.description")}
              />
            ) : (
              <section className="space-y-3">
                {actionGroups.map((group) => (
                  <ActionGroupItem
                    key={group.key}
                    group={group}
                    busy={busyKey === group.key || group.actions.some((action) => busyKey === action.key)}
                    currencySymbol={currencySymbolMap.get(group.primary.currency.toUpperCase())}
                    language={i18n.language}
                    onMarkRenewed={handleMarkRenewed}
                    onCancelAtPeriodEnd={handleCancelAtPeriodEnd}
                    onKeepSubscription={handleKeepSubscription}
                    onSnooze={handleSnooze}
                    onEdit={openEdit}
                    onOpenDetail={openDetail}
                  />
                ))}
              </section>
            )}
          </>
        )}
      </main>

      {editingSub && (
        <Suspense fallback={null}>
          <SubscriptionForm
            key={editingSub.id}
            open={!!editingSub}
            onOpenChange={(open) => {
              if (!open) setEditingSub(null)
            }}
            subscription={editingSub}
            onSubmit={handleFormSubmit}
            onMarkRenewed={handleFormMarkRenewed}
            userCurrencies={userCurrencies}
            categories={categories}
            paymentMethods={paymentMethods}
          />
        </Suspense>
      )}

      {detailSub && (
        <Suspense fallback={null}>
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
            onEdit={(sub) => {
              setDetailSub(null)
              setEditingSub(sub)
            }}
          />
        </Suspense>
      )}
    </div>
  )
}

function groupSubscriptionActions(items: SubscriptionAction[]): SubscriptionActionGroup[] {
  const groups: SubscriptionActionGroup[] = []
  const bySubscription = new Map<number, SubscriptionActionGroup>()

  for (const item of items) {
    const existing = bySubscription.get(item.subscription_id)
    if (existing) {
      existing.actions.push(item)
      continue
    }

    const group: SubscriptionActionGroup = {
      key: `subscription-action-group:${item.subscription_id}`,
      primary: item,
      actions: [item],
    }
    bySubscription.set(item.subscription_id, group)
    groups.push(group)
  }

  return groups
}

function SummaryCard({
  detail,
  icon: Icon,
  label,
  value,
}: {
  detail: string
  icon: LucideIcon
  label: string
  value: string
}) {
  return (
    <Card className="py-0">
      <CardContent className="px-2 py-1.5 sm:p-4">
        <div className="flex items-start justify-between gap-2 sm:gap-3">
          <div className="min-w-0">
            <p className="truncate text-[11px] font-medium text-muted-foreground sm:text-sm">{label}</p>
            <p className="mt-0.5 text-base font-semibold tabular-nums sm:mt-2 sm:text-2xl">{value}</p>
            <p className="mt-0.5 truncate text-[10px] text-muted-foreground sm:mt-1 sm:text-xs">{detail}</p>
          </div>
          <div className="hidden rounded-lg bg-primary/10 p-2 text-primary sm:block">
            <Icon className="size-4" />
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function ActionGroupItem({
  group,
  busy,
  currencySymbol,
  language,
  onCancelAtPeriodEnd,
  onEdit,
  onKeepSubscription,
  onMarkRenewed,
  onOpenDetail,
  onSnooze,
}: {
  group: SubscriptionActionGroup
  busy: boolean
  currencySymbol?: string
  language: string
  onCancelAtPeriodEnd: (action: SubscriptionAction) => void | Promise<void>
  onEdit: (action: SubscriptionAction) => void
  onKeepSubscription: (action: SubscriptionAction) => void | Promise<void>
  onMarkRenewed: (action: SubscriptionAction) => void | Promise<void>
  onOpenDetail: (action: SubscriptionAction) => void
  onSnooze: (group: SubscriptionActionGroup) => void | Promise<void>
}) {
  const { t } = useTranslation()
  const primary = group.primary
  const amount = formatCurrencyWithSymbol(primary.amount, primary.currency, currencySymbol, language)
  const markRenewedAction = findAllowedAction(group.actions, "mark_renewed")
  const keepSubscriptionAction = findActionByType(group.actions, "ending_soon")
  const cancelAtPeriodEndAction = primary.renewal_mode === "cancel_at_period_end"
    ? null
    : findAllowedAction(group.actions, "cancel_at_period_end")

  return (
    <Card className="overflow-hidden py-0">
      <CardContent className="p-0">
        <div className="grid gap-0 sm:grid-cols-[minmax(0,1fr)_auto]">
          <button
            type="button"
            className="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-3 p-4 text-left transition-colors hover:bg-muted/40"
            onClick={() => onOpenDetail(primary)}
          >
            <div className="flex size-11 items-center justify-center overflow-hidden rounded-lg border bg-muted/30">
              {renderActionIcon(primary.subscription_icon, primary.subscription_name)}
            </div>
            <div className="min-w-0">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                {group.actions.map((action) => (
                  <ActionTypeBadge key={action.key} action={action} />
                ))}
              </div>
              <h2 className="mt-2 truncate text-sm font-semibold">{primary.subscription_name}</h2>
              <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
                <span>{amount}</span>
                {group.actions.length > 1 ? <span>{t("actions.group.itemCount", { count: group.actions.length })}</span> : null}
              </div>
              <div className="mt-3 space-y-2">
                {group.actions.map((action) => (
                  <ActionSummaryLine
                    key={action.key}
                    action={action}
                    currencySymbol={currencySymbol}
                    language={language}
                  />
                ))}
              </div>
            </div>
          </button>

          <div className="flex flex-wrap items-center gap-2 border-t p-3 sm:w-48 sm:flex-col sm:items-stretch sm:border-t-0 sm:border-l">
            {markRenewedAction ? (
              <Button size="sm" onClick={() => void onMarkRenewed(markRenewedAction)} disabled={busy}>
                <CheckCircle2 className="size-4" />
                {t("actions.command.markRenewed")}
              </Button>
            ) : null}
            {keepSubscriptionAction ? (
              <Button size="sm" onClick={() => void onKeepSubscription(keepSubscriptionAction)} disabled={busy}>
                <RefreshCw className="size-4" />
                {t("actions.command.keepSubscription")}
              </Button>
            ) : null}
            {cancelAtPeriodEndAction ? (
              <Button size="sm" onClick={() => void onCancelAtPeriodEnd(cancelAtPeriodEndAction)} disabled={busy}>
                <Clock3 className="size-4" />
                {t("actions.command.cancelAtPeriodEnd")}
              </Button>
            ) : null}
            <Button size="sm" variant="outline" onClick={() => onEdit(primary)} disabled={busy}>
              <Pencil className="size-4" />
              {t("subscription.detail.edit")}
            </Button>
            <Button size="sm" variant="outline" onClick={() => onOpenDetail(primary)} disabled={busy}>
              <Eye className="size-4" />
              {t("subscription.detail.open")}
            </Button>
            <Button size="sm" variant="outline" onClick={() => void onSnooze(group)} disabled={busy}>
              <Clock3 className="size-4" />
              {t("actions.command.snooze")}
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function ActionTypeBadge({ action }: { action: SubscriptionAction }) {
  const { t } = useTranslation()
  const Icon = actionIconMap[action.type]

  return (
    <Badge variant="outline" className={cn("gap-1", severityStyles[action.severity])}>
      <Icon className="size-3.5" />
      {t(`actions.type.${action.type}`)}
    </Badge>
  )
}

function ActionSummaryLine({
  action,
  currencySymbol,
  language,
}: {
  action: SubscriptionAction
  currencySymbol?: string
  language: string
}) {
  const { t } = useTranslation()
  const dueText = formatActionDate(action, t, language)
  const priceDelta = action.delta_monthly_amount != null
    ? formatCurrencyWithSymbol(action.delta_monthly_amount, action.currency, currencySymbol, language)
    : ""

  return (
    <div className="rounded-lg border bg-background/70 p-2">
      <p className="line-clamp-2 text-sm text-muted-foreground">
        {t(`actions.message.${action.type}`)}
      </p>
      <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
        <span>{dueText}</span>
        {action.notification_channel ? <span>{action.notification_channel}</span> : null}
        {priceDelta ? <span>{t("actions.priceDelta", { amount: priceDelta })}</span> : null}
      </div>
      {action.notification_error ? (
        <p className="mt-1 truncate text-xs text-destructive" title={action.notification_error}>
          {action.notification_error}
        </p>
      ) : null}
    </div>
  )
}

function findAllowedAction(actions: SubscriptionAction[], actionName: string): SubscriptionAction | null {
  return actions.find((action) => action.allowed_actions.includes(actionName)) ?? null
}

function findActionByType(actions: SubscriptionAction[], actionType: SubscriptionActionType): SubscriptionAction | null {
  return actions.find((action) => action.type === actionType) ?? null
}

function EmptyState({ title, description }: { title: string, description: string }) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center justify-center px-4 py-14 text-center">
        <div className="mb-4 rounded-full bg-muted p-4">
          <ChevronRight className="size-6 text-muted-foreground" />
        </div>
        <h2 className="font-medium">{title}</h2>
        <p className="mt-1 max-w-md text-sm text-muted-foreground">{description}</p>
      </CardContent>
    </Card>
  )
}

function formatActionDate(
  action: SubscriptionAction,
  t: TFunction<"translation", undefined>,
  language: string
): string {
  if (action.due_date) {
    if (action.days_until === 0) {
      return t("actions.date.today")
    }
    if (action.days_until !== null && action.days_until < 0) {
      return t("actions.date.overdue", { count: Math.abs(action.days_until) })
    }
    if (action.days_until !== null) {
      return t("actions.date.inDays", {
        count: action.days_until,
        date: formatDate(action.due_date, language),
      })
    }
    return formatDate(action.due_date, language)
  }
  if (action.event_date) {
    return t("actions.date.event", { date: formatDate(action.event_date, language) })
  }
  return t("actions.date.none")
}
