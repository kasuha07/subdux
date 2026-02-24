import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ArrowLeft, Settings, ChevronLeft, ChevronRight, Copy, Plus, Trash2, Link2 } from "lucide-react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { api } from "@/lib/api"
import { cn } from "@/lib/utils"
import type { Category, CreateSubscriptionInput, PaymentMethod, Subscription, UserCurrency } from "@/types"
import SubscriptionForm from "@/features/subscriptions/subscription-form"

interface CalendarToken {
  id: number
  token: string
  name: string
  created_at: string
}

// Returns all dates (as YYYY-MM-DD strings) within the given year/month
// that a subscription bills on.
function getBillingDatesInMonth(sub: Subscription, year: number, month: number): Set<number> {
  const days = new Set<number>()
  const daysInMonth = new Date(year, month + 1, 0).getDate()

  if (sub.billing_type === "one_time") {
    if (!sub.next_billing_date) return days
    const d = new Date(sub.next_billing_date + "T00:00:00")
    if (d.getFullYear() === year && d.getMonth() === month) {
      days.add(d.getDate())
    }
    return days
  }

  // recurring
  if (sub.recurrence_type === "monthly_date" && sub.monthly_day !== null) {
    const day = sub.monthly_day
    if (day >= 1 && day <= daysInMonth) {
      days.add(day)
    } else if (day > daysInMonth) {
      // clamp to last day of month
      days.add(daysInMonth)
    }
    return days
  }

  if (sub.recurrence_type === "yearly_date" && sub.yearly_month !== null && sub.yearly_day !== null) {
    // yearly_month is 1-based
    if (sub.yearly_month - 1 === month) {
      const day = Math.min(sub.yearly_day, daysInMonth)
      days.add(day)
    }
    return days
  }

  if (sub.recurrence_type === "interval" && sub.next_billing_date) {
    const count = sub.interval_count ?? 1
    if (count <= 0) return days
    const unit = sub.interval_unit

    let cursor = new Date(sub.next_billing_date + "T00:00:00")
    const monthStart = new Date(year, month, 1)
    const monthEnd = new Date(year, month, daysInMonth)

    // Walk forward from next_billing_date until past end of month
    // Also walk backward in case next_billing_date is after the displayed month
    // First, rewind cursor to before or at monthStart
    while (cursor > monthEnd) {
      cursor = addInterval(cursor, -count, unit)
    }
    // Now advance until we pass monthEnd
    while (cursor <= monthEnd) {
      if (cursor >= monthStart) {
        days.add(cursor.getDate())
      }
      cursor = addInterval(cursor, count, unit)
    }
    return days
  }

  return days
}

function addInterval(date: Date, count: number, unit: string): Date {
  const d = new Date(date)
  switch (unit) {
    case "day":
      d.setDate(d.getDate() + count)
      break
    case "week":
      d.setDate(d.getDate() + count * 7)
      break
    case "month":
      d.setMonth(d.getMonth() + count)
      break
    case "year":
      d.setFullYear(d.getFullYear() + count)
      break
  }
  return d
}

function buildCalendarGrid(year: number, month: number): (number | null)[] {
  const firstDay = new Date(year, month, 1).getDay() // 0=Sun
  const daysInMonth = new Date(year, month + 1, 0).getDate()
  const cells: (number | null)[] = []
  for (let i = 0; i < firstDay; i++) cells.push(null)
  for (let d = 1; d <= daysInMonth; d++) cells.push(d)
  while (cells.length < 42) cells.push(null)
  return cells
}

