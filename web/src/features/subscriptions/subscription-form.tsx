import { useState, useEffect, useMemo, useRef, useCallback, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { api } from "@/lib/api"
import { DEFAULT_CURRENCY_FALLBACK, getPresetCurrencyMeta } from "@/lib/currencies"
import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import IconPicker from "./icon-picker"
import type {
  Subscription,
  CreateSubscriptionInput,
  UserCurrency,
  UploadIconResponse,
  Category,
  PaymentMethod,
} from "@/types"

interface SubscriptionFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  subscription?: Subscription | null
  onSubmit: (data: CreateSubscriptionInput) => Promise<Subscription>
}

const noPaymentMethodValue = "__none__"

function formatDateInput(value: string | null | undefined): string {
  if (!value) {
    return new Date().toISOString().split("T")[0]
  }
  return new Date(value).toISOString().split("T")[0]
}

export default function SubscriptionForm({
  open,
  onOpenChange,
  subscription,
  onSubmit,
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
  const [yearlyMonth, setYearlyMonth] = useState((subscription?.yearly_month ?? (new Date().getMonth() + 1)).toString())
  const [yearlyDay, setYearlyDay] = useState((subscription?.yearly_day ?? new Date().getDate()).toString())
  const [trialEnabled, setTrialEnabled] = useState(subscription?.trial_enabled ?? false)
  const [trialStartDate, setTrialStartDate] = useState(formatDateInput(subscription?.trial_start_date))
  const [trialEndDate, setTrialEndDate] = useState(formatDateInput(subscription?.trial_end_date))
  const [categoryId, setCategoryId] = useState<string>(
    subscription?.category_id?.toString() || ""
  )
  const [paymentMethodId, setPaymentMethodId] = useState<string>(
    subscription?.payment_method_id?.toString() || ""
  )
  const [icon, setIcon] = useState(subscription?.icon || "")
  const [url, setUrl] = useState(subscription?.url || "")
  const [notes, setNotes] = useState(subscription?.notes || "")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [iconFile, setIconFile] = useState<File | null>(null)
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([])
  const wasOpenRef = useRef(false)

  const handleIconChange = useCallback((value: string) => {
    setIcon(value)
    setIconFile(null)
  }, [])

  const handleIconFileSelected = useCallback((file: File) => {
    setIconFile(file)
    setIcon("")
  }, [])

  const iconPickerNode = useMemo(() => (
    <IconPicker
      value={icon}
      onChange={handleIconChange}
      onFileSelected={handleIconFileSelected}
      maxFileSizeKB={64}
      triggerSize="sm"
    />
  ), [icon, handleIconChange, handleIconFileSelected])

  const currencyOptions = useMemo(() => {
    if (userCurrencies.length > 0) {
      return userCurrencies.map((item) => ({
        code: item.code,
        label: item.alias.trim() || getPresetCurrencyMeta(item.code, i18n.language)?.alias || item.code,
      }))
    }

    return DEFAULT_CURRENCY_FALLBACK.map((code) => ({
      code,
      label: getPresetCurrencyMeta(code, i18n.language)?.alias || code,
    }))
  }, [i18n.language, userCurrencies])

  useEffect(() => {
    api.get<UserCurrency[]>("/currencies")
      .then((list) => {
        if (list && list.length > 0) {
          setUserCurrencies(list)
        }
      })
      .catch(() => void 0)

    api.get<Category[]>("/categories")
      .then((list) => {
        setCategories(list ?? [])
      })
      .catch(() => void 0)

    api.get<PaymentMethod[]>("/payment-methods")
      .then((list) => {
        setPaymentMethods(list ?? [])
      })
      .catch(() => void 0)
  }, [])

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
        setYearlyMonth((subscription.yearly_month ?? (new Date().getMonth() + 1)).toString())
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

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
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
        interval_count: billingType === "recurring" && recurrenceType === "interval" ? parseInt(intervalCount, 10) : null,
        interval_unit: billingType === "recurring" && recurrenceType === "interval" ? intervalUnit : "",
        billing_anchor_date: billingAnchorDate,
        monthly_day: billingType === "recurring" && recurrenceType === "monthly_date" ? parseInt(monthlyDay, 10) : null,
        yearly_month: billingType === "recurring" && recurrenceType === "yearly_date" ? parseInt(yearlyMonth, 10) : null,
        yearly_day: billingType === "recurring" && recurrenceType === "yearly_date" ? parseInt(yearlyDay, 10) : null,
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
          await api.uploadFile<UploadIconResponse>(
            `/subscriptions/${created.id}/icon`,
            formData
          )
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

  const isRecurring = billingType === "recurring"
  const needsAnchorDate =
    billingType === "recurring" ||
    billingType === "one_time"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>{isEditing ? t("subscription.form.editTitle") : t("subscription.form.addTitle")}</DialogTitle>
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
                onChange={(e) => setName(e.target.value)}
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
                onChange={(e) => setAmount(e.target.value)}
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
                    <SelectItem value="recurring">{t("subscription.form.billingType.recurring")}</SelectItem>
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
                onChange={(e) => setBillingAnchorDate(e.target.value)}
                required={needsAnchorDate}
              />
            </div>
          )}

          {isRecurring && (
            <>
              <div className="space-y-2">
                <div className="grid grid-cols-[11rem_minmax(0,1fr)] items-start gap-2">
                  <div className="space-y-1">
                    <Label className="flex h-4 items-center text-xs" htmlFor="recurrence-type">{t("subscription.form.recurrenceTypeLabel")}</Label>
                    <Select value={recurrenceType} onValueChange={setRecurrenceType}>
                      <SelectTrigger id="recurrence-type">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="interval">{t("subscription.form.recurrenceType.interval")}</SelectItem>
                        <SelectItem value="monthly_date">{t("subscription.form.recurrenceType.monthly_date")}</SelectItem>
                        <SelectItem value="yearly_date">{t("subscription.form.recurrenceType.yearly_date")}</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  {recurrenceType === "interval" && (
                    <div className="flex min-w-0 items-start gap-2">
                      <div className="w-24 shrink-0 space-y-1">
                        <Label className="flex h-4 items-center text-xs" htmlFor="interval-count">{t("subscription.form.intervalCountLabel")}</Label>
                        <Input
                          id="interval-count"
                          className="w-24 shrink-0"
                          type="number"
                          min="1"
                          step="1"
                          value={intervalCount}
                          onChange={(e) => setIntervalCount(e.target.value)}
                          required
                        />
                      </div>
                      <div className="min-w-[132px] flex-1 space-y-1">
                        <Label className="flex h-4 items-center text-xs" htmlFor="interval-unit">{t("subscription.form.intervalUnitLabel")}</Label>
                        <Select value={intervalUnit} onValueChange={setIntervalUnit}>
                          <SelectTrigger id="interval-unit">
                            <SelectValue />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="day">{t("subscription.form.intervalUnit.day")}</SelectItem>
                            <SelectItem value="week">{t("subscription.form.intervalUnit.week")}</SelectItem>
                            <SelectItem value="month">{t("subscription.form.intervalUnit.month")}</SelectItem>
                            <SelectItem value="year">{t("subscription.form.intervalUnit.year")}</SelectItem>
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                  )}

                  {recurrenceType === "monthly_date" && (
                    <div className="w-32 space-y-1">
                      <Label className="flex h-4 items-center text-xs" htmlFor="monthly-day">{t("subscription.form.monthlyDayLabel")}</Label>
                      <Input
                        id="monthly-day"
                        className="w-32"
                        type="number"
                        min="1"
                        max="31"
                        step="1"
                        value={monthlyDay}
                        onChange={(e) => setMonthlyDay(e.target.value)}
                        required
                      />
                    </div>
                  )}

                  {recurrenceType === "yearly_date" && (
                    <div className="flex items-start gap-2">
                      <div className="w-24 space-y-1">
                        <Label className="flex h-4 items-center text-xs" htmlFor="yearly-month">{t("subscription.form.yearlyMonthLabel")}</Label>
                        <Input
                          id="yearly-month"
                          className="w-24"
                          type="number"
                          min="1"
                          max="12"
                          step="1"
                          value={yearlyMonth}
                          onChange={(e) => setYearlyMonth(e.target.value)}
                          required
                        />
                      </div>
                      <div className="w-24 space-y-1">
                        <Label className="flex h-4 items-center text-xs" htmlFor="yearly-day">{t("subscription.form.yearlyDayLabel")}</Label>
                        <Input
                          id="yearly-day"
                          className="w-24"
                          type="number"
                          min="1"
                          max="31"
                          step="1"
                          value={yearlyDay}
                          onChange={(e) => setYearlyDay(e.target.value)}
                          required
                        />
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </>
          )}

          {isRecurring && (
            <>
              <div className="space-y-2">
                <Label htmlFor="trial-enabled">{t("subscription.form.trialLabel")}</Label>
                <div className="inline-flex h-9 w-fit items-center rounded-md border px-3">
                  <Switch id="trial-enabled" checked={trialEnabled} onCheckedChange={setTrialEnabled} />
                  <span className="ml-2 text-sm text-muted-foreground">
                    {trialEnabled ? t("subscription.form.enabled") : t("subscription.form.disabled")}
                  </span>
                </div>
              </div>

              {trialEnabled && (
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label htmlFor="trial-start">{t("subscription.form.trialStartLabel")}</Label>
                    <Input
                      id="trial-start"
                      type="date"
                      value={trialStartDate}
                      onChange={(e) => setTrialStartDate(e.target.value)}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="trial-end">{t("subscription.form.trialEndLabel")}</Label>
                    <Input
                      id="trial-end"
                      type="date"
                      value={trialEndDate}
                      onChange={(e) => setTrialEndDate(e.target.value)}
                      required
                    />
                  </div>
                </div>
              )}
            </>
          )}

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="category">{t("subscription.form.categoryLabel")}</Label>
              <Select value={categoryId} onValueChange={setCategoryId}>
                <SelectTrigger id="category">
                  <SelectValue placeholder={t("subscription.form.categoryPlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c.id} value={c.id.toString()}>{getCategoryLabel(c, t)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="payment-method">{t("subscription.form.paymentMethodLabel")}</Label>
              <Select
                value={paymentMethodId || noPaymentMethodValue}
                onValueChange={(value) => {
                  setPaymentMethodId(value === noPaymentMethodValue ? "" : value)
                }}
              >
                <SelectTrigger id="payment-method">
                  <SelectValue placeholder={t("subscription.form.paymentMethodPlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={noPaymentMethodValue}>{t("subscription.form.noPaymentMethod")}</SelectItem>
                  {paymentMethods.map((method) => (
                    <SelectItem key={method.id} value={method.id.toString()}>{getPaymentMethodLabel(method, t)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="url">{t("subscription.form.urlLabel")}</Label>
            <Input
              id="url"
              type="url"
              placeholder={t("subscription.form.urlPlaceholder")}
              value={url}
              onChange={(e) => setUrl(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="notes">{t("subscription.form.notesLabel")}</Label>
            <Input
              id="notes"
              placeholder={t("subscription.form.notesPlaceholder")}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
            />
          </div>

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
              {loading ? t("subscription.form.saving") : isEditing ? t("subscription.form.update") : t("subscription.form.addButton")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
