// @vitest-environment happy-dom
import { act, StrictMode, useCallback, useState } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { api } from "@/lib/api"
import type { Category } from "@/types"
import type { JevCategorySuggestion } from "@/types/jev"
import { useJevCategory } from "./use-jev-category"

vi.mock("@/lib/api", () => ({ api: { get: vi.fn(), post: vi.fn() } }))
const categories: Category[] = [
  { id: 1, revision: 1, name: "Music", system_key: null, name_customized: true, display_order: 0 },
  { id: 2, revision: 1, name: "Work", system_key: null, name_customized: true, display_order: 1 },
]
function Harness({ open = true, isEditing = false, name = "Spotify", url = "", choices = categories }) {
  const [category, setCategory] = useState("")
  const onSuggestion = useCallback((id: string) => setCategory(id), [])
  const stop = useJevCategory({ open, isEditing, name, url, categories: choices, onSuggestion })
  return <>
    <output>{category}</output>
    <button id="manual" onClick={() => { stop(); setCategory("2") }}>Choose work</button>
    <button id="clear" onClick={() => { stop(); setCategory("") }}>Clear</button>
    <button id="submit" onClick={stop}>Submit</button>
  </>
}
function deferredSuggestion() {
  let resolve!: (value: JevCategorySuggestion) => void
  const promise = new Promise<JevCategorySuggestion>((done) => { resolve = done })
  return { promise, resolve }
}
describe("Jev automatic category selection", () => {
  let container: HTMLDivElement
  let root: Root
  beforeEach(() => {
    vi.useFakeTimers()
    vi.resetAllMocks()
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    vi.mocked(api.get).mockResolvedValue({ enabled: true, api_key_configured: true, revision: 1 })
    vi.mocked(api.post).mockResolvedValue({ category_id: 1 })
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })
  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
    vi.useRealTimers()
  })
  const render = async (props: Parameters<typeof Harness>[0] = {}) => {
    await act(async () => root.render(<StrictMode><Harness {...props} /></StrictMode>))
  }
  const advance = async () => { await act(async () => { await vi.advanceTimersByTimeAsync(600) }) }
  const category = () => container.querySelector("output")?.textContent
  const click = async (selector: string) => { await act(async () => { container.querySelector<HTMLButtonElement>(selector)?.click() }) }

  it("debounces typing and fills a valid suggestion without confirmation", async () => {
    await render({ name: "Spo" })
    await render({ name: "Spotify" })
    expect(api.post).not.toHaveBeenCalled()
    await advance()
    expect(api.post).toHaveBeenCalledTimes(1)
    expect(category()).toBe("1")
    await render({ choices: categories.map((value) => ({ ...value })) })
    await advance()
    expect(api.post).toHaveBeenCalledTimes(1)
  })
  it.each([
    { enabled: false, api_key_configured: true },
    { enabled: true, api_key_configured: false },
  ])("does not classify without both opt-in and a key: %o", async (settings) => {
    vi.mocked(api.get).mockResolvedValue(settings)
    await render()
    await advance()
    expect(api.post).not.toHaveBeenCalled()
  })
  it("leaves existing subscriptions and empty taxonomies alone", async () => {
    await render({ isEditing: true })
    await advance()
    expect(api.get).not.toHaveBeenCalled()
    await render({ choices: [] })
    await advance()
    expect(api.post).not.toHaveBeenCalled()
  })
  it.each(["#manual", "#clear", "#submit"])("ignores an in-flight response after %s", async (button) => {
    const pending = deferredSuggestion()
    vi.mocked(api.post).mockReturnValue(pending.promise)
    await render()
    await advance()
    await click(button)
    await act(async () => pending.resolve({ category_id: 1 }))
    expect(category()).toBe(button === "#manual" ? "2" : "")
    await render({ name: "Netflix" })
    await advance()
    expect(api.post).toHaveBeenCalledTimes(1)
  })
  it("does not send a queued request after a manual choice", async () => {
    await render()
    await click("#manual")
    await advance()
    expect(api.post).not.toHaveBeenCalled()
    expect(category()).toBe("2")
  })
  it("ignores stale input results even if the transport ignores abort", async () => {
    const old = deferredSuggestion()
    vi.mocked(api.post).mockReturnValueOnce(old.promise).mockResolvedValueOnce({ category_id: 2 })
    await render()
    await advance()
    await render({ name: "GitHub Copilot" })
    await advance()
    await act(async () => old.resolve({ category_id: 1 }))
    expect(category()).toBe("2")
  })
  it("invalidates automatic choices when the input changes", async () => {
    await render()
    await advance()
    expect(category()).toBe("1")
    await render({ name: "" })
    expect(category()).toBe("")
  })
  it("ignores closed-form results and resets manual protection on reopen", async () => {
    const old = deferredSuggestion()
    vi.mocked(api.post).mockReturnValueOnce(old.promise)
    await render()
    await advance()
    await render({ open: false })
    await act(async () => old.resolve({ category_id: 1 }))
    expect(category()).toBe("")
    await render()
    await click("#manual")
    await render({ open: false })
    await render()
    await advance()
    expect(category()).toBe("1")
  })
  it.each([null, 999])("ignores unavailable or deleted categories: %s", async (id) => {
    vi.mocked(api.post).mockResolvedValue({ category_id: id })
    await render()
    await advance()
    expect(category()).toBe("")
  })
  it("silently tolerates provider errors", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("unavailable"))
    await render()
    await advance()
    expect(category()).toBe("")
  })
})
