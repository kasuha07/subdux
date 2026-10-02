// @vitest-environment happy-dom
import { act } from "react"
import { createRoot, type Root } from "react-dom/client"
import { MemoryRouter } from "react-router"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { api } from "@/lib/api"
import MCPConsentPage from "./mcp-consent-page"

const interaction = vi.hoisted(() => ({ request: "a".repeat(43), t: (key: string) => key }))
vi.mock("react-router", async (original) => ({
  ...await original<typeof import("react-router")>(),
  useSearchParams: () => [new URLSearchParams({ request: interaction.request })],
}))
vi.mock("react-i18next", () => ({ useTranslation: () => ({ t: interaction.t }) }))
vi.mock("@/lib/api", () => ({
  api: { get: vi.fn(), post: vi.fn() },
  getUser: () => ({ username: "test-user" }),
}))

function client(name: string) {
  return { client_id: name, client_name: name, redirect_uri: "https://client.example/callback", scopes: ["read", "write"] }
}

describe("MCP consent interactions", () => {
  let container: HTMLDivElement
  let root: Root
  beforeEach(() => {
    vi.resetAllMocks()
    interaction.request = "a".repeat(43)
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })
  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })
  const render = async () => {
    await act(async () => root.render(<MemoryRouter><MCPConsentPage /></MemoryRouter>))
  }

  it("clears displayed client and write approval while a new request loads", async () => {
    vi.mocked(api.get).mockResolvedValue(client("First client"))
    await render()
    await act(async () => container.querySelector<HTMLButtonElement>("#mcp-write")?.click())
    expect(container.querySelector("#mcp-write")?.getAttribute("data-state")).toBe("checked")

    let resolve!: (value: ReturnType<typeof client>) => void
    vi.mocked(api.get).mockReturnValue(new Promise((done) => { resolve = done }))
    interaction.request = "b".repeat(43)
    await render()
    expect(container.textContent).not.toContain("First client")
    expect(container.querySelector("#mcp-write")).toBeNull()
    expect(container.querySelector('[role="status"]')).not.toBeNull()
    await act(async () => resolve(client("Second client")))
    expect(container.textContent).toContain("Second client")
    expect(container.querySelector("#mcp-write")?.getAttribute("data-state")).toBe("unchecked")

    vi.mocked(api.post).mockRejectedValue(new Error("fixture stops before redirect"))
    await act(async () => {
      Array.from(container.querySelectorAll("button")).find((button) => button.textContent === "settings.mcpOAuth.authorize")?.click()
    })
    expect(api.post).toHaveBeenCalledWith(`/mcp/oauth/requests/${interaction.request}`, { approve: true, allow_write: false })
  })

  it("ignores a late response from the previous interaction", async () => {
    let resolve!: (value: ReturnType<typeof client>) => void
    vi.mocked(api.get).mockReturnValueOnce(new Promise((done) => { resolve = done }))
    await render()
    interaction.request = "b".repeat(43)
    vi.mocked(api.get).mockResolvedValue(client("Current client"))
    await render()
    await act(async () => resolve(client("Stale client")))
    expect(container.textContent).toContain("Current client")
    expect(container.textContent).not.toContain("Stale client")
  })
})
