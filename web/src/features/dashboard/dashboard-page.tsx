import { useState, useEffect, useCallback } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Skeleton } from "@/components/ui/skeleton"
import { api, isAdmin } from "@/lib/api"
import { formatCurrency } from "@/lib/utils"
import { toast } from "sonner"
import type { Subscription, DashboardSummary, CreateSubscriptionInput } from "@/types"
import SubscriptionCard from "@/features/subscriptions/subscription-card"
import SubscriptionForm from "@/features/subscriptions/subscription-form"
import { Plus, Settings, DollarSign, CalendarDays, Repeat, TrendingUp, Shield } from "lucide-react"

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

  const fetchData = useCallback(async () => {
    try {
      const [subs, sum] = await Promise.all([
        api.get<Subscription[]>("/subscriptions"),
        api.get<DashboardSummary>("/dashboard/summary"),
      ])
      setSubscriptions(subs || [])
      setSummary(sum)
    } catch {
      void 0
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

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
      await api.put(`/subscriptions/${editingSub.id}`, data)
      toast.success(t("dashboard.updateSuccess"))
    } else {
      await api.post("/subscriptions", data)
      toast.success(t("dashboard.createSuccess"))
    }
    setEditingSub(null)
    setFormOpen(false)
    await fetchData()
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
          <div className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <DollarSign className="size-4" />
                  <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.monthly")}</span>
                </div>
                <p className="mt-1 text-2xl font-bold tabular-nums">
                  {formatCurrency(summary.total_monthly, localStorage.getItem("defaultCurrency") || "USD", i18n.language)}
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
                  {formatCurrency(summary.total_yearly, localStorage.getItem("defaultCurrency") || "USD", i18n.language)}
                </p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Repeat className="size-4" />
                  <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.active")}</span>
                </div>
                <p className="mt-1 text-2xl font-bold tabular-nums">{summary.active_count}</p>
              </CardContent>
            </Card>
            <Card>
              <CardContent className="p-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <CalendarDays className="size-4" />
                  <span className="text-xs font-medium uppercase tracking-wider">{t("dashboard.stats.upcoming")}</span>
                </div>
                <p className="mt-1 text-2xl font-bold tabular-nums">
                  {summary.upcoming_renewals?.length || 0}
                </p>
              </CardContent>
            </Card>
          </div>
        )}

        <Separator className="mb-6" />

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
          ) : (
            subscriptions.map((sub) => (
              <SubscriptionCard
                key={sub.id}
                subscription={sub}
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
      />
    </div>
  )
}
