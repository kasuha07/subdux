import { useSyncExternalStore } from "react"

/**
 * Detects whether the current environment is a mobile device using modern
 * browser Client Hints, standard UserAgent patterns, iPadOS touch heuristics,
 * and CSS pointer/hover media queries.
 */
export function isMobileDevice(): boolean {
  if (typeof window === "undefined" || typeof navigator === "undefined") {
    return false
  }

  // 1. User-Agent Client Hints API (Chromium / modern Android)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const nav = navigator as any
  if (typeof nav.userAgentData?.mobile === "boolean") {
    return nav.userAgentData.mobile
  }

  const ua = navigator.userAgent || navigator.vendor || ""

  // 2. Standard Mobile User-Agent regex
  if (/android|iphone|ipad|ipod|blackberry|iemobile|opera mini|mobile/i.test(ua)) {
    return true
  }

  // 3. iPadOS on Safari (reports as Macintosh, but with multi-touch points)
  if (
    navigator.maxTouchPoints &&
    navigator.maxTouchPoints > 1 &&
    /macintosh/i.test(ua)
  ) {
    return true
  }

  // 4. Browser Media Query: primary pointer is coarse (touch) and cannot hover
  if (typeof window.matchMedia === "function") {
    try {
      const isTouchOnly = window.matchMedia("(pointer: coarse) and (hover: none)").matches
      if (isTouchOnly) {
        return true
      }
    } catch {
      // Ignore media query evaluation errors in legacy environments
    }
  }

  return false
}

function subscribeMobile(callback: () => void): () => void {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return () => {}
  }

  try {
    const mediaQuery = window.matchMedia("(pointer: coarse) and (hover: none)")
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener("change", callback)
      return () => mediaQuery.removeEventListener("change", callback)
    } else if ("addListener" in mediaQuery) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const legacyQuery = mediaQuery as any
      legacyQuery.addListener(callback)
      return () => legacyQuery.removeListener(callback)
    }
  } catch {
    // Ignore
  }

  return () => {}
}

/**
 * Hook that returns whether the current environment is a mobile device.
 * Uses useSyncExternalStore to subscribe to touch/pointer media query changes
 * without triggering cascading renders or hydration mismatches.
 */
export function useIsMobileDevice(): boolean {
  return useSyncExternalStore(subscribeMobile, isMobileDevice, isMobileDevice)
}
