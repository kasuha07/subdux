import { lazy, Suspense, useEffect, useState, type CSSProperties } from "react"
import { useTranslation } from "react-i18next"
import { ExternalLink, Pencil, Trash2, X } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Tooltip } from "@/components/ui/tooltip"
import { safeHref } from "@/lib/safe-href"
import { formatCurrencyWithSymbol } from "@/lib/utils"
import type { Subscription, SubscriptionDetail } from "@/types"
import {
  getCachedSubscriptionDetail,
  loadSubscriptionDetail,
} from "./subscription-detail-cache"
import { SubscriptionIcon } from "./subscription-icon"

const loadSubscriptionDetailContent = () => import("./subscription-detail-content")
const SubscriptionDetailContent = lazy(loadSubscriptionDetailContent)

export interface SubscriptionDetailDrawerProps {
  open: boolean
  subscription: Subscription | null
  categoryName?: string
  currencySymbol?: string
  paymentMethodName?: string
  onOpenChange: (open: boolean) => void
  onEdit: (subscription: Subscription) => void
  onDelete?: (id: number) => void
}

function animationDelay(delay: number): CSSProperties & { "--detail-delay"?: string } {
  return { "--detail-delay": `${delay}ms` }
}

function DetailSkeleton() {
  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, index) => (
          <Skeleton
            key={index}
            className="detail-drawer-stage h-20 rounded-lg"
            style={animationDelay(index * 40)}
          />
        ))}
      </div>
      <Skeleton className="detail-drawer-stage h-10 rounded-lg" style={animationDelay(120)} />
      <Skeleton className="detail-drawer-stage h-64 rounded-lg" style={animationDelay(160)} />
    </div>
  )
}

export default function SubscriptionDetailDrawer({
  open,
  subscription,
  categoryName,
  currencySymbol,
  paymentMethodName,
  onOpenChange,
  onEdit,
  onDelete,
}: SubscriptionDetailDrawerProps) {
  const { t, i18n } = useTranslation()
  const [detail, setDetail] = useState<SubscriptionDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  useEffect(() => {
    // Preserve the visible detail while the drawer plays its exit animation.
    if (!open || !subscription) return

    let active = true
    const cached = getCachedSubscriptionDetail(subscription.id)
    setDetail(cached)
    setLoading(!cached)
    setError("")
    loadSubscriptionDetail(subscription.id)
      .then((data) => {
        if (active) {
          setDetail(data)
        }
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : t("subscription.detail.error"))
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false)
        }
      })

    return () => {
      active = false
    }
  }, [open, subscription, t])

  const activeSubscription = detail?.subscription ?? subscription

  function formatAmount(amount: number, currency = activeSubscription?.currency ?? "USD") {
    const symbol = currency.toUpperCase() === activeSubscription?.currency.toUpperCase()
      ? currencySymbol
      : undefined
    return formatCurrencyWithSymbol(amount, currency, symbol, i18n.language)
  }

  function handleEdit() {
    if (!activeSubscription) {
      return
    }
    onEdit(activeSubscription)
    onOpenChange(false)
  }

  function handleDelete() {
    if (!activeSubscription || !onDelete) {
      return
    }
    onDelete(activeSubscription.id)
  }

  const subscriptionUrlHref = activeSubscription?.url ? safeHref(activeSubscription.url) : null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-motion="drawer"
        className="fixed top-0 right-0 left-auto flex h-dvh max-h-dvh w-full max-w-full translate-x-0 translate-y-0 flex-col gap-0 rounded-none border-y-0 border-r-0 p-0 duration-300 sm:max-w-full md:max-w-xl"
        showCloseButton={false}
        onOpenAutoFocus={(event) => event.preventDefault()}
      >
        <DialogHeader className="detail-drawer-stage flex-row items-start justify-between gap-3 border-b px-5 pt-5 pb-4 text-left sm:px-6">
          <div className="flex min-w-0 items-center gap-3">
            {activeSubscription ? (
              <div className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-xl bg-muted">
                <SubscriptionIcon
                  icon={activeSubscription.icon}
                  name={activeSubscription.name}
                  size={26}
                  className="size-7 object-contain"
                />
              </div>
            ) : null}
            <div className="min-w-0">
              <DialogTitle className="truncate">
                {activeSubscription?.name ?? t("subscription.detail.titleFallback")}
              </DialogTitle>
              <DialogDescription className="sr-only">
                {t("subscription.detail.description")}
              </DialogDescription>
              {activeSubscription ? (
                <p className="mt-0.5 text-sm font-medium text-muted-foreground">
                  {formatAmount(activeSubscription.amount, activeSubscription.currency)}
                </p>
              ) : null}
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {subscriptionUrlHref ? (
              <Tooltip content={t("subscription.form.urlLabel")}>
                <Button variant="outline" size="icon-sm" asChild>
                  <a
                    href={subscriptionUrlHref}
                    target="_blank"
                    rel="noopener noreferrer"
                    aria-label={t("subscription.form.urlLabel")}
                  >
                    <ExternalLink className="size-4" />
                  </a>
                </Button>
              </Tooltip>
            ) : null}
            <Tooltip content={t("subscription.detail.edit")}>
              <Button
                variant="outline"
                size="icon-sm"
                onClick={handleEdit}
                disabled={!activeSubscription}
                aria-label={t("subscription.detail.edit")}
              >
                <Pencil className="size-4" />
              </Button>
            </Tooltip>
            {onDelete ? (
              <Tooltip content={t("common.delete")}>
                <Button
                  variant="outline"
                  size="icon-sm"
                  className="text-destructive hover:bg-destructive/10 hover:text-destructive"
                  onClick={handleDelete}
                  disabled={!activeSubscription}
                  aria-label={t("common.delete")}
                >
                  <Trash2 className="size-4" />
                </Button>
              </Tooltip>
            ) : null}
            <Tooltip content={t("common.close")}>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                className="-mr-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
                asChild
              >
                <DialogClose aria-label={t("common.close")}>
                  <X />
                </DialogClose>
              </Button>
            </Tooltip>
          </div>
        </DialogHeader>

        <ScrollArea className="min-h-0 flex-1">
          <div className="space-y-5 px-5 py-5 sm:px-6">
            <Suspense fallback={<DetailSkeleton />}>
              {activeSubscription && (
                <SubscriptionDetailContent
                  subscription={activeSubscription}
                  detail={detail}
                  loading={loading}
                  error={error}
                  categoryName={categoryName}
                  currencySymbol={currencySymbol}
                  paymentMethodName={paymentMethodName}
                />
              )}
            </Suspense>
          </div>
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}
