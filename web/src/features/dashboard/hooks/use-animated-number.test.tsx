// @vitest-environment happy-dom
import { act, StrictMode } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { useAnimatedNumber, type UseAnimatedNumberOptions } from "./use-animated-number"

function NumberHarness({
  target,
  options,
}: {
  target: number
  options?: UseAnimatedNumberOptions
}) {
  const value = useAnimatedNumber(target, options)
  return <span data-testid="value">{value}</span>
}

describe("useAnimatedNumber", () => {
  let container: HTMLDivElement
  let root: Root
  let rafCallbacks: Map<number, FrameRequestCallback>
  let nextRafId: number

  beforeEach(() => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    rafCallbacks = new Map()
    nextRafId = 1

    vi.spyOn(window, "requestAnimationFrame").mockImplementation((cb: FrameRequestCallback) => {
      const id = nextRafId++
      rafCallbacks.set(id, cb)
      return id
    })

    vi.spyOn(window, "cancelAnimationFrame").mockImplementation((id: number) => {
      rafCallbacks.delete(id)
    })

    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => {
      root.unmount()
    })
    container.remove()
    vi.restoreAllMocks()
  })

  const triggerRaf = async (timestamp: number) => {
    await act(async () => {
      const callbacks = Array.from(rafCallbacks.entries())
      rafCallbacks.clear()
      for (const [, cb] of callbacks) {
        cb(timestamp)
      }
    })
  }

  const getValue = () => {
    const el = container.querySelector('[data-testid="value"]')
    return el ? parseFloat(el.textContent || "0") : 0
  }

  it("returns target immediately when disabled", async () => {
    await act(async () => {
      root.render(
        <StrictMode>
          <NumberHarness target={120} options={{ disabled: true }} />
        </StrictMode>
      )
    })

    expect(getValue()).toBe(120)
  })

  it("returns target immediately when prefers-reduced-motion is active", async () => {
    vi.spyOn(window, "matchMedia").mockImplementation((query: string) => ({
      matches: query.includes("prefers-reduced-motion"),
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    }))

    await act(async () => {
      root.render(
        <StrictMode>
          <NumberHarness target={120} />
        </StrictMode>
      )
    })

    expect(getValue()).toBe(120)
  })

  it("animates from 0 to target over specified duration", async () => {
    await act(async () => {
      root.render(
        <NumberHarness target={100} options={{ duration: 500 }} />
      )
    })

    // Initially at from (0)
    expect(getValue()).toBe(0)

    // First frame (t = 0ms)
    await triggerRaf(0)
    expect(getValue()).toBe(0)

    // Mid frame (t = 250ms -> 50% elapsed)
    // easeOutCubic(0.5) = 1 - (1 - 0.5)^3 = 0.875
    await triggerRaf(250)
    expect(getValue()).toBeCloseTo(87.5, 1)

    // End frame (t = 500ms -> 100% elapsed)
    await triggerRaf(500)
    expect(getValue()).toBe(100)
  })

  it("animates smoothly to a new target when target updates", async () => {
    await act(async () => {
      root.render(
        <NumberHarness target={100} options={{ duration: 500 }} />
      )
    })

    // Complete first animation
    await triggerRaf(0)
    await triggerRaf(500)
    expect(getValue()).toBe(100)

    // Update target to 200
    await act(async () => {
      root.render(
        <NumberHarness target={200} options={{ duration: 500 }} />
      )
    })

    // Halfway through new animation (t = 250ms)
    // start = 100, delta = 100. 100 + 100 * 0.875 = 187.5
    await triggerRaf(1000)
    await triggerRaf(1250)
    expect(getValue()).toBeCloseTo(187.5, 1)

    // Finish new animation
    await triggerRaf(1500)
    expect(getValue()).toBe(200)
  })

  it("cancels animation frame on unmount", async () => {
    await act(async () => {
      root.render(
        <NumberHarness target={100} options={{ duration: 500 }} />
      )
    })

    expect(rafCallbacks.size).toBeGreaterThan(0)
    await act(async () => {
      root.unmount()
    })
    expect(rafCallbacks.size).toBe(0)
  })
})
