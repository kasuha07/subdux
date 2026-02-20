import { useState, useEffect, useMemo, useRef, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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
  const [billingCycle, setBillingCycle] = useState<string>(subscription?.billing_cycle || "monthly")
  const [nextBillingDate, setNextBillingDate] = useState(
    subscription?.next_billing_date
      ? new Date(subscription.next_billing_date).toISOString().split("T")[0]
      : new Date().toISOString().split("T")[0]
  )
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
    if (open && !wasOpenRef.current && !isEditing && currencyOptions.length > 0) {
      setCurrency(currencyOptions[0].code)
    }
    if (!open) {
      setIconFile(null)
    }

    wasOpenRef.current = open
  }, [currencyOptions, isEditing, open])

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

      const created = await onSubmit({
        name,
        amount: parseFloat(amount),
        currency,
        billing_cycle: billingCycle,
        next_billing_date: nextBillingDate,
        category: "",
        category_id: categoryId ? parseInt(categoryId, 10) : null,
        payment_method_id: paymentMethodId ? parseInt(paymentMethodId, 10) : null,
        icon: iconFile && !isEditing ? "" : iconValue,
        url,
        notes,
      })

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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
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
              <IconPicker
                value={icon}
                onChange={(v) => {
                  setIcon(v)
                  setIconFile(null)
                }}
                onFileSelected={(f) => {
                  setIconFile(f)
                  setIcon("")
                }}
                maxFileSizeKB={64}
                triggerSize="sm"
              />
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
                min="0"
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
              <Label htmlFor="cycle">{t("subscription.form.cycleLabel")}</Label>
              <Select value={billingCycle} onValueChange={setBillingCycle}>
                <SelectTrigger id="cycle">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="weekly">{t("subscription.form.cycle.weekly")}</SelectItem>
                  <SelectItem value="monthly">{t("subscription.form.cycle.monthly")}</SelectItem>
                  <SelectItem value="yearly">{t("subscription.form.cycle.yearly")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="next-billing">{t("subscription.form.nextBillingLabel")}</Label>
              <Input
                id="next-billing"
                type="date"
                value={nextBillingDate}
                onChange={(e) => setNextBillingDate(e.target.value)}
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="category">{t("subscription.form.categoryLabel")}</Label>
              <Select value={categoryId} onValueChange={setCategoryId}>
                <SelectTrigger id="category">
                  <SelectValue placeholder={t("subscription.form.categoryPlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c.id} value={c.id.toString()}>{c.name}</SelectItem>
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
                    <SelectItem key={method.id} value={method.id.toString()}>{method.name}</SelectItem>
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