export default function CalendarPage() {
  const { t, i18n } = useTranslation()
  const today = new Date()
  const [viewYear, setViewYear] = useState(today.getFullYear())
  const [viewMonth, setViewMonth] = useState(today.getMonth())
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [tokens, setTokens] = useState<CalendarToken[]>([])
  const [loadingSubs, setLoadingSubs] = useState(true)
  const [loadingTokens, setLoadingTokens] = useState(true)
  const [selectedDay, setSelectedDay] = useState<number | null>(null)
  const [newTokenName, setNewTokenName] = useState("")
  const [creatingToken, setCreatingToken] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [newlyCreatedUrl, setNewlyCreatedUrl] = useState<string | null>(null)
  const [editingSub, setEditingSub] = useState<Subscription | null>(null)
  const [formOpen, setFormOpen] = useState(false)
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])

  useEffect(() => {
    Promise.all([
      api.get<Subscription[]>("/subscriptions"),
      api.get<CalendarToken[]>("/calendar/tokens"),
      api.get<Category[]>("/categories"),
      api.get<PaymentMethod[]>("/payment-methods"),
      api.get<UserCurrency[]>("/currencies"),
    ]).then(([subs, toks, cats, methods, currencies]) => {
      setSubscriptions(subs || [])
      setTokens(toks || [])
      setCategories(cats || [])
      setPaymentMethods(methods || [])
      setUserCurrencies(currencies || [])
    }).catch(() => void 0).finally(() => {
      setLoadingSubs(false)
      setLoadingTokens(false)
    })
  }, [])

  // Map: day number -> subscriptions billing on that day
  const billingMap = useMemo(() => {
    const map = new Map<number, Subscription[]>()
    for (const sub of subscriptions.filter(s => s.enabled)) {
      const days = getBillingDatesInMonth(sub, viewYear, viewMonth)
      for (const day of days) {
        const existing = map.get(day) ?? []
        existing.push(sub)
        map.set(day, existing)
      }
    }
    return map
  }, [subscriptions, viewYear, viewMonth])

  const cells = useMemo(() => buildCalendarGrid(viewYear, viewMonth), [viewYear, viewMonth])

  function prevMonth() {
    if (viewMonth === 0) { setViewYear(y => y - 1); setViewMonth(11) }
    else setViewMonth(m => m - 1)
    setSelectedDay(null)
  }

  function nextMonth() {
    if (viewMonth === 11) { setViewYear(y => y + 1); setViewMonth(0) }
    else setViewMonth(m => m + 1)
    setSelectedDay(null)
  }

  const monthLabel = new Date(viewYear, viewMonth, 1).toLocaleDateString(i18n.language, {
    month: "long",
    year: "numeric",
  })

  const dayHeaders = Array.from({ length: 7 }, (_, i) =>
    new Date(2023, 0, i + 1).toLocaleDateString(i18n.language, { weekday: "short" })
  )

  async function handleCreateToken() {
    if (!newTokenName.trim()) return
    setCreatingToken(true)
    try {
      const token = await api.post<CalendarToken>("/calendar/tokens", { name: newTokenName.trim() })
      setTokens(prev => [...prev, token])
      setNewlyCreatedUrl(getICalUrl(token.token))
      setNewTokenName("")
      toast.success(t("calendar.token.createSuccess"))
    } catch {
      void 0
    } finally {
      setCreatingToken(false)
    }
  }

  function handleDialogClose(open: boolean) {
    setCreateOpen(open)
    if (!open) {
      setNewlyCreatedUrl(null)
      setNewTokenName("")
    }
  }

  async function handleDeleteToken(id: number) {
    if (!confirm(t("calendar.token.deleteConfirm"))) return
    try {
      await api.delete(`/calendar/tokens/${id}`)
      setTokens(prev => prev.filter(t => t.id !== id))
      toast.success(t("calendar.token.deleteSuccess"))
    } catch {
      void 0
    }
  }

  function getICalUrl(token: string) {
    return `${window.location.origin}/api/calendar/feed?token=${token}`
  }

  function copyToClipboard(text: string) {
    navigator.clipboard.writeText(text).then(() => {
      toast.success(t("calendar.token.copied"))
    }).catch(() => void 0)
  }

  async function handleFormSubmit(data: CreateSubscriptionInput) {
    if (editingSub) {
      const updated = await api.put<Subscription>(`/subscriptions/${editingSub.id}`, {
        ...data,
        payment_method_id: data.payment_method_id ?? 0,
      })
      setSubscriptions(prev => prev.map(s => s.id === editingSub.id ? updated : s))
      toast.success(t("calendar.editSuccess"))
      setEditingSub(null)
      setFormOpen(false)
      return updated
    }
    const created = await api.post<Subscription>("/subscriptions", data)
    setSubscriptions(prev => [...prev, created])
    setEditingSub(null)
    setFormOpen(false)
    return created
  }

  const selectedSubs = selectedDay !== null ? (billingMap.get(selectedDay) ?? []) : []

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-4xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/">
                <ArrowLeft className="size-4" />
              </Link>
            </Button>
            <h1 className="text-lg font-bold tracking-tight">{t("calendar.title")}</h1>
          </div>
          <Button variant="ghost" size="icon-sm" asChild>
            <Link to="/settings">
              <Settings className="size-4" />
            </Link>
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-6 space-y-6">
        {/* Calendar */}
        <Card>
          <CardHeader className="pb-3">
            <div className="flex items-center justify-between">
              <Button variant="ghost" size="icon-sm" onClick={prevMonth}>
                <ChevronLeft className="size-4" />
              </Button>
              <span className="font-semibold capitalize">{monthLabel}</span>
              <Button variant="ghost" size="icon-sm" onClick={nextMonth}>
                <ChevronRight className="size-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent className="p-0 pb-4">
            {/* Day headers */}
            <div className="grid grid-cols-7 border-b">
              {dayHeaders.map((d) => (
                <div key={d} className="py-2 text-center text-xs font-medium text-muted-foreground">
                  {d}
                </div>
              ))}
            </div>
            {/* Day cells */}
            {loadingSubs ? (
              <div className="grid grid-cols-7">
                {Array.from({ length: 42 }).map((_, i) => (
                  <div key={i} className="min-h-[60px] sm:min-h-[80px] border-b border-r last:border-r-0 p-1" />
                ))}
              </div>
            ) : (
              <div className="grid grid-cols-7">
                {cells.map((day, idx) => {
                  const isToday =
                    day !== null &&
                    day === today.getDate() &&
                    viewMonth === today.getMonth() &&
                    viewYear === today.getFullYear()
                  const hasBilling = day !== null && billingMap.has(day)
                  const isSelected = day !== null && day === selectedDay
                  const subsForDay = day !== null ? (billingMap.get(day) ?? []) : []

                  return (
                    <div
                      key={idx}
                      onClick={() => {
                        if (day === null) return
                        setSelectedDay(isSelected ? null : day)
                      }}
                      className={cn(
                        "min-h-[60px] sm:min-h-[80px] border-b border-r p-1 flex flex-col",
                        (idx + 1) % 7 === 0 && "border-r-0",
                        day === null && "bg-muted/30",
                        day !== null && "cursor-pointer hover:bg-muted/50 transition-colors",
                        isSelected && "bg-muted",
                      )}
                    >
                      {day !== null && (
                        <>
                          <span
                            className={cn(
                              "text-sm w-7 h-7 flex items-center justify-center rounded-full self-end",
                              isToday && "ring-2 ring-primary font-semibold text-primary",
                              !isToday && "text-foreground",
                            )}
                          >
                            {day}
                          </span>
                          {hasBilling && (
                            <div className="mt-auto flex flex-col gap-0.5 overflow-hidden px-0.5 pb-0.5">
                              {subsForDay.slice(0, 2).map((sub) => (
                                <span
                                  key={sub.id}
                                  className="truncate rounded bg-primary/10 px-1 text-[10px] leading-4 text-primary"
                                >
                                  {sub.name}
                                </span>
                              ))}
                              {subsForDay.length > 2 && (
                                <span className="px-1 text-[10px] leading-4 text-muted-foreground">
                                  +{subsForDay.length - 2}
                                </span>
                              )}
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  )
                })}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Selected day subscriptions */}
        {selectedDay !== null && (
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">
                {new Date(viewYear, viewMonth, selectedDay).toLocaleDateString(i18n.language, {
                  weekday: "long",
                  month: "long",
                  day: "numeric",
                })}
              </CardTitle>
            </CardHeader>
            <CardContent>
              {selectedSubs.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("calendar.noSubscriptions")}</p>
              ) : (
                <div className="space-y-2">
                  {selectedSubs.map((sub) => (
                    <div
                      key={sub.id}
                      className="flex items-center gap-3 rounded-md border p-2 cursor-pointer hover:bg-muted/50 transition-colors"
                      onClick={() => { setEditingSub(sub); setFormOpen(true) }}
                    >
                      {sub.icon ? (
                        <img src={sub.icon} alt="" className="size-8 rounded object-contain shrink-0" />
                      ) : (
                        <div className="size-8 rounded bg-muted flex items-center justify-center shrink-0">
                          <span className="text-xs font-medium text-muted-foreground">
                            {sub.name.charAt(0).toUpperCase()}
                          </span>
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium truncate">{sub.name}</p>
                      </div>
                      <span className="text-sm font-semibold shrink-0">
                        {sub.amount.toLocaleString(i18n.language, {
                          style: "currency",
                          currency: sub.currency,
                          minimumFractionDigits: 2,
                        })}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        )}

        <Separator />

        {/* Calendar Subscription section */}
        <div className="space-y-4">
          <div className="flex items-start justify-between">
            <div>
              <h2 className="text-base font-semibold tracking-tight">
                {t("calendar.token.title")}
              </h2>
              <p className="mt-0.5 text-sm text-muted-foreground">
                {t("calendar.token.description")}
              </p>
            </div>
            <Dialog open={createOpen} onOpenChange={handleDialogClose}>
              <DialogTrigger asChild>
                <Button size="sm" className="gap-1.5" disabled={tokens.length >= 5}>
                  <Plus className="size-4" />
                  {t("calendar.token.createLink")}
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-md">
                <DialogHeader>
                  <DialogTitle>{t("calendar.token.createLink")}</DialogTitle>
                </DialogHeader>
                {newlyCreatedUrl ? (
                  <div className="space-y-3">
                    <div className="rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400">
                      {t("calendar.token.copyWarning")}
                    </div>
                    <div className="space-y-2">
                      <Label>{t("calendar.token.urlLabel")}</Label>
                      <div className="flex gap-2">
                        <code className="flex-1 rounded-md border bg-muted px-3 py-2 text-xs break-all">
                          {newlyCreatedUrl}
                        </code>
                        <Button
                          size="icon-sm"
                          variant="outline"
                          onClick={() => copyToClipboard(newlyCreatedUrl)}
                        >
                          <Copy className="size-4" />
                        </Button>
                      </div>
                    </div>
                    <div className="space-y-2">
                      <Label>{t("calendar.token.usageTitle")}</Label>
                      <p className="text-sm text-muted-foreground">
                        {t("calendar.token.usageDescription")}
                      </p>
                    </div>
                  </div>
                ) : (
                  <form
                    onSubmit={(e) => {
                      e.preventDefault()
                      void handleCreateToken()
                    }}
                    className="space-y-4"
                  >
                    <div className="space-y-2">
                      <Label htmlFor="token-name">{t("calendar.token.name")}</Label>
                      <Input
                        id="token-name"
                        placeholder={t("calendar.token.namePlaceholder")}
                        value={newTokenName}
                        onChange={(e) => setNewTokenName(e.target.value)}
                        maxLength={100}
                        required
                      />
                    </div>
                    <Button size="sm" type="submit" disabled={creatingToken || !newTokenName.trim()}>
                      {creatingToken ? t("calendar.token.creating") : t("calendar.token.create")}
                    </Button>
                  </form>
                )}
              </DialogContent>
            </Dialog>
          </div>

          <Separator />

          {!loadingTokens && tokens.length === 0 && (
            <div className="py-8 text-center">
              <Link2 className="mx-auto size-8 text-muted-foreground/50" />
              <p className="mt-2 text-sm font-medium text-muted-foreground">
                {t("calendar.token.empty.title")}
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                {t("calendar.token.empty.description")}
              </p>
            </div>
          )}

          {tokens.length > 0 && (
            <div className="space-y-2">
              {tokens.map((token) => (
                <div
                  key={token.id}
                  className="flex items-center justify-between rounded-md border px-3 py-2.5"
                >
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-medium">{token.name}</p>
                    <p className="mt-0.5 text-xs text-muted-foreground">
                      {new Date(token.created_at).toLocaleDateString(i18n.language)}
                    </p>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleDeleteToken(token.id)}
                      title={t("calendar.token.delete")}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
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
