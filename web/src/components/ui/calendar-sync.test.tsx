// @vitest-environment happy-dom
import { act } from "react"
import { createRoot, type Root } from "react-dom/client"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { Calendar } from "./calendar"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({ t: (key: string) => key, i18n: { language: "en" } }),
  }
})

describe("Calendar controlled value sync", () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
    container = document.createElement("div")
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  function viewSelects() {
    const year = container.querySelector<HTMLSelectElement>('select[aria-label="calendar.year"]')
    const month = container.querySelector<HTMLSelectElement>('select[aria-label="calendar.month"]')
    return { year: year?.value, month: month?.value }
  }

  it("moves the visible month when the controlled value changes", async () => {
    await act(async () => root.render(<Calendar value="2026-03-15" onChange={vi.fn()} />))
    expect(viewSelects()).toEqual({ year: "2026", month: "2" })

    await act(async () => root.render(<Calendar value="2027-08-01" onChange={vi.fn()} />))
    expect(viewSelects()).toEqual({ year: "2027", month: "7" })
  })

  it("keeps manual navigation when the value stays in the same month", async () => {
    await act(async () => root.render(<Calendar value="2026-03-15" onChange={vi.fn()} />))
    const next = container.querySelector<HTMLButtonElement>('button[aria-label="calendar.nextMonth"]')
    await act(async () => next?.click())
    expect(viewSelects()).toEqual({ year: "2026", month: "3" })

    await act(async () => root.render(<Calendar value="2026-03-20" onChange={vi.fn()} />))
    expect(viewSelects()).toEqual({ year: "2026", month: "3" })
  })
})
