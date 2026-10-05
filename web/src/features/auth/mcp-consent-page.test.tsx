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
  isBackendAPIError: () => false,
}))
// Stand-in for the step-up dialog: when opened, it exposes a control that
// completes verification with a fixed ticket.
vi.mock("@/features/admin/reauth-dialog", () => ({
  default: ({ open, operation, onVerified }: { open: boolean; operation: string; onVerified: (ticket: string) => void }) =>
    open ? <button id="reauth-verify" data-operation={operation} onClick={() => void onVerified("test-ticket")}>verify</button> : null,
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
  const clickButton = (label: string) => {
    Array.from(container.querySelectorAll("button")).find((button) => button.textContent === label)?.click()
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
    await act(async () => clickButton("settings.mcpOAuth.authorize"))
    expect(api.post).not.toHaveBeenCalled()
    expect(container.querySelector("#reauth-verify")?.getAttribute("data-operation")).toBe("authorize_mcp_client")
    await act(async () => container.querySelector<HTMLButtonElement>("#reauth-verify")?.click())
    expect(api.post).toHaveBeenCalledWith(
      `/mcp/oauth/requests/${interaction.request}`,
      { approve: true, allow_write: false },
      { headers: { "X-Reauth-Ticket": "test-ticket" } },
    )
  })

  it("denies without step-up", async () => {
    vi.mocked(api.get).mockResolvedValue(client("Client"))
    vi.mocked(api.post).mockRejectedValue(new Error("fixture stops before redirect"))
    await render()
    await act(async () => clickButton("settings.mcpOAuth.deny"))
    expect(container.querySelector("#reauth-verify")).toBeNull()
    expect(api.post).toHaveBeenCalledWith(`/mcp/oauth/requests/${interaction.request}`, { approve: false, allow_write: false }, undefined)
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
