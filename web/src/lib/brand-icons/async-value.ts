const BRAND_ICON_VALUE_PATTERN = /^(?:bl|custom|lg|sx):/i

export function isAsyncBrandIconValue(value: string): boolean {
  return BRAND_ICON_VALUE_PATTERN.test(value.trim())
}
