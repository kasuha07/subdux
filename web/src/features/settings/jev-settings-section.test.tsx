// @vitest-environment happy-dom
import { act, StrictMode } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { api } from "@/lib/api"
import type { JevSettings } from "@/types/jev"
import { JevSettingsSection } from "./jev-settings-section"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
}))
vi.mock("@/lib/api", () => ({
  api: { get: vi.fn(), post: vi.fn(), put: vi.fn() },
  getAPIErrorMessage: () => "request failed",
}))
vi.mock("@/lib/toast", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const initial: JevSettings = {
  revision: 1,
  enabled: true,
  api_key_configured: true,
  connection_status: "not_tested",
  last_checked_at: null,
  last_success_at: null,
  classification_requests: 2,
  classification_suggestions: 1,
}

describe("Jev connection diagnostics", () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    vi.resetAllMocks()
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    vi.mocked(api.get).mockResolvedValue(initial)
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  it("tests the saved key and renders refreshed status and counts", async () => {
    const available: JevSettings = {
      ...initial,
      connection_status: "available",
      last_checked_at: "2026-09-22T07:00:00Z",
      last_success_at: "2026-09-22T07:00:00Z",
      classification_requests: 4,
      classification_suggestions: 3,
    }
    vi.mocked(api.post).mockResolvedValue(available)

    await act(async () => root.render(<StrictMode><JevSettingsSection /></StrictMode>))
    const button = [...container.querySelectorAll("button")].find((item) => item.textContent === "settings.jev.testConnection")
    expect(button).toBeDefined()
    await act(async () => button?.click())

    expect(api.post).toHaveBeenCalledWith("/jev/test-connection", {})
    expect(container.textContent).toContain("settings.jev.status.available")
    expect(container.textContent).toContain("4")
    expect(container.textContent).toContain("3")
  })
})
