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
import type { Subscription, CreateSubscriptionInput, UserCurrency, UploadIconResponse } from "@/types"

interface SubscriptionFormProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  subscription?: Subscription | null
  onSubmit: (data: CreateSubscriptionInput) => Promise<Subscription>
}

const categories = [
  "Entertainment",
  "Productivity",
  "Development",
  "Music",
  "Cloud",
  "Finance",
  "Health",
  "Education",
  "News",
  "Other",
]

const colors = [
  "#18181b", "#dc2626", "#ea580c", "#ca8a04",
  "#16a34a", "#0891b2", "#2563eb", "#7c3aed",
  "#db2777", "#64748b",
]

export default function SubscriptionForm({
  open,
  onOpenChange,
  subscription,
  onSubmit,
}: SubscriptionFormProps) {
  const { t } = useTranslation()
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
  const [category, setCategory] = useState(subscription?.category || "")
  const [icon, setIcon] = useState(subscription?.icon || "")
  const [url, setUrl] = useState(subscription?.url || "")
  const [notes, setNotes] = useState(subscription?.notes || "")
  const [color, setColor] = useState(subscription?.color || "#18181b")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [iconFile, setIconFile] = useState<File | null>(null)
  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const wasOpenRef = useRef(false)

  const currencyOptions = useMemo(() => {
    if (userCurrencies.length > 0) {
      return userCurrencies.map((item) => ({
        code: item.code,
        label: item.alias.trim() || getPresetCurrencyMeta(item.code)?.alias || item.code,
      }))
    }

    return DEFAULT_CURRENCY_FALLBACK.map((code) => ({
      code,
      label: getPresetCurrencyMeta(code)?.alias || code,
    }))
  }, [userCurrencies])

  useEffect(() => {
    api.get<UserCurrency[]>("/currencies")
      .then((list) => {
        if (list && list.length > 0) {
          setUserCurrencies(list)
        }
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
        category,
        icon: iconFile && !isEditing ? "" : iconValue,
        url,
        notes,
        color,
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
              <Select value={category} onValueChange={setCategory}>
                <SelectTrigger id="category">
                  <SelectValue placeholder={t("subscription.form.categoryPlaceholder")} />
                </SelectTrigger>
                <SelectContent>
                  {categories.map((c) => (
                    <SelectItem key={c} value={c}>{t(`subscription.form.categories.${c.toLowerCase()}`)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
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
              />
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

          <div className="space-y-2">
            <Label>{t("subscription.form.colorLabel")}</Label>
            <div className="flex gap-2">
              {colors.map((c) => (
                <button
                  key={c}
                  type="button"
                  className={`h-7 w-7 rounded-full border-2 transition-all ${
                    color === c ? "border-foreground scale-110" : "border-transparent"
                  }`}
                  style={{ backgroundColor: c }}
                  onClick={() => setColor(c)}
                />
              ))}
            </div>
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
