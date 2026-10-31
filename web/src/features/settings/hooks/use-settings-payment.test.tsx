// @vitest-environment happy-dom
import { act, useEffect } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { getPresetCurrencies, getPresetCurrencyMeta } from "@/lib/currencies"
import { useSettingsPayment } from "./use-settings-payment"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
  }
})

describe("useSettingsPayment add-currency fields", () => {
  let container: HTMLDivElement
  let root: Root
  let state: ReturnType<typeof useSettingsPayment>

  function Harness() {
    const payment = useSettingsPayment({ active: false })
    useEffect(() => { state = payment }, [payment])
    return null
  }

  beforeEach(async () => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
    await act(async () => root.render(<Harness />))
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  it("defaults to the first addable preset with its symbol and alias", () => {
    const [first] = getPresetCurrencies("en")
    expect(state.addCode).toBe(first.code)
    expect(state.addSymbol).toBe(first.symbol)
    expect(state.addAlias).toBe(first.alias)
  })

  it("keeps edits for the selected code and resets them when the code changes", async () => {
    const [, second] = getPresetCurrencies("en")
    await act(async () => state.setAddSymbol("custom-symbol"))
    expect(state.addSymbol).toBe("custom-symbol")

    await act(async () => state.setAddCode(second.code))
    const preset = getPresetCurrencyMeta(second.code, "en")
    expect(state.addCode).toBe(second.code)
    expect(state.addSymbol).toBe(preset?.symbol)
    expect(state.addAlias).toBe(preset?.alias)
  })

  it("clears the symbol and alias for custom codes", async () => {
    await act(async () => state.setAddCode(state.customCodeOption))
    expect(state.addCode).toBe(state.customCodeOption)
    expect(state.addSymbol).toBe("")
    expect(state.addAlias).toBe("")

    await act(async () => state.setAddAlias("Custom alias"))
    expect(state.addAlias).toBe("Custom alias")
  })

  it("falls back to an addable preset for unknown codes", async () => {
    await act(async () => state.setAddCode("NOT-A-PRESET"))
    expect(state.addCode).toBe(getPresetCurrencies("en")[0].code)
  })
})
