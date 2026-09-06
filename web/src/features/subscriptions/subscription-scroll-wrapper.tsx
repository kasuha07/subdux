import { useEffect, useRef, type ReactNode } from "react"
import { cn } from "@/lib/utils"

interface SubscriptionScrollWrapperProps {
  children: ReactNode
  className?: string
  style?: React.CSSProperties
}

const supportsScrollTimeline =
  typeof CSS !== "undefined" &&
  typeof CSS.supports === "function" &&
  CSS.supports("animation-timeline", "view()")

const registeredElements = new Set<HTMLElement>()
let rafScheduled = false

function updateElements() {
  rafScheduled = false
  const windowHeight = window.innerHeight

  for (const el of registeredElements) {
    const rect = el.getBoundingClientRect()

    // When the bottom of the card is near or above the top edge (exit zone)
    if (rect.bottom <= 50 && rect.top < 20) {
      if (el.dataset.scrollState !== "exiting-top") {
        el.dataset.scrollState = "exiting-top"
      }
    } else if (rect.top >= windowHeight - 30) {
      // When the top of the card is near or below the bottom edge (enter zone)
      if (el.dataset.scrollState !== "entering-bottom") {
        el.dataset.scrollState = "entering-bottom"
      }
    } else {
      if (el.dataset.scrollState !== "in-view") {
        el.dataset.scrollState = "in-view"
      }
    }
  }
}

function requestUpdate() {
  if (rafScheduled) return
  rafScheduled = true
  requestAnimationFrame(updateElements)
}

let scrollListenerActive = false

function ensureScrollListener() {
  if (scrollListenerActive || typeof window === "undefined") return
  scrollListenerActive = true
  window.addEventListener("scroll", requestUpdate, { passive: true })
  window.addEventListener("resize", requestUpdate, { passive: true })
}

function cleanupScrollListenerIfEmpty() {
  if (!scrollListenerActive || registeredElements.size > 0 || typeof window === "undefined") return
  scrollListenerActive = false
  window.removeEventListener("scroll", requestUpdate)
  window.removeEventListener("resize", requestUpdate)
}

export default function SubscriptionScrollWrapper({
  children,
  className,
  style,
}: SubscriptionScrollWrapperProps) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el || supportsScrollTimeline) return

    registeredElements.add(el)
    ensureScrollListener()
    requestUpdate()

    return () => {
      registeredElements.delete(el)
      cleanupScrollListenerIfEmpty()
    }
  }, [])

  return (
    <div
      ref={ref}
      data-scroll-state="in-view"
      className={cn("subscription-scroll-wrap", className)}
      style={style}
    >
      {children}
    </div>
  )
}
