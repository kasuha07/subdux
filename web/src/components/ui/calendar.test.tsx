import { describe, expect, it, vi } from "vitest"
import { renderToStaticMarkup } from "react-dom/server"

import { Calendar } from "./calendar"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string) => {
        if (key === "calendar.presets.today") return "Today"
        if (key === "calendar.presets.tomorrow") return "Tomorrow"
        if (key === "calendar.presets.nextWeek") return "+1 Week"
        if (key === "calendar.presets.nextMonth") return "+1 Month"
        if (key === "calendar.presets.nextYear") return "+1 Year"
        if (key === "calendar.prevMonth") return "Previous month"
        if (key === "calendar.nextMonth") return "Next month"
        if (key === "calendar.prevYear") return "Previous year"
        if (key === "calendar.nextYear") return "Next year"
        if (key.startsWith("calendar.months.")) return `Month ${key.split(".").pop()}`
        if (key.startsWith("calendar.weekdays.")) return key.split(".").pop()?.toUpperCase()
        return key
      },
      i18n: { language: "en" },
    }),
  }
})

describe("Calendar component", () => {
  it("renders calendar grid and navigation correctly", () => {
    const markup = renderToStaticMarkup(
      <Calendar value="2026-09-20" onChange={vi.fn()} />
    )

    // Check year options and month options
    expect(markup).toContain("2026")
    expect(markup).toContain("Month 9")
    // Check preset buttons
    expect(markup).toContain("Today")
    expect(markup).toContain("Tomorrow")
    expect(markup).toContain("+1 Month")
    // Check selected date is highlighted
    expect(markup).toContain("bg-primary text-primary-foreground")
  })

  it("supports hiding quick presets", () => {
    const markup = renderToStaticMarkup(
      <Calendar value="2026-09-20" showQuickPresets={false} onChange={vi.fn()} />
    )

    expect(markup).not.toContain("+1 Month")
    expect(markup).toContain("Month 9")
  })
})

describe("shiftDate", () => {
  it("rolls over month correctly when shifting days (+1 week)", async () => {
    const { shiftDate } = await import("./calendar")

    // September 25 + 7 days should be October 2
    const result = shiftDate({ year: 2026, month: 8, day: 25 }, { days: 7 })
    expect(result).toBe("2026-10-02")
  })

  it("rolls over year correctly when shifting days (+1 week)", async () => {
    const { shiftDate } = await import("./calendar")

    // December 28 + 7 days should be January 4 next year
    const result = shiftDate({ year: 2026, month: 11, day: 28 }, { days: 7 })
    expect(result).toBe("2027-01-04")
  })

  it("rolls over month correctly for tomorrow at month end", async () => {
    const { shiftDate } = await import("./calendar")

    // September 30 + 1 day should be October 1
    const result = shiftDate({ year: 2026, month: 8, day: 30 }, { days: 1 })
    expect(result).toBe("2026-10-01")
  })

  it("rolls over year correctly for +1 month at year end", async () => {
    const { shiftDate } = await import("./calendar")

    const result = shiftDate({ year: 2026, month: 11, day: 15 }, { months: 1 })
    expect(result).toBe("2027-01-15")
  })

  it("clamps days correctly for +1 month when next month has fewer days", async () => {
    const { shiftDate } = await import("./calendar")

    // January 31 + 1 month should be February 28
    const result = shiftDate({ year: 2026, month: 0, day: 31 }, { months: 1 })
    expect(result).toBe("2026-02-28")
  })
})
