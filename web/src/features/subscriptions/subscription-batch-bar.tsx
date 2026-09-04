import { useState } from "react"
import { useTranslation } from "react-i18next"
import { CheckSquare, CreditCard, Play, RotateCcw, Square, Tags, Trash2, X } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api, localizeBackendError } from "@/lib/api"
import { toast } from "@/lib/toast"
import type {
  Category,
  PaymentMethod,
  SubscriptionBatchFailure,
  SubscriptionBatchInput,
  SubscriptionBatchResult,
} from "@/types"

import { createActivateBatchInput } from "./subscription-batch-inputs"

const CLEAR_VALUE = "__none__"

interface SubscriptionBatchBarProps {
  categories: Category[]
  getSubscriptionName: (id: number) => string
  onBatchApplied: () => void
  onClearSelection: () => void
  paymentMethodLabelMap: Map<number, string>
  paymentMethods: PaymentMethod[]
  selectedCount: number
  selectedIDs: number[]
}

function failureSummary(
  getSubscriptionName: (id: number) => string,
  failures: SubscriptionBatchFailure[]
): string {
  return failures
    .slice(0, 3)
    .map((failure) => {
      const name = getSubscriptionName(failure.id)
      const label = name ? name : `#${failure.id}`
      return `${label}: ${localizeBackendError(failure.code)}`
    })
    .join("\n")
}

