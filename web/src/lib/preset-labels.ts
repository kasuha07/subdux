import type { TFunction } from "i18next"
import type { Category, PaymentMethod } from "@/types"

function resolvePresetLabel(t: TFunction, key: string): string | null {
  const translated = t(key, { defaultValue: "" }).trim()
  if (translated) {
    return translated
  }
  return null
}

export function getCategoryLabel(category: Category, t: TFunction): string {
  if (category.system_key && !category.name_customized) {
    return resolvePresetLabel(t, `presets.category.${category.system_key}`) ?? category.name
  }
  return category.name
}

export function getPaymentMethodLabel(method: PaymentMethod, t: TFunction): string {
  if (method.system_key && !method.name_customized) {
    return resolvePresetLabel(t, `presets.payment_method.${method.system_key}`) ?? method.name
  }
  return method.name
}
