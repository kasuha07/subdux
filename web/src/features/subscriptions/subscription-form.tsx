import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
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
import { Switch } from "@/components/ui/switch"
import { api } from "@/lib/api"
import { DEFAULT_CURRENCY_FALLBACK, getPresetCurrencyMeta } from "@/lib/currencies"
import type {
  Category,
  CreateSubscriptionInput,
  PaymentMethod,
  Subscription,
  UploadIconResponse,
  UserCurrency,
} from "@/types"

import IconPicker from "./icon-picker"
import SubscriptionMetadataFields from "./subscription-metadata-fields"
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

function formatDateInput(value: string | null | undefined): string {
  if (!value) {
    return new Date().toISOString().split("T")[0]
  }
  return new Date(value).toISOString().split("T")[0]
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
  const isEditing = !!subscription

  const [name, setName] = useState(subscription?.name || "")
  const [amount, setAmount] = useState(subscription?.amount?.toString() || "")
  const [currency, setCurrency] = useState(subscription?.currency || DEFAULT_CURRENCY_FALLBACK[0] || "")
  const [enabled, setEnabled] = useState(subscription?.enabled ?? true)
  const [billingType, setBillingType] = useState<string>(subscription?.billing_type || "recurring")
  const [recurrenceType, setRecurrenceType] = useState<string>(subscription?.recurrence_type || "interval")
  const [intervalCount, setIntervalCount] = useState((subscription?.interval_count ?? 1).toString())
  const [intervalUnit, setIntervalUnit] = useState<string>(subscription?.interval_unit || "month")
  const [billingAnchorDate, setBillingAnchorDate] = useState(formatDateInput(subscription?.billing_anchor_date))
  const [monthlyDay, setMonthlyDay] = useState((subscription?.monthly_day ?? new Date().getDate()).toString())
  const [yearlyMonth, setYearlyMonth] = useState(
    (subscription?.yearly_month ?? new Date().getMonth() + 1).toString()
  )
  const [yearlyDay, setYearlyDay] = useState((subscription?.yearly_day ?? new Date().getDate()).toString())
  const [trialEnabled, setTrialEnabled] = useState(subscription?.trial_enabled ?? false)
  const [trialStartDate, setTrialStartDate] = useState(formatDateInput(subscription?.trial_start_date))
  const [trialEndDate, setTrialEndDate] = useState(formatDateInput(subscription?.trial_end_date))
  const [categoryId, setCategoryId] = useState<string>(subscription?.category_id?.toString() || "")
  const [paymentMethodId, setPaymentMethodId] = useState<string>(
    subscription?.payment_method_id?.toString() || ""
  )
  const [icon, setIcon] = useState(subscription?.icon || "")
  const [url, setUrl] = useState(subscription?.url || "")
  const [notes, setNotes] = useState(subscription?.notes || "")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [iconFile, setIconFile] = useState<File | null>(null)
  const wasOpenRef = useRef(false)

  const handleIconChange = useCallback((value: string) => {
    setIcon(value)
    setIconFile(null)
  }, [])

  const handleIconFileSelected = useCallback((file: File) => {
    setIconFile(file)
    setIcon("")
  }, [])

  const iconPickerNode = useMemo(
    () => (
      <IconPicker
        value={icon}
        onChange={handleIconChange}
        onFileSelected={handleIconFileSelected}
        maxFileSizeKB={64}
        triggerSize="sm"
      />
    ),
    [handleIconChange, handleIconFileSelected, icon]
  )

  const currencyOptions = useMemo(() => {
    if (userCurrencies.length > 0) {
      return userCurrencies.map((item) => ({
        code: item.code,
        label:
          item.alias.trim() || getPresetCurrencyMeta(item.code, i18n.language)?.alias || item.code,
      }))
    }

    return DEFAULT_CURRENCY_FALLBACK.map((code) => ({
      code,
      label: getPresetCurrencyMeta(code, i18n.language)?.alias || code,
    }))
  }, [i18n.language, userCurrencies])

  useEffect(() => {
    const isOpening = open && !wasOpenRef.current

    if (isOpening) {
      setError("")
      setLoading(false)
      setIconFile(null)

      if (subscription) {
        setName(subscription.name)
        setAmount(subscription.amount.toString())
        setCurrency(subscription.currency || currencyOptions[0]?.code || DEFAULT_CURRENCY_FALLBACK[0] || "")
        setEnabled(subscription.enabled)
        setBillingType(subscription.billing_type || "recurring")
        setRecurrenceType(subscription.recurrence_type || "interval")
        setIntervalCount((subscription.interval_count ?? 1).toString())
        setIntervalUnit(subscription.interval_unit || "month")
        setBillingAnchorDate(formatDateInput(subscription.billing_anchor_date))
        setMonthlyDay((subscription.monthly_day ?? new Date().getDate()).toString())
        setYearlyMonth((subscription.yearly_month ?? new Date().getMonth() + 1).toString())
        setYearlyDay((subscription.yearly_day ?? new Date().getDate()).toString())
        setTrialEnabled(subscription.trial_enabled)
        setTrialStartDate(formatDateInput(subscription.trial_start_date))
        setTrialEndDate(formatDateInput(subscription.trial_end_date))
        setCategoryId(subscription.category_id?.toString() || "")
        setPaymentMethodId(subscription.payment_method_id?.toString() || "")
        setIcon(subscription.icon || "")
        setUrl(subscription.url || "")
        setNotes(subscription.notes || "")
      } else {
        setName("")
        setAmount("")
        setCurrency(currencyOptions[0]?.code || DEFAULT_CURRENCY_FALLBACK[0] || "")
        setEnabled(true)
        setBillingType("recurring")
        setRecurrenceType("interval")
        setIntervalCount("1")
        setIntervalUnit("month")
        setBillingAnchorDate(new Date().toISOString().split("T")[0])
        setMonthlyDay(new Date().getDate().toString())
        setYearlyMonth((new Date().getMonth() + 1).toString())
        setYearlyDay(new Date().getDate().toString())
        setTrialEnabled(false)
        setTrialStartDate(new Date().toISOString().split("T")[0])
        setTrialEndDate(new Date().toISOString().split("T")[0])
        setCategoryId("")
        setPaymentMethodId("")
        setIcon("")
        setUrl("")
        setNotes("")
      }
    }

    if (!open) {
      setIconFile(null)
    }

    wasOpenRef.current = open
  }, [currencyOptions, open, subscription])

  useEffect(() => {
    if (!open || currencyOptions.length === 0) {
      return
    }

    if (!currencyOptions.some((option) => option.code === currency)) {
      setCurrency(currencyOptions[0].code)
    }
  }, [currency, currencyOptions, open])

  useEffect(() => {
    if (!open || !paymentMethodId) {
      return
    }

    const exists = paymentMethods.some((item) => item.id.toString() === paymentMethodId)
    if (!exists) {
      setPaymentMethodId("")
    }
  }, [open, paymentMethodId, paymentMethods])

  async function handleSubmit(event: FormEvent) {
    event.preventDefault()
    setError("")
    setLoading(true)

    try {
      let iconValue = icon

      if (iconFile && isEditing && subscription?.id) {
        const formData = new FormData()
        formData.append("icon", iconFile)
        const result = await api.uploadFile<UploadIconResponse>(
          `/subscriptions/${subscription.id}/icon`,
          formData
        )
        iconValue = result.icon
      }

      const normalizedRecurrenceType = billingType === "recurring" ? recurrenceType : ""
      const payload: CreateSubscriptionInput = {
        name,
        amount: parseFloat(amount),
        currency,
        enabled,
        billing_type: billingType,
        recurrence_type: normalizedRecurrenceType,
        interval_count:
          billingType === "recurring" && recurrenceType === "interval"
            ? parseInt(intervalCount, 10)
            : null,
        interval_unit:
          billingType === "recurring" && recurrenceType === "interval" ? intervalUnit : "",
        billing_anchor_date: billingAnchorDate,
        monthly_day:
          billingType === "recurring" && recurrenceType === "monthly_date"
            ? parseInt(monthlyDay, 10)
            : null,
        yearly_month:
          billingType === "recurring" && recurrenceType === "yearly_date"
            ? parseInt(yearlyMonth, 10)
            : null,
        yearly_day:
          billingType === "recurring" && recurrenceType === "yearly_date"
            ? parseInt(yearlyDay, 10)
            : null,
        trial_enabled: billingType === "recurring" ? trialEnabled : false,
        trial_start_date: billingType === "recurring" && trialEnabled ? trialStartDate : "",
        trial_end_date: billingType === "recurring" && trialEnabled ? trialEndDate : "",
        category: "",
        category_id: categoryId ? parseInt(categoryId, 10) : null,
        payment_method_id: paymentMethodId ? parseInt(paymentMethodId, 10) : null,
        icon: iconFile && !isEditing ? "" : iconValue,
        url,
        notes,
      }

      const created = await onSubmit(payload)

      if (iconFile && !isEditing && created?.id) {
        const formData = new FormData()
        formData.append("icon", iconFile)
        try {
          await api.uploadFile<UploadIconResponse>(`/subscriptions/${created.id}/icon`, formData)
        } catch {
          setError(t("subscription.form.iconUploadFailed"))
          setTimeout(() => onOpenChange(false), 1500)
          return
        }
      }

      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : t("subscription.form.error"))
    } finally {
      setLoading(false)
    }
  }

  const needsAnchorDate = billingType === "recurring" || billingType === "one_time"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? t("subscription.form.editTitle") : t("subscription.form.addTitle")}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}

          <div className="grid grid-cols-[auto_1fr] gap-3">
            <div className="space-y-2">
              <Label>{t("subscription.form.iconPicker.label")}</Label>
              {iconPickerNode}
            </div>
            <div className="space-y-2">
              <Label htmlFor="name">{t("subscription.form.nameLabel")}</Label>
              <Input
                id="name"
                placeholder={t("subscription.form.namePlaceholder")}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="amount">{t("subscription.form.amountLabel")}</Label>
              <Input
                id="amount"
                type="number"
                step="0.01"
                min="0.01"
                placeholder={t("subscription.form.amountPlaceholder")}
                value={amount}
                onChange={(event) => setAmount(event.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="currency">{t("subscription.form.currencyLabel")}</Label>
              <Select value={currency} onValueChange={setCurrency}>
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

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="billing-type">{t("subscription.form.billingTypeLabel")}</Label>
              <Select value={billingType} onValueChange={setBillingType}>
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
              <div className="inline-flex h-9 w-fit items-center rounded-md border px-3">
                <Switch id="enabled" checked={enabled} onCheckedChange={setEnabled} />
                <span className="ml-2 text-sm text-muted-foreground">
                  {enabled ? t("subscription.form.enabled") : t("subscription.form.disabled")}
                </span>
              </div>
            </div>
          </div>

          {needsAnchorDate && (
            <div className="space-y-2">
              <Label htmlFor="anchor-date">
                {billingType === "one_time"
                  ? t("subscription.form.purchaseDateLabel")
                  : t("subscription.form.anchorDateLabel")}
              </Label>
              <Input
                id="anchor-date"
                type="date"
                value={billingAnchorDate}
                onChange={(event) => setBillingAnchorDate(event.target.value)}
                required={needsAnchorDate}
              />
            </div>
          )}

          <SubscriptionRecurrenceFields
            billingType={billingType}
            recurrenceType={recurrenceType}
            onRecurrenceTypeChange={setRecurrenceType}
            intervalCount={intervalCount}
            onIntervalCountChange={setIntervalCount}
            intervalUnit={intervalUnit}
            onIntervalUnitChange={setIntervalUnit}
            monthlyDay={monthlyDay}
            onMonthlyDayChange={setMonthlyDay}
            yearlyMonth={yearlyMonth}
            onYearlyMonthChange={setYearlyMonth}
            yearlyDay={yearlyDay}
            onYearlyDayChange={setYearlyDay}
            trialEnabled={trialEnabled}
            onTrialEnabledChange={setTrialEnabled}
            trialStartDate={trialStartDate}
            onTrialStartDateChange={setTrialStartDate}
            trialEndDate={trialEndDate}
            onTrialEndDateChange={setTrialEndDate}
          />

          <SubscriptionMetadataFields
            categories={categories}
            categoryId={categoryId}
            onCategoryIdChange={setCategoryId}
            paymentMethods={paymentMethods}
            paymentMethodId={paymentMethodId}
            onPaymentMethodIdChange={setPaymentMethodId}
            url={url}
            onURLChange={setUrl}
            notes={notes}
            onNotesChange={setNotes}
          />

          <div className="flex gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              className="flex-1"
              onClick={() => onOpenChange(false)}
            >
              {t("subscription.form.cancel")}
            </Button>
            <Button type="submit" className="flex-1" disabled={loading}>
              {loading
                ? t("subscription.form.saving")
                : isEditing
                  ? t("subscription.form.update")
                  : t("subscription.form.addButton")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
