import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react"

import { api } from "@/lib/api"
import { DEFAULT_CURRENCY_FALLBACK, getPresetCurrencyMeta } from "@/lib/currencies"
import { formatDateKey } from "@/lib/utils"
import type {
  CreateSubscriptionInput,
  PaymentMethod,
  Subscription,
  SubscriptionRenewalMode,
  SubscriptionStatus,
  UploadIconResponse,
  UserCurrency,
} from "@/types"

export type SubscriptionNotifySetting = "default" | "enabled" | "disabled"

interface SubscriptionFormValues {
  amount: string
  nextBillingDate: string
  endsAt: string
  billingType: string
  categoryId: string
  currency: string
  icon: string
  intervalCount: string
  intervalUnit: string
  monthlyDay: string
  name: string
  notes: string
  notifyDaysBefore: string
  notifyEnabled: SubscriptionNotifySetting
  paymentMethodId: string
  recurrenceType: string
  renewalMode: SubscriptionRenewalMode
  status: SubscriptionStatus
  url: string
  yearlyDay: string
  yearlyMonth: string
}

interface UseSubscriptionFormStateOptions {
  language: string
  onMarkRenewed?: (subscription: Subscription) => Promise<Subscription>
  onOpenChange: (open: boolean) => void
  onSubmit: (data: CreateSubscriptionInput) => Promise<Subscription>
  open: boolean
  paymentMethods: PaymentMethod[]
  subscription?: Subscription | null
  t: (key: string) => string
  userCurrencies: UserCurrency[]
}

interface UseSubscriptionFormStateResult {
  currencyOptions: Array<{ code: string; label: string }>
  error: string
  handleIconChange: (value: string) => void
  handleIconFileSelected: (file: File) => void
  handleMarkRenewed: () => Promise<void>
  handleSubmit: (event: FormEvent) => Promise<void>
  iconFile: File | null
  isEditing: boolean
  loading: boolean
  setField: <K extends keyof SubscriptionFormValues>(
    field: K,
    value: SubscriptionFormValues[K]
  ) => void
  values: SubscriptionFormValues
}

const MAX_NOTIFICATION_DAYS_BEFORE = 10

function formatDateInput(value: string | null | undefined): string {
  if (!value) {
    return formatDateKey(new Date())
  }
  const match = /^(\d{4}-\d{2}-\d{2})/.exec(value.trim())
  if (match) {
    return match[1]
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return formatDateKey(new Date())
  }

  return formatDateKey(parsed)
}

function buildInitialValues(
  subscription: Subscription | null | undefined,
  fallbackCurrencyCode: string
): SubscriptionFormValues {
  const today = new Date()
  const todayDate = formatDateKey(today)

  if (subscription) {
    return {
      amount: subscription.amount.toString(),
      nextBillingDate: formatDateInput(subscription.next_billing_date),
      endsAt: formatDateInput(subscription.ends_at || subscription.next_billing_date),
      billingType: "recurring",
      categoryId: subscription.category_id?.toString() || "",
      currency: subscription.currency || fallbackCurrencyCode,
      icon: subscription.icon || "",
      intervalCount: (subscription.interval_count ?? 1).toString(),
      intervalUnit: subscription.interval_unit || "month",
      monthlyDay: (subscription.monthly_day ?? today.getDate()).toString(),
      name: subscription.name,
      notes: subscription.notes || "",
      notifyDaysBefore: subscription.notify_days_before?.toString() || "",
      notifyEnabled:
        subscription.notify_enabled === null
          ? "default"
          : subscription.notify_enabled
            ? "enabled"
            : "disabled",
      paymentMethodId: subscription.payment_method_id?.toString() || "",
      recurrenceType: subscription.recurrence_type || "interval",
      renewalMode:
        subscription.renewal_mode || "auto_renew",
      status: subscription.status || "active",
      url: subscription.url || "",
      yearlyDay: (subscription.yearly_day ?? today.getDate()).toString(),
      yearlyMonth: (subscription.yearly_month ?? today.getMonth() + 1).toString(),
    }
  }

  return {
    amount: "",
    nextBillingDate: todayDate,
    endsAt: todayDate,
    billingType: "recurring",
    categoryId: "",
    currency: fallbackCurrencyCode,
    icon: "",
    intervalCount: "1",
    intervalUnit: "month",
    monthlyDay: today.getDate().toString(),
    name: "",
    notes: "",
    notifyDaysBefore: "",
    notifyEnabled: "default",
    paymentMethodId: "",
    recurrenceType: "interval",
    renewalMode: "auto_renew",
    status: "active",
    url: "",
    yearlyDay: today.getDate().toString(),
    yearlyMonth: (today.getMonth() + 1).toString(),
  }
}

