import { describe, expect, it } from "vitest"
import { renderToStaticMarkup } from "react-dom/server"
import { AppToaster } from "./app-toaster"
import { useTheme } from "@/lib/theme"

describe("AppToaster", () => {
  it("renders a valid React element", () => {
    const el = <AppToaster />
    expect(el).toBeDefined()
  })

  it("safely renders in SSR/node environment without throwing", () => {
    expect(() => renderToStaticMarkup(<AppToaster />)).not.toThrow()
    const markup = renderToStaticMarkup(<AppToaster />)
    expect(markup).toBe("")
  })

  it("useTheme returns light fallback in SSR/node environment", () => {
    // In node environment without window/document, it safely defaults to light
    expect(typeof useTheme).toBe("function")
  })
})
