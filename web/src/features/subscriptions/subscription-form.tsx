import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
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
  onOpenChange: (open: boolean) => void
  onSubmit: (data: CreateSubscriptionInput) => Promise<Subscription>
  open: boolean
  paymentMethods: PaymentMethod[]
  subscription?: Subscription | null
  userCurrencies: UserCurrency[]
}

export default function SubscriptionForm({
  categories,
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
    handleSubmit,
    isEditing,
    loading,
    needsAnchorDate,
    setField,
    values,
  } = useSubscriptionFormState({
    language: i18n.language,
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
      />
    ),
    [handleIconChange, handleIconFileSelected, values.icon]
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-2xl flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>
            {isEditing ? t("subscription.form.editTitle") : t("subscription.form.addTitle")}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={(event) => void handleSubmit(event)} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 space-y-5 overflow-y-auto px-5 py-4 sm:px-6">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="space-y-4">
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-[auto_1fr]">
                <div className="space-y-2">
                  <Label>{t("subscription.form.iconPicker.label")}</Label>
                  {iconPickerNode}
                </div>
                <div className="space-y-2">
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

              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="amount">{t("subscription.form.amountLabel")}</Label>
                  <Input
                    id="amount"
                    type="number"
                    step="0.01"
                    min="0.01"
                    placeholder={t("subscription.form.amountPlaceholder")}
                    value={values.amount}
                    onChange={(event) => setField("amount", event.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="currency">{t("subscription.form.currencyLabel")}</Label>
                  <Select value={values.currency} onValueChange={(value) => setField("currency", value)}>
                    <SelectTrigger id="currency">
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

            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="billing-type">{t("subscription.form.billingTypeLabel")}</Label>
                <Select
                  value={values.billingType}
                  onValueChange={(value) => setField("billingType", value)}
                >
                  <SelectTrigger id="billing-type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="recurring">
                      {t("subscription.form.billingType.recurring")}
                    </SelectItem>
                    <SelectItem value="one_time">{t("subscription.form.billingType.one_time")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="enabled">{t("subscription.form.enabledLabel")}</Label>
                <div className="inline-flex h-9 w-full items-center rounded-md border px-3 sm:w-fit">
                  <Switch
                    id="enabled"
                    checked={values.enabled}
                    onCheckedChange={(checked) => setField("enabled", checked)}
                  />
                  <span className="ml-2 text-sm text-muted-foreground">
                    {values.enabled ? t("subscription.form.enabled") : t("subscription.form.disabled")}
                  </span>
                </div>
              </div>
            </div>

            {needsAnchorDate && (
              <div className="space-y-2">
                <Label htmlFor="anchor-date">
                  {values.billingType === "one_time"
                    ? t("subscription.form.purchaseDateLabel")
                    : t("subscription.form.anchorDateLabel")}
                </Label>
                <Input
                  id="anchor-date"
                  type="date"
                  value={values.billingAnchorDate}
                  onChange={(event) => setField("billingAnchorDate", event.target.value)}
                  required={needsAnchorDate}
                />
              </div>
            )}

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
              trialEnabled={values.trialEnabled}
              onTrialEnabledChange={(enabled) => setField("trialEnabled", enabled)}
              trialStartDate={values.trialStartDate}
              onTrialStartDateChange={(value) => setField("trialStartDate", value)}
              trialEndDate={values.trialEndDate}
              onTrialEndDateChange={(value) => setField("trialEndDate", value)}
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

          <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
            <div className="flex flex-col-reverse gap-2 sm:flex-row">
              <Button
                type="button"
                variant="outline"
                className="w-full sm:flex-1"
                onClick={() => onOpenChange(false)}
              >
                {t("subscription.form.cancel")}
              </Button>
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
