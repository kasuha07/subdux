import { useEffect, useRef, useState } from "react"

const defaultEasing = (t: number) => 1 - Math.pow(1 - t, 3)

export interface UseAnimatedNumberOptions {
  /**
   * Animation duration in milliseconds. Defaults to 700ms.
   */
  duration?: number
  /**
   * If true, animation is disabled and target is returned immediately.
   */
  disabled?: boolean
  /**
   * Starting number on initial mount. Defaults to 0.
   */
  from?: number
  /**
   * Easing function mapping progress [0, 1] to eased progress [0, 1].
   */
  easing?: (t: number) => number
}

/**
 * Hook to animate a numeric value from an initial value (default 0)
 * to a target value using requestAnimationFrame and cubic ease-out.
 * Respects `prefers-reduced-motion` and avoids animation on the server / static rendering.
 */
export function useAnimatedNumber(
  target: number,
  options?: UseAnimatedNumberOptions
): number {
  const duration = options?.duration ?? 700
  const easing = options?.easing ?? defaultEasing
  const disabled = options?.disabled ?? false
  const from = options?.from ?? 0

  const isClient = typeof window !== "undefined"
  const prefersReducedMotion =
    isClient &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-reduced-motion: reduce)").matches

  const shouldAnimate = isClient && !disabled && !prefersReducedMotion

  const [current, setCurrent] = useState(() => (shouldAnimate ? from : target))
  const currentValueRef = useRef(shouldAnimate ? from : target)

  useEffect(() => {
    if (!shouldAnimate) {
      currentValueRef.current = target
      return
    }

    const startValue = currentValueRef.current
    const delta = target - startValue

    if (Math.abs(delta) < 0.0001) {
      return
    }

    let startTime: number | null = null
    let rafId: number

    const step = (timestamp: number) => {
      if (startTime === null) {
        startTime = timestamp
      }
      const elapsed = timestamp - startTime
      const progress = Math.min(elapsed / duration, 1)
      const eased = easing(progress)
      const nextVal = startValue + delta * eased

      if (progress < 1) {
        setCurrent(nextVal)
        currentValueRef.current = nextVal
        rafId = requestAnimationFrame(step)
      } else {
        setCurrent(target)
        currentValueRef.current = target
      }
    }

    rafId = requestAnimationFrame(step)

    return () => {
      cancelAnimationFrame(rafId)
    }
  }, [target, duration, shouldAnimate, easing])

  return shouldAnimate ? current : target
}
