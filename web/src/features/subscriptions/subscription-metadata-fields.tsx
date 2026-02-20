import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { getCategoryLabel, getPaymentMethodLabel } from "@/lib/preset-labels"
import type { Category, PaymentMethod } from "@/types"

const noPaymentMethodValue = "__none__"

interface SubscriptionMetadataFieldsProps {
  categories: Category[]
  categoryId: string
  notes: string
  onCategoryIdChange: (value: string) => void
  onNotesChange: (value: string) => void
  onPaymentMethodIdChange: (value: string) => void
  onURLChange: (value: string) => void
  paymentMethodId: string
  paymentMethods: PaymentMethod[]
  url: string
}

export default function SubscriptionMetadataFields({
  categories,
  categoryId,
  notes,
  onCategoryIdChange,
  onNotesChange,
  onPaymentMethodIdChange,
  onURLChange,
  paymentMethodId,
  paymentMethods,
  url,
}: SubscriptionMetadataFieldsProps) {
  const { t } = useTranslation()

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label htmlFor="category">{t("subscription.form.categoryLabel")}</Label>
          <Select value={categoryId} onValueChange={onCategoryIdChange}>
            <SelectTrigger id="category">
              <SelectValue placeholder={t("subscription.form.categoryPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              {categories.map((category) => (
                <SelectItem key={category.id} value={category.id.toString()}>
                  {getCategoryLabel(category, t)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="payment-method">{t("subscription.form.paymentMethodLabel")}</Label>
          <Select
            value={paymentMethodId || noPaymentMethodValue}
            onValueChange={(value) => {
              onPaymentMethodIdChange(value === noPaymentMethodValue ? "" : value)
            }}
          >
            <SelectTrigger id="payment-method">
              <SelectValue placeholder={t("subscription.form.paymentMethodPlaceholder")} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={noPaymentMethodValue}>{t("subscription.form.noPaymentMethod")}</SelectItem>
              {paymentMethods.map((method) => (
                <SelectItem key={method.id} value={method.id.toString()}>
                  {getPaymentMethodLabel(method, t)}
                </SelectItem>
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
          onChange={(event) => onURLChange(event.target.value)}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="notes">{t("subscription.form.notesLabel")}</Label>
        <Input
          id="notes"
          placeholder={t("subscription.form.notesPlaceholder")}
          value={notes}
          onChange={(event) => onNotesChange(event.target.value)}
        />
      </div>
    </>
  )
}
