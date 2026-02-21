import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from "react"

import { api } from "@/lib/api"
import { DEFAULT_CURRENCY_FALLBACK, getPresetCurrencyMeta } from "@/lib/currencies"
import type {
  CreateSubscriptionInput,
  PaymentMethod,
  Subscription,
  UploadIconResponse,
  UserCurrency,
} from "@/types"

export type SubscriptionNotifySetting = "default" | "enabled" | "disabled"

interface SubscriptionFormValues {
  amount: string
  billingAnchorDate: string
  billingType: string
  categoryId: string
  currency: string
  enabled: boolean
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
  trialEnabled: boolean
  trialEndDate: string
  trialStartDate: string
  url: string
  yearlyDay: string
  yearlyMonth: string
}

interface UseSubscriptionFormStateOptions {
  language: string
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
  handleSubmit: (event: FormEvent) => Promise<void>
  iconFile: File | null
  isEditing: boolean
  loading: boolean
  needsAnchorDate: boolean
  setField: <K extends keyof SubscriptionFormValues>(
    field: K,
    value: SubscriptionFormValues[K]
  ) => void
  values: SubscriptionFormValues
}

function formatDateInput(value: string | null | undefined): string {
  if (!value) {
    return new Date().toISOString().split("T")[0]
  }
  return new Date(value).toISOString().split("T")[0]
}

function buildInitialValues(
  subscription: Subscription | null | undefined,
  fallbackCurrencyCode: string
): SubscriptionFormValues {
  const today = new Date()
  const todayDate = today.toISOString().split("T")[0]

  if (subscription) {
    return {
      amount: subscription.amount.toString(),
      billingAnchorDate: formatDateInput(subscription.billing_anchor_date),
      billingType: subscription.billing_type || "recurring",
      categoryId: subscription.category_id?.toString() || "",
      currency: subscription.currency || fallbackCurrencyCode,
      enabled: subscription.enabled,
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
      trialEnabled: subscription.trial_enabled,
      trialEndDate: formatDateInput(subscription.trial_end_date),
      trialStartDate: formatDateInput(subscription.trial_start_date),
      url: subscription.url || "",
      yearlyDay: (subscription.yearly_day ?? today.getDate()).toString(),
      yearlyMonth: (subscription.yearly_month ?? today.getMonth() + 1).toString(),
    }
  }

  return {
    amount: "",
    billingAnchorDate: todayDate,
    billingType: "recurring",
    categoryId: "",
    currency: fallbackCurrencyCode,
    enabled: true,
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
    trialEnabled: false,
    trialEndDate: todayDate,
    trialStartDate: todayDate,
    url: "",
    yearlyDay: today.getDate().toString(),
    yearlyMonth: (today.getMonth() + 1).toString(),
  }
}

export function useSubscriptionFormState({
  language,
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

  const needsAnchorDate = values.billingType === "recurring" || values.billingType === "one_time"

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

      const normalizedRecurrenceType = values.billingType === "recurring" ? values.recurrenceType : ""
      const payload: CreateSubscriptionInput = {
        name: values.name,
        amount: parseFloat(values.amount),
        currency: values.currency,
        enabled: values.enabled,
        billing_type: values.billingType,
        recurrence_type: normalizedRecurrenceType,
        interval_count:
          values.billingType === "recurring" && values.recurrenceType === "interval"
            ? parseInt(values.intervalCount, 10)
            : null,
        interval_unit:
          values.billingType === "recurring" && values.recurrenceType === "interval"
            ? values.intervalUnit
            : "",
        billing_anchor_date: values.billingAnchorDate,
        monthly_day:
          values.billingType === "recurring" && values.recurrenceType === "monthly_date"
            ? parseInt(values.monthlyDay, 10)
            : null,
        yearly_month:
          values.billingType === "recurring" && values.recurrenceType === "yearly_date"
            ? parseInt(values.yearlyMonth, 10)
            : null,
        yearly_day:
          values.billingType === "recurring" && values.recurrenceType === "yearly_date"
            ? parseInt(values.yearlyDay, 10)
            : null,
        trial_enabled: values.billingType === "recurring" ? values.trialEnabled : false,
        trial_start_date:
          values.billingType === "recurring" && values.trialEnabled ? values.trialStartDate : "",
        trial_end_date:
          values.billingType === "recurring" && values.trialEnabled ? values.trialEndDate : "",
        category: "",
        category_id: values.categoryId ? parseInt(values.categoryId, 10) : null,
        payment_method_id: values.paymentMethodId ? parseInt(values.paymentMethodId, 10) : null,
        notify_enabled: values.notifyEnabled === "default" ? null : values.notifyEnabled === "enabled",
        notify_days_before:
          values.notifyEnabled === "enabled" && values.notifyDaysBefore
            ? parseInt(values.notifyDaysBefore, 10)
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
    handleSubmit,
    iconFile,
    isEditing,
    loading,
    needsAnchorDate,
    setField,
    values,
  }
}