export function useSubscriptionFormState({
  language,
  onMarkRenewed,
  onOpenChange,
  onSubmit,
  open,
  paymentMethods,
  subscription,
  t,
  userCurrencies,
}: UseSubscriptionFormStateOptions): UseSubscriptionFormStateResult {
  const isEditing = !!subscription

  const currencyOptions = useMemo(() => {
    if (userCurrencies.length > 0) {
      return userCurrencies.map((item) => ({
        code: item.code,
        label: item.alias.trim() || getPresetCurrencyMeta(item.code, language)?.alias || item.code,
      }))
    }

    return DEFAULT_CURRENCY_FALLBACK.map((code) => ({
      code,
      label: getPresetCurrencyMeta(code, language)?.alias || code,
    }))
  }, [language, userCurrencies])

  const defaultCurrencyCode = currencyOptions[0]?.code || DEFAULT_CURRENCY_FALLBACK[0] || ""

  const [values, setValues] = useState<SubscriptionFormValues>(() =>
    buildInitialValues(subscription, defaultCurrencyCode)
  )
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [iconFile, setIconFile] = useState<File | null>(null)
  const wasOpenRef = useRef(false)

  const setField = useCallback(
    <K extends keyof SubscriptionFormValues>(field: K, value: SubscriptionFormValues[K]) => {
      setValues((prev) => ({
        ...prev,
        [field]: value,
      }))
    },
    []
  )

  const handleIconChange = useCallback((value: string) => {
    setField("icon", value)
    setIconFile(null)
  }, [setField])

  const handleIconFileSelected = useCallback((file: File) => {
    setIconFile(file)
    setField("icon", "")
  }, [setField])

  useEffect(() => {
    const isOpening = open && !wasOpenRef.current

    if (isOpening) {
      setError("")
      setLoading(false)
      setIconFile(null)
      setValues(buildInitialValues(subscription, defaultCurrencyCode))
    }

    if (!open) {
      setIconFile(null)
    }

    wasOpenRef.current = open
  }, [defaultCurrencyCode, open, subscription])

  useEffect(() => {
    if (!open || currencyOptions.length === 0) {
      return
    }

    if (!currencyOptions.some((option) => option.code === values.currency)) {
      setField("currency", currencyOptions[0].code)
    }
  }, [currencyOptions, open, setField, values.currency])

  useEffect(() => {
    if (!open || !values.paymentMethodId) {
      return
    }

    const exists = paymentMethods.some((item) => item.id.toString() === values.paymentMethodId)
    if (!exists) {
      setField("paymentMethodId", "")
    }
  }, [open, paymentMethods, setField, values.paymentMethodId])

  useEffect(() => {
    if (!open) {
      return
    }

    if (values.status === "active") {
      if (values.renewalMode === "cancel_at_period_end") {
        if (values.endsAt !== values.nextBillingDate) {
          setField("endsAt", values.nextBillingDate)
        }
      }
      return
    }

    if (!values.endsAt) {
      setField("endsAt", values.nextBillingDate || formatDateKey(new Date()))
    }
  }, [open, setField, values.endsAt, values.nextBillingDate, values.renewalMode, values.status])

  const handleMarkRenewed = useCallback(async () => {
    if (!subscription || !onMarkRenewed) {
      return
    }

    setError("")
    setLoading(true)
    try {
      await onMarkRenewed(subscription)
      onOpenChange(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : t("subscription.form.error"))
    } finally {
      setLoading(false)
    }
  }, [onMarkRenewed, onOpenChange, subscription, t])

  const handleSubmit = useCallback(async (event: FormEvent) => {
    event.preventDefault()
    setError("")
    setLoading(true)

    try {
      let iconValue = values.icon

      if (iconFile && isEditing && subscription?.id) {
        const formData = new FormData()
        formData.append("icon", iconFile)
        const result = await api.uploadFile<UploadIconResponse>(
          `/subscriptions/${subscription.id}/icon`,
          formData
        )
        iconValue = result.icon
      }

      const parsedNotifyDaysBefore = parseInt(values.notifyDaysBefore, 10)
      const payload: CreateSubscriptionInput = {
        name: values.name,
        amount: parseFloat(values.amount),
        currency: values.currency,
        status: values.status,
        renewal_mode: values.renewalMode,
        ends_at:
          values.status === "ended"
            ? values.endsAt
            : values.renewalMode === "cancel_at_period_end"
              ? values.nextBillingDate
              : null,
        billing_type: "recurring",
        recurrence_type: values.recurrenceType,
        interval_count:
          values.recurrenceType === "interval"
            ? parseInt(values.intervalCount, 10)
            : null,
        interval_unit:
          values.recurrenceType === "interval"
            ? values.intervalUnit
            : "",
        next_billing_date: values.nextBillingDate,
        monthly_day:
          values.recurrenceType === "monthly_date"
            ? parseInt(values.monthlyDay, 10)
            : null,
        yearly_month:
          values.recurrenceType === "yearly_date"
            ? parseInt(values.yearlyMonth, 10)
            : null,
        yearly_day:
          values.recurrenceType === "yearly_date"
            ? parseInt(values.yearlyDay, 10)
            : null,
        category: "",
        category_id: values.categoryId ? parseInt(values.categoryId, 10) : null,
        payment_method_id: values.paymentMethodId ? parseInt(values.paymentMethodId, 10) : null,
        notify_enabled: values.notifyEnabled === "default" ? null : values.notifyEnabled === "enabled",
        notify_days_before:
          values.notifyEnabled === "enabled" && values.notifyDaysBefore
            ? Number.isNaN(parsedNotifyDaysBefore)
              ? 0
              : Math.min(MAX_NOTIFICATION_DAYS_BEFORE, Math.max(0, parsedNotifyDaysBefore))
            : null,
        icon: iconFile && !isEditing ? "" : iconValue,
        url: values.url,
        notes: values.notes,
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
  }, [iconFile, isEditing, onOpenChange, onSubmit, subscription?.id, t, values])

  return {
    currencyOptions,
    error,
    handleIconChange,
    handleIconFileSelected,
    handleMarkRenewed,
    handleSubmit,
    iconFile,
    isEditing,
    loading,
    setField,
    values,
  }
}
