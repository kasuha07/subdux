import { useEffect, useState, type ReactNode } from "react"

import { isAsyncBrandIconValue } from "@/lib/brand-icons/async-value"
import type { BrandIconRuntime } from "@/lib/brand-icons/types"

type BrandIconLookup = (value: string) => BrandIconRuntime | undefined

const brandIconCache = new Map<string, BrandIconRuntime | null>()

let brandIconLookupPromise: Promise<BrandIconLookup> | null = null

function loadBrandIconLookup(): Promise<BrandIconLookup> {
  if (!brandIconLookupPromise) {
    brandIconLookupPromise = import("@/lib/brand-icons").then((module) => module.getBrandIconFromValue)
  }

  return brandIconLookupPromise
}

export function AsyncBrandIcon({
  className,
  color = "default",
  fallback,
  size = 20,
  value,
}: {
  className?: string
  color?: string
  fallback: ReactNode
  size?: number | string
  value: string
}) {
  const normalizedValue = value.trim()
  const [loadedValue, setLoadedValue] = useState("")
  const [loadedIcon, setLoadedIcon] = useState<BrandIconRuntime | null>(null)

  const cachedIcon = isAsyncBrandIconValue(normalizedValue)
    ? brandIconCache.get(normalizedValue)
    : null
  const resolvedIcon = cachedIcon !== undefined
    ? cachedIcon
    : loadedValue === normalizedValue
      ? loadedIcon
      : undefined

  useEffect(() => {
    if (!isAsyncBrandIconValue(normalizedValue)) {
      return
    }

    const cached = brandIconCache.get(normalizedValue)
    if (cached !== undefined) {
      return
    }

    let cancelled = false

    loadBrandIconLookup()
      .then((lookup) => {
        const icon = lookup(normalizedValue) ?? null
        brandIconCache.set(normalizedValue, icon)
        if (!cancelled) {
          setLoadedValue(normalizedValue)
          setLoadedIcon(icon)
        }
      })
      .catch(() => {
        brandIconCache.set(normalizedValue, null)
        if (!cancelled) {
          setLoadedValue(normalizedValue)
          setLoadedIcon(null)
        }
      })

    return () => {
      cancelled = true
    }
  }, [normalizedValue])

  if (resolvedIcon) {
    const Icon = resolvedIcon.Icon
    return <Icon size={size} color={color} className={className} />
  }

  return <>{fallback}</>
}
