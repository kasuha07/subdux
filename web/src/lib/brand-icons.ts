import { createLazySvgIcon } from "./brand-icons/lazy-icon"
import { brandSpecs } from "./brand-icons/specs"

import type { BrandIcon } from "./brand-icons/types"

export type { BrandIcon, BrandIconComponent } from "./brand-icons/types"

export const brandIcons: BrandIcon[] = brandSpecs.map((spec) => ({
  slug: spec.slug,
  value: `${spec.prefix}:${spec.slug}`,
  title: spec.title,
  hex: spec.hex,
  keywords: spec.keywords ?? [],
  Icon: createLazySvgIcon(spec.loadIcon),
}))

const brandIconMap = new Map(brandIcons.map((icon) => [icon.slug, icon] as const))
const brandIconValueMap = new Map(brandIcons.map((icon) => [icon.value, icon] as const))

export function getBrandIcon(slug: string): BrandIcon | undefined {
  return brandIconMap.get(slug)
}

export function getBrandIconFromValue(value: string): BrandIcon | undefined {
  if (!value) {
    return undefined
  }

  return brandIconValueMap.get(value)
}
