import type { ReactNode } from "react"

import { AsyncBrandIcon } from "@/components/async-brand-icon"
import { isAsyncBrandIconValue } from "@/lib/brand-icons/async-value"
import { cn } from "@/lib/utils"

export interface SubscriptionIconProps {
  icon?: string | null
  name: string
  size?: number
  className?: string
  fallbackClassName?: string
}

export function SubscriptionIcon({
  icon,
  name,
  size = 24,
  className,
  fallbackClassName,
}: SubscriptionIconProps): ReactNode {
  const initial = name ? name.trim().charAt(0).toUpperCase() : "?"
  const fallbackInitial = (
    <span
      className={cn(
        "flex size-full items-center justify-center bg-muted text-sm font-bold text-foreground select-none",
        fallbackClassName
      )}
    >
      {initial}
    </span>
  )

  if (!icon || !icon.trim()) {
    return fallbackInitial
  }

  const trimmed = icon.trim()

  if (isAsyncBrandIconValue(trimmed)) {
    return (
      <AsyncBrandIcon
        value={trimmed}
        size={size}
        color="default"
        fallback={fallbackInitial}
      />
    )
  }

  if (
    trimmed.startsWith("http://") ||
    trimmed.startsWith("https://") ||
    trimmed.startsWith("/api/icon-proxy/")
  ) {
    return (
      <img
        src={trimmed}
        alt={name}
        className={cn("size-7 object-contain", className)}
        loading="lazy"
      />
    )
  }

  if (trimmed.startsWith("file:")) {
    const filename = trimmed.slice("file:".length)
    if (filename && !filename.includes("/") && !filename.includes("\\")) {
      return (
        <img
          src={`/uploads/icons/${filename}`}
          alt={name}
          className={cn("size-7 object-contain", className)}
          loading="lazy"
        />
      )
    }
  }

  if (trimmed.includes(":")) {
    return fallbackInitial
  }

  return (
    <span
      className={cn("text-lg leading-none select-none", className)}
      role="img"
      aria-label={name}
    >
      {trimmed}
    </span>
  )
}
