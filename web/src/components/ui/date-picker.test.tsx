import { describe, expect, it, vi } from "vitest"
import { renderToStaticMarkup } from "react-dom/server"

import { DatePicker } from "./date-picker"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => {
        if (key === "calendar.selectDate") return "Select date"
        if (key === "common.clear") return "Clear"
        return key
      },
      i18n: { language: "en" },
    }),
  }
})

describe("DatePicker component", () => {
  it("renders trigger button with formatted date", () => {
    const markup = renderToStaticMarkup(
      <DatePicker id="test-date" value="2026-09-20" onChange={vi.fn()} />
    )

    expect(markup).toContain('id="test-date"')
    expect(markup).toContain("Sep 20, 2026")
  })

  it("renders placeholder when value is empty", () => {
    const markup = renderToStaticMarkup(
      <DatePicker id="test-date-empty" placeholder="Choose a date" onChange={vi.fn()} />
    )

    expect(markup).toContain("Choose a date")
  })

  it("renders clear button when clearable and value present", () => {
    const markup = renderToStaticMarkup(
      <DatePicker value="2026-09-20" clearable={true} onChange={vi.fn()} />
    )

    expect(markup).toContain('aria-label="Clear"')
  })
})
