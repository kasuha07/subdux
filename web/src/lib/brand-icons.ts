import { createLazySvgIcon } from "./brand-icons/lazy-icon"
import { brandRuntimeSpecs } from "./brand-icons/runtime-specs"

import type { BrandIcon, BrandIconRuntime } from "./brand-icons/types"

export type { BrandIcon, BrandIconRuntime, BrandIconComponent } from "./brand-icons/types"

const brandRuntimeIcons: BrandIconRuntime[] = brandRuntimeSpecs.map((spec) => ({
  slug: spec.slug,
  value: `${spec.prefix}:${spec.slug}`,
  Icon: createLazySvgIcon(spec.loadIcon),
}))

const brandIconMap = new Map(brandRuntimeIcons.map((icon) => [icon.slug, icon] as const))
const brandIconValueMap = new Map(brandRuntimeIcons.map((icon) => [icon.value, icon] as const))

const legacyIconValueAliases = new Map<string, string>([
  ["lg:adobecreativecloud", "custom:adobecreativecloud"],
  ["lg:bilibili", "custom:bilibili"],
  ["lg:bitwarden", "custom:bitwarden"],
  ["lg:chatgpt", "lg:openai"],
  ["lg:coinbase", "custom:coinbase"],
  ["lg:godaddy", "custom:godaddy"],
  ["lg:icloud", "custom:icloud"],
  ["lg:kugoumusic", "custom:kugoumusic"],
  ["lg:kuwomusic", "custom:qqmusic"],
  ["lg:nintendo", "custom:nintendo"],
  ["lg:qqmusic", "custom:qqmusic"],
  ["lg:tencentvideo", "custom:tencentvideo"],
  ["lg:times", "lg:nytimes"],
  ["lg:x", "custom:x"],
  ["lg:xpremium", "custom:x"],
  ["lg:youtubemusic", "custom:youtubemusic"],
  ["sx:neteasecloudmusic", "custom:neteasecloudmusic"],
])

let brandIconCatalogPromise: Promise<BrandIcon[]> | null = null

export function getBrandIcon(slug: string): BrandIconRuntime | undefined {
  return brandIconMap.get(slug)
}

export function getBrandIconFromValue(value: string): BrandIconRuntime | undefined {
  if (!value) {
    return undefined
  }

  const icon = brandIconValueMap.get(value)
  if (icon) {
    return icon
  }

  const aliasValue = legacyIconValueAliases.get(value)
  if (!aliasValue) {
    return undefined
  }

  return brandIconValueMap.get(aliasValue)
}

export function loadBrandIconsCatalog(): Promise<BrandIcon[]> {
  if (!brandIconCatalogPromise) {
    brandIconCatalogPromise = import("./brand-icons/specs").then(({ brandSpecs }) => (
      brandSpecs.map((spec) => {
        const value = `${spec.prefix}:${spec.slug}`
        const runtimeIcon = brandIconValueMap.get(value)

        return {
          slug: spec.slug,
          value,
          title: spec.title,
          hex: spec.hex,
          keywords: spec.keywords ?? [],
          Icon: runtimeIcon?.Icon ?? createLazySvgIcon(spec.loadIcon),
        }
      })
    ))
  }

  return brandIconCatalogPromise
}
