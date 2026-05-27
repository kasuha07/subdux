import { useMemo } from "react"
import { useTranslation } from "react-i18next"
import { X } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  useSubscriptionFormState,
} from "@/features/subscriptions/hooks/use-subscription-form-state"
import type {
  Category,
  CreateSubscriptionInput,
  PaymentMethod,
  Subscription,
  UserCurrency,
} from "@/types"

import IconPicker from "./icon-picker"
import SubscriptionMetadataFields from "./subscription-metadata-fields"
import SubscriptionNotificationFields from "./subscription-notification-fields"
import SubscriptionRecurrenceFields from "./subscription-recurrence-fields"

interface SubscriptionFormProps {
  categories: Category[]
  onMarkRenewed?: (subscription: Subscription) => Promise<Subscription>
  onOpenChange: (open: boolean) => void
  onSubmit: (data: CreateSubscriptionInput) => Promise<Subscription>
  open: boolean
  paymentMethods: PaymentMethod[]
  subscription?: Subscription | null
  userCurrencies: UserCurrency[]
}

export default function SubscriptionForm({
  categories,
  onMarkRenewed,
  onOpenChange,
  onSubmit,
  open,
  paymentMethods,
  subscription,
  userCurrencies,
}: SubscriptionFormProps) {
  const { t, i18n } = useTranslation()

  const {
    currencyOptions,
    error,
    handleIconChange,
    handleIconFileSelected,
    handleMarkRenewed,
    handleSubmit,
    isEditing,
    loading,
    setField,
    values,
  } = useSubscriptionFormState({
    language: i18n.language,
    onMarkRenewed,
    onOpenChange,
    onSubmit,
    open,
    paymentMethods,
    subscription,
    t,
    userCurrencies,
  })

  const iconPickerNode = useMemo(
    () => (
      <IconPicker
        value={values.icon}
        onChange={handleIconChange}
        onFileSelected={handleIconFileSelected}
        maxFileSizeKB={64}
        triggerSize="sm"
        allowImageUrl
      />
    ),
    [handleIconChange, handleIconFileSelected, values.icon]
  )
  const editDialogClass = isEditing ? "subscription-edit-modal" : ""

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className={`subscription-form-dialog ${editDialogClass} flex max-h-[calc(100vh-1.5rem)] max-w-2xl flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]`}
        showCloseButton={false}
        onInteractOutside={(event) => event.preventDefault()}
        onPointerDownOutside={(event) => event.preventDefault()}
      >
        <DialogHeader className="subscription-form-dialog-header flex-row items-start justify-between gap-3 border-b px-5 pt-5 pb-4 text-left sm:px-6">
          <div className="min-w-0 space-y-2">
            <DialogTitle>
              {isEditing ? t("subscription.form.editTitle") : t("subscription.form.addTitle")}
            </DialogTitle>
          </div>
          <DialogDescription className="sr-only">
            {isEditing ? t("subscription.form.editDescription") : t("subscription.form.addDescription")}
          </DialogDescription>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="-mt-1 -mr-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
            asChild
          >
            <DialogClose aria-label={t("common.close")}>
              <X />
            </DialogClose>
          </Button>
        </DialogHeader>
        <form onSubmit={(event) => void handleSubmit(event)} className="flex min-h-0 flex-1 flex-col">
          <div className="subscription-form-dialog-body min-h-0 flex-1 space-y-5 overflow-y-auto px-5 py-4 sm:px-6">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="space-y-4">
              <div className="grid grid-cols-[auto_minmax(0,1fr)] items-end gap-3">
                <div className="space-y-2">
                  <Label>{t("subscription.form.iconPicker.label")}</Label>
                  {iconPickerNode}
                </div>
                <div className="min-w-0 space-y-2">
                  <Label htmlFor="name">{t("subscription.form.nameLabel")}</Label>
                  <Input
                    id="name"
                    placeholder={t("subscription.form.namePlaceholder")}
                    value={values.name}
                    onChange={(event) => setField("name", event.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="grid grid-cols-[minmax(0,1fr)_minmax(7rem,0.9fr)] gap-3 sm:grid-cols-2">
                <div className="min-w-0 space-y-2">
                  <Label htmlFor="amount">{t("subscription.form.amountLabel")}</Label>
                  <Input
                    id="amount"
                    type="number"
                    step="0.01"
                    min="0"
                    placeholder={t("subscription.form.amountPlaceholder")}
                    value={values.amount}
                    onChange={(event) => setField("amount", event.target.value)}
                    required
                  />
                </div>
                <div className="min-w-0 space-y-2">
                  <Label htmlFor="currency">{t("subscription.form.currencyLabel")}</Label>
                  <Select value={values.currency} onValueChange={(value) => setField("currency", value)}>
                    <SelectTrigger id="currency" className="w-full min-w-0">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {currencyOptions.map((option) => (
                        <SelectItem key={option.code} value={option.code}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            <div className="grid grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] gap-3 sm:grid-cols-2">
              <div className="min-w-0 space-y-2">
                <Label htmlFor="status">{t("subscription.form.statusLabel")}</Label>
                <Select value={values.status} onValueChange={(value) => setField("status", value as typeof values.status)}>
                  <SelectTrigger id="status" className="w-full min-w-0">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="active">{t("subscription.form.status.active")}</SelectItem>
                    <SelectItem value="ended">{t("subscription.form.status.ended")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="min-w-0 space-y-2">
                <Label htmlFor="renewal-mode">{t("subscription.form.renewalModeLabel")}</Label>
                <Select
                  value={values.renewalMode}
                  onValueChange={(value) => setField("renewalMode", value as typeof values.renewalMode)}
                  disabled={values.status === "ended"}
                >
                  <SelectTrigger id="renewal-mode" className="w-full min-w-0">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="auto_renew">{t("subscription.form.renewalMode.auto_renew")}</SelectItem>
                    <SelectItem value="manual_renew">{t("subscription.form.renewalMode.manual_renew")}</SelectItem>
                    <SelectItem value="cancel_at_period_end">
                      {t("subscription.form.renewalMode.cancel_at_period_end")}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="next-billing-date">{t("subscription.form.nextBillingDateLabel")}</Label>
              <Input
                id="next-billing-date"
                type="date"
                value={values.nextBillingDate}
                onChange={(event) => setField("nextBillingDate", event.target.value)}
                required
              />
            </div>

            {values.status === "ended" ? (
              <div className="space-y-2">
                <Label htmlFor="ends-at">{t("subscription.form.endsAtLabel")}</Label>
                <Input
                  id="ends-at"
                  type="date"
                  value={values.endsAt}
                  onChange={(event) => setField("endsAt", event.target.value)}
                  required
                />
              </div>
            ) : null}

            <SubscriptionRecurrenceFields
              billingType={values.billingType}
              recurrenceType={values.recurrenceType}
              onRecurrenceTypeChange={(value) => setField("recurrenceType", value)}
              intervalCount={values.intervalCount}
              onIntervalCountChange={(value) => setField("intervalCount", value)}
              intervalUnit={values.intervalUnit}
              onIntervalUnitChange={(value) => setField("intervalUnit", value)}
              monthlyDay={values.monthlyDay}
              onMonthlyDayChange={(value) => setField("monthlyDay", value)}
              yearlyMonth={values.yearlyMonth}
              onYearlyMonthChange={(value) => setField("yearlyMonth", value)}
              yearlyDay={values.yearlyDay}
              onYearlyDayChange={(value) => setField("yearlyDay", value)}
            />

            <SubscriptionMetadataFields
              categories={categories}
              categoryId={values.categoryId}
              onCategoryIdChange={(value) => setField("categoryId", value)}
              paymentMethods={paymentMethods}
              paymentMethodId={values.paymentMethodId}
              onPaymentMethodIdChange={(value) => setField("paymentMethodId", value)}
              url={values.url}
              onURLChange={(value) => setField("url", value)}
              notes={values.notes}
              onNotesChange={(value) => setField("notes", value)}
            />

            <SubscriptionNotificationFields
              notifyEnabled={values.notifyEnabled}
              notifyDaysBefore={values.notifyDaysBefore}
              onNotifyEnabledChange={(value) => setField("notifyEnabled", value)}
              onNotifyDaysBeforeChange={(value) => setField("notifyDaysBefore", value)}
            />
          </div>

          <div className="subscription-form-dialog-footer sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
            <div className="flex flex-col-reverse gap-2 sm:flex-row">
              <Button
                type="button"
                variant="outline"
                className="w-full sm:flex-1"
                onClick={() => onOpenChange(false)}
              >
                {t("subscription.form.cancel")}
              </Button>
              {isEditing &&
              subscription &&
              values.status === "active" &&
              values.renewalMode === "manual_renew" ? (
                <Button
                  type="button"
                  variant="secondary"
                  className="w-full sm:flex-1"
                  onClick={() => void handleMarkRenewed()}
                  disabled={loading}
                >
                  {t("subscription.form.renewalMode.manual_renew")}
                </Button>
              ) : null}
              <Button type="submit" className="w-full sm:flex-1" disabled={loading}>
                {loading
                  ? t("subscription.form.saving")
                  : isEditing
                    ? t("subscription.form.update")
                    : t("subscription.form.addButton")}
              </Button>
            </div>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
