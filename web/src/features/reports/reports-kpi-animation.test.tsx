// @vitest-environment happy-dom
import { act, StrictMode } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { TrendingUp } from "lucide-react"

import { KpiCard } from "./reports-panels"

describe("KpiCard animation in browser environment", () => {
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

  const getDisplayedValue = (testId: string) => {
    const el = container.querySelector(`[data-testid="${testId}"]`)
    return el?.textContent?.trim() ?? ""
  }

  it("animates numericValue from 0 to target over duration", async () => {
    await act(async () => {
      root.render(
        <KpiCard
          icon={TrendingUp}
          label="Total Monthly"
          numericValue={100}
          formatValue={(val) => `$${val.toFixed(2)}`}
          detail="Yearly $1,200"
          testId="kpi-monthly"
        />
      )
    })

    // Initial mount is at 0
    expect(getDisplayedValue("kpi-monthly")).toBe("$0.00")

    // Halfway through animation (t = 350ms for default 700ms)
    // easeOutCubic(0.5) = 1 - 0.5^3 = 0.875 -> 87.50
    await triggerRaf(0)
    await triggerRaf(350)
    expect(getDisplayedValue("kpi-monthly")).toBe("$87.50")

    // Complete animation (t = 700ms)
    await triggerRaf(700)
    expect(getDisplayedValue("kpi-monthly")).toBe("$100.00")
  })

  it("immediately returns final value when animate is false", async () => {
    await act(async () => {
      root.render(
        <StrictMode>
          <KpiCard
            icon={TrendingUp}
            label="Total Monthly"
            numericValue={100}
            formatValue={(val) => `$${val.toFixed(2)}`}
            detail="Yearly $1,200"
            animate={false}
            testId="kpi-monthly"
          />
        </StrictMode>
      )
    })

    expect(getDisplayedValue("kpi-monthly")).toBe("$100.00")
  })
})