export default function SubscriptionBatchBar({
  categories,
  getSubscriptionName,
  onBatchApplied,
  onClearSelection,
  paymentMethodLabelMap,
  paymentMethods,
  selectedCount,
  selectedIDs,
}: SubscriptionBatchBarProps) {
  const { t } = useTranslation()
  const [dialog, setDialog] = useState<"delete" | "category" | "payment_method" | null>(null)
  const [categoryValue, setCategoryValue] = useState<string>("")
  const [paymentMethodValue, setPaymentMethodValue] = useState<string>("")
  const [submitting, setSubmitting] = useState(false)

  async function runBatch(input: SubscriptionBatchInput) {
    setSubmitting(true)
    try {
      const result = await api.post<SubscriptionBatchResult>("/subscriptions/batch", input, {
        errorHandling: "toast",
      })
      reportResult(input, result)
      onBatchApplied()
    } catch {
      void 0
    } finally {
      setSubmitting(false)
    }
  }

  function reportResult(input: SubscriptionBatchInput, result: SubscriptionBatchResult) {
    const description = failureSummary(getSubscriptionName, result.failures)
    if (result.failed === 0) {
      toast.success(t(`subscription.batch.success.${input.action}`, { count: result.succeeded }))
      return
    }
    if (result.succeeded === 0) {
      toast.error(t("subscription.batch.failed", { count: result.failed }), { description })
      return
    }
    toast.error(
      t("subscription.batch.partial", { succeeded: result.succeeded, failed: result.failed }),
      { description }
    )
  }

  function handleDeleteConfirm() {
    void runBatch({ action: "delete", ids: selectedIDs })
    setDialog(null)
  }

  function handleCategoryConfirm() {
    const value = categoryValue === CLEAR_VALUE ? null : Number(categoryValue)
    void runBatch({
      action: "update",
      ids: selectedIDs,
      category_id: value,
    })
    setCategoryValue("")
    setDialog(null)
  }

  function handlePaymentMethodConfirm() {
    const value = paymentMethodValue === CLEAR_VALUE ? null : Number(paymentMethodValue)
    void runBatch({
      action: "update",
      ids: selectedIDs,
      payment_method_id: value,
    })
    setPaymentMethodValue("")
    setDialog(null)
  }

  return (
    <>
      <div className="mb-4 flex flex-wrap items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2">
        <CheckSquare className="size-4 text-muted-foreground" />
        <span className="text-sm font-medium">
          {t("subscription.batch.selected", { count: selectedCount })}
        </span>

        <DropdownMenu modal={false}>
          <DropdownMenuTrigger asChild>
            <Button variant="secondary" size="sm" className="shrink-0">
              {t("subscription.batch.actions")}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuLabel>{t("subscription.batch.actionsLabel")}</DropdownMenuLabel>
            <DropdownMenuItem
              disabled={submitting}
              onSelect={(event) => {
                event.preventDefault()
                void runBatch({ action: "mark_renewed", ids: selectedIDs })
              }}
            >
              <RotateCcw className="size-4" />
              {t("subscription.batch.markRenewed")}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={submitting}
              onSelect={(event) => {
                event.preventDefault()
                void runBatch(createActivateBatchInput(selectedIDs))
              }}
            >
              <Play className="size-4" />
              {t("subscription.batch.activate")}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={submitting}
              onSelect={(event) => {
                event.preventDefault()
                void runBatch({ action: "update", ids: selectedIDs, status: "ended" })
              }}
            >
              <Square className="size-4" />
              {t("subscription.batch.end")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={submitting || categories.length === 0}
              onSelect={() => {
                setCategoryValue("")
                setDialog("category")
              }}
            >
              <Tags className="size-4" />
              {t("subscription.batch.setCategory")}
            </DropdownMenuItem>
            <DropdownMenuItem
              disabled={submitting || paymentMethods.length === 0}
              onSelect={() => {
                setPaymentMethodValue("")
                setDialog("payment_method")
              }}
            >
              <CreditCard className="size-4" />
              {t("subscription.batch.setPaymentMethod")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              disabled={submitting}
              variant="destructive"
              onSelect={() => {
                setDialog("delete")
              }}
            >
              <Trash2 className="size-4" />
              {t("subscription.batch.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          variant="ghost"
          size="sm"
          className="ml-auto shrink-0"
          onClick={onClearSelection}
        >
          <X className="size-4" />
          {t("subscription.batch.clear")}
        </Button>
      </div>

      <Dialog open={dialog === "delete"} onOpenChange={(open) => !open && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("subscription.batch.deleteTitle")}</DialogTitle>
            <DialogDescription>
              {t("subscription.batch.deleteDescription", { count: selectedCount })}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {t("common.cancel")}
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirm} disabled={submitting}>
              <Trash2 className="size-4" />
              {t("subscription.batch.deleteConfirm", { count: selectedCount })}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={dialog === "category"}
        onOpenChange={(open) => !open && setDialog(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("subscription.batch.categoryTitle")}</DialogTitle>
            <DialogDescription>
              {t("subscription.batch.categoryDescription", { count: selectedCount })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="batch-category">{t("subscription.batch.categoryLabel")}</Label>
            <Select value={categoryValue} onValueChange={setCategoryValue}>
              <SelectTrigger id="batch-category" className="w-full">
                <SelectValue placeholder={t("subscription.batch.categoryPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={CLEAR_VALUE}>{t("subscription.batch.noCategory")}</SelectItem>
                {categories.map((category) => (
                  <SelectItem key={category.id} value={String(category.id)}>
                    {category.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {t("common.cancel")}
            </Button>
            <Button onClick={handleCategoryConfirm} disabled={submitting || !categoryValue}>
              {t("subscription.batch.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={dialog === "payment_method"}
        onOpenChange={(open) => !open && setDialog(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("subscription.batch.paymentMethodTitle")}</DialogTitle>
            <DialogDescription>
              {t("subscription.batch.paymentMethodDescription", { count: selectedCount })}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="batch-payment-method">
              {t("subscription.batch.paymentMethodLabel")}
            </Label>
            <Select value={paymentMethodValue} onValueChange={setPaymentMethodValue}>
              <SelectTrigger id="batch-payment-method" className="w-full">
                <SelectValue placeholder={t("subscription.batch.paymentMethodPlaceholder")} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={CLEAR_VALUE}>
                  {t("subscription.batch.noPaymentMethod")}
                </SelectItem>
                {paymentMethods.map((method) => (
                  <SelectItem key={method.id} value={String(method.id)}>
                    {paymentMethodLabelMap.get(method.id) ?? method.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialog(null)}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handlePaymentMethodConfirm}
              disabled={submitting || !paymentMethodValue}
            >
              {t("subscription.batch.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
