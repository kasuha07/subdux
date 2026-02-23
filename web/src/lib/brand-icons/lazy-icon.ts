import { createElement, useEffect, useState } from "react"

import type { BrandIconComponent, SvgIconComponent } from "./types"

export function createLazySvgIcon(loadIcon: () => Promise<SvgIconComponent>): BrandIconComponent {
  let loadedIcon: SvgIconComponent | null = null
  let loadingPromise: Promise<SvgIconComponent> | null = null

  function LazySvgIcon({ size = 20, color, className }: { size?: number | string; color?: string; className?: string }) {
    const [Icon, setIcon] = useState<SvgIconComponent | null>(() => loadedIcon)
    const resolvedIcon = Icon ?? loadedIcon

    useEffect(() => {
      let cancelled = false

      if (resolvedIcon) {
        return
      }

      if (!loadingPromise) {
        loadingPromise = loadIcon().then((nextIcon) => {
          loadedIcon = nextIcon
          return nextIcon
        })
      }

      loadingPromise
        .then((nextIcon) => {
          if (!cancelled) {
            setIcon(() => nextIcon)
          }
        })
        .catch(() => {
          if (!cancelled) {
            setIcon(null)
          }
        })

      return () => {
        cancelled = true
      }
    }, [resolvedIcon])

    const resolvedSize = normalizeSize(size)

    if (!resolvedIcon) {
      return createElement("span", {
        className,
        style: {
          width: resolvedSize,
          height: resolvedSize,
          display: "inline-block",
          borderRadius: 4,
          backgroundColor: "var(--muted)",
        },
      })
    }

    const resolvedColor = color === "default" ? undefined : color

    return createElement(resolvedIcon, {
      width: resolvedSize,
      height: resolvedSize,
      color: resolvedColor,
      className,
    })
  }

  return LazySvgIcon
}

function normalizeSize(size: number | string): number {
  if (typeof size === "number") {
    return Number.isFinite(size) ? size : 20
  }

  const parsed = Number.parseFloat(size)
  return Number.isFinite(parsed) ? parsed : 20
}
