import { renderToStaticMarkup } from "react-dom/server"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { isMobileDevice, useIsMobileDevice } from "./device"

describe("isMobileDevice", () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("returns false in non-browser environment", () => {
    vi.stubGlobal("window", undefined)
    vi.stubGlobal("navigator", undefined)
    expect(isMobileDevice()).toBe(false)
  })

  it("identifies desktop browser as non-mobile", () => {
    vi.stubGlobal("window", {
      matchMedia: vi.fn().mockReturnValue({ matches: false }),
    })
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      maxTouchPoints: 0,
    })

    expect(isMobileDevice()).toBe(false)
  })

  it("identifies iPhone user agent as mobile", () => {
    vi.stubGlobal("window", {})
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
      maxTouchPoints: 5,
    })

    expect(isMobileDevice()).toBe(true)
  })

  it("identifies Android user agent as mobile", () => {
    vi.stubGlobal("window", {})
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
      maxTouchPoints: 5,
    })

    expect(isMobileDevice()).toBe(true)
  })

  it("identifies iPadOS reporting as Macintosh with multi-touch as mobile", () => {
    vi.stubGlobal("window", {})
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
      maxTouchPoints: 5,
    })

    expect(isMobileDevice()).toBe(true)
  })

  it("identifies userAgentData.mobile boolean when present", () => {
    vi.stubGlobal("window", {})
    vi.stubGlobal("navigator", {
      userAgent: "custom",
      userAgentData: { mobile: true },
    })

    expect(isMobileDevice()).toBe(true)
  })

  it("falls back to pointer: coarse and hover: none media query", () => {
    vi.stubGlobal("navigator", {
      userAgent: "unknown-browser",
      maxTouchPoints: 1,
    })
    vi.stubGlobal("window", {
      matchMedia: vi.fn().mockImplementation((query: string) => ({
        matches: query === "(pointer: coarse) and (hover: none)",
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      })),
    })

    expect(isMobileDevice()).toBe(true)
  })

  it("renders through useIsMobileDevice hook in component", () => {
    vi.stubGlobal("window", {})
    vi.stubGlobal("navigator", {
      userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
      maxTouchPoints: 5,
    })

    function DummyComponent() {
      const isMobile = useIsMobileDevice()
      return <span>{isMobile ? "is-mobile" : "is-desktop"}</span>
    }

    const markup = renderToStaticMarkup(<DummyComponent />)
    expect(markup).toContain("is-mobile")
  })
})
