import * as React from "react"
import { useState, useMemo } from "react"
import { useTranslation } from "react-i18next"
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react"

import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

export interface CalendarPreset {
  label: string
  date: string
}

export interface CalendarProps {
  value?: string // YYYY-MM-DD
  onChange?: (date: string) => void
  minDate?: string // YYYY-MM-DD
  maxDate?: string // YYYY-MM-DD
  className?: string
  showQuickPresets?: boolean
  presets?: CalendarPreset[]
  firstDayOfWeek?: number // 0 = Sunday, 1 = Monday
}

interface ParsedDate {
  year: number
  month: number // 0-11
  day: number
}

function parseDateKey(val?: string | null): ParsedDate | null {
  if (!val) return null
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(val.trim())
  if (!match) return null
  const year = parseInt(match[1], 10)
  const month = parseInt(match[2], 10) - 1
  const day = parseInt(match[3], 10)
  if (Number.isNaN(year) || Number.isNaN(month) || Number.isNaN(day)) return null
  return { year, month, day }
}

function formatDateString(year: number, month: number, day: number): string {
  const y = String(year).padStart(4, "0")
  const m = String(month + 1).padStart(2, "0")
  const d = String(day).padStart(2, "0")
  return `${y}-${m}-${d}`
}

function getDaysInMonth(year: number, month: number): number {
  return new Date(year, month + 1, 0).getDate()
}

export function shiftDate(
  base: ParsedDate,
  shift: { years?: number; months?: number; days?: number }
): string {
  let targetYear = base.year + (shift.years || 0)
  let targetMonth = base.month + (shift.months || 0)

  // Normalize month overflow/underflow
  while (targetMonth < 0) {
    targetMonth += 12
    targetYear -= 1
  }
  while (targetMonth > 11) {
    targetMonth -= 12
    targetYear += 1
  }

  const maxDay = getDaysInMonth(targetYear, targetMonth)
  const targetDay = Math.min(base.day, maxDay)

  if (shift.days) {
    const d = new Date(targetYear, targetMonth, targetDay + shift.days)
    return formatDateString(d.getFullYear(), d.getMonth(), d.getDate())
  }

  return formatDateString(targetYear, targetMonth, targetDay)
}

export function Calendar({
  value,
  onChange,
  minDate,
  maxDate,
  className,
  showQuickPresets = true,
  presets,
  firstDayOfWeek,
}: CalendarProps) {
  const { t, i18n } = useTranslation()

  // Default first day of week: Monday (1) for Chinese and standard locales, Sunday (0) for Japanese / en-US if preferred
  // Monday is 1, Sunday is 0
  const effectiveFirstDay =
    firstDayOfWeek ?? (i18n.language.startsWith("zh") ? 1 : 0)

  const today = useMemo(() => {
    const now = new Date()
    return {
      year: now.getFullYear(),
      month: now.getMonth(),
      day: now.getDate(),
      key: formatDateString(now.getFullYear(), now.getMonth(), now.getDate()),
    }
  }, [])

  const parsedValue = useMemo(() => parseDateKey(value), [value])

  const [viewYear, setViewYear] = useState(() => parsedValue?.year ?? today.year)
  const [viewMonth, setViewMonth] = useState(() => parsedValue?.month ?? today.month)

  const parsedYear = parsedValue?.year
  const parsedMonth = parsedValue?.month

  // Sync view when controlled value changes to a different month/year
  React.useEffect(() => {
    if (parsedYear !== undefined && parsedMonth !== undefined) {
      setViewYear(parsedYear)
      setViewMonth(parsedMonth)
    }
  }, [parsedYear, parsedMonth])

  // Year options for fast select
  const yearOptions = useMemo(() => {
    const startYear = Math.min(viewYear - 5, today.year - 10)
    const endYear = Math.max(viewYear + 10, today.year + 15)
    const years: number[] = []
    for (let y = startYear; y <= endYear; y++) {
      years.push(y)
    }
    return years
  }, [viewYear, today.year])

  // Localized month names from i18n
  const monthNames = useMemo(() => {
    return Array.from({ length: 12 }, (_, i) => t(`calendar.months.${i + 1}`))
  }, [t])

  // Weekday column headers based on first day of week
  const weekdayHeaders = useMemo(() => {
    const days = [
      { key: "sun", label: t("calendar.weekdays.sun") },
      { key: "mon", label: t("calendar.weekdays.mon") },
      { key: "tue", label: t("calendar.weekdays.tue") },
      { key: "wed", label: t("calendar.weekdays.wed") },
      { key: "thu", label: t("calendar.weekdays.thu") },
      { key: "fri", label: t("calendar.weekdays.fri") },
      { key: "sat", label: t("calendar.weekdays.sat") },
    ]
    if (effectiveFirstDay === 1) {
      return [...days.slice(1), days[0]]
    }
    return days
  }, [effectiveFirstDay, t])

  // Quick preset shortcuts
  const defaultPresets = useMemo<CalendarPreset[]>(() => {
    const base = parsedValue ?? today
    return [
      { label: t("calendar.presets.today"), date: today.key },
      { label: t("calendar.presets.tomorrow"), date: shiftDate(today, { days: 1 }) },
      { label: t("calendar.presets.nextWeek"), date: shiftDate(base, { days: 7 }) },
      { label: t("calendar.presets.nextMonth"), date: shiftDate(base, { months: 1 }) },
      { label: t("calendar.presets.nextYear"), date: shiftDate(base, { years: 1 }) },
    ]
  }, [parsedValue, today, t])

  const activePresets = presets || defaultPresets

  function handlePrevMonth() {
    if (viewMonth === 0) {
      setViewYear((y) => y - 1)
      setViewMonth(11)
    } else {
      setViewMonth((m) => m - 1)
    }
  }

  function handleNextMonth() {
    if (viewMonth === 11) {
      setViewYear((y) => y + 1)
      setViewMonth(0)
    } else {
      setViewMonth((m) => m + 1)
    }
  }

  function handlePrevYear() {
    setViewYear((y) => y - 1)
  }

  function handleNextYear() {
    setViewYear((y) => y + 1)
  }

  function handleSelectDate(dateKey: string, cellMonth: number, cellYear: number) {
    if (cellMonth !== viewMonth || cellYear !== viewYear) {
      setViewYear(cellYear)
      setViewMonth(cellMonth)
    }
    onChange?.(dateKey)
  }

  // 42 cells grid (6 rows x 7 cols)
  const cells = useMemo(() => {
    const daysInCurrent = getDaysInMonth(viewYear, viewMonth)
    const daysInPrev = getDaysInMonth(
      viewMonth === 0 ? viewYear - 1 : viewYear,
      viewMonth === 0 ? 11 : viewMonth - 1
    )

    const firstDayOfWeekIndex = new Date(viewYear, viewMonth, 1).getDay()
    const leadingDaysCount = (firstDayOfWeekIndex - effectiveFirstDay + 7) % 7

    const result = []

    // Previous month days
    const prevYear = viewMonth === 0 ? viewYear - 1 : viewYear
    const prevMonth = viewMonth === 0 ? 11 : viewMonth - 1
    for (let i = leadingDaysCount - 1; i >= 0; i--) {
      const dayNum = daysInPrev - i
      const dateKey = formatDateString(prevYear, prevMonth, dayNum)
      const isDisabled = Boolean(
        (minDate && dateKey < minDate) || (maxDate && dateKey > maxDate)
      )
      result.push({
        key: `prev-${dayNum}`,
        dateKey,
        dayNum,
        year: prevYear,
        month: prevMonth,
        isCurrentMonth: false,
        isToday: dateKey === today.key,
        isSelected: dateKey === value,
        isDisabled,
      })
    }

    // Current month days
    for (let dayNum = 1; dayNum <= daysInCurrent; dayNum++) {
      const dateKey = formatDateString(viewYear, viewMonth, dayNum)
      const isDisabled = Boolean(
        (minDate && dateKey < minDate) || (maxDate && dateKey > maxDate)
      )
      result.push({
        key: `curr-${dayNum}`,
        dateKey,
        dayNum,
        year: viewYear,
        month: viewMonth,
        isCurrentMonth: true,
        isToday: dateKey === today.key,
        isSelected: dateKey === value,
        isDisabled,
      })
    }

    // Next month days to make 42
    const nextYear = viewMonth === 11 ? viewYear + 1 : viewYear
    const nextMonth = viewMonth === 11 ? 0 : viewMonth + 1
    const trailingDaysCount = 42 - result.length
    for (let dayNum = 1; dayNum <= trailingDaysCount; dayNum++) {
      const dateKey = formatDateString(nextYear, nextMonth, dayNum)
      const isDisabled = Boolean(
        (minDate && dateKey < minDate) || (maxDate && dateKey > maxDate)
      )
      result.push({
        key: `next-${dayNum}`,
        dateKey,
        dayNum,
        year: nextYear,
        month: nextMonth,
        isCurrentMonth: false,
        isToday: dateKey === today.key,
        isSelected: dateKey === value,
        isDisabled,
      })
    }

    return result
  }, [viewYear, viewMonth, effectiveFirstDay, minDate, maxDate, today.key, value])

  return (
    <div className={cn("w-[280px] select-none space-y-2.5", className)}>
      {/* Quick Presets */}
      {showQuickPresets && (
        <div className="flex items-center gap-1 overflow-x-auto pb-1.5 scrollbar-none border-b border-border/60">
          {activePresets.map((preset) => {
            const isPresetActive = value === preset.date
            return (
              <Button
                key={preset.label}
                type="button"
                variant={isPresetActive ? "default" : "outline"}
                size="xs"
                className={cn(
                  "h-6 px-2 text-[11px] font-normal shrink-0 rounded-md transition-all",
                  isPresetActive && "font-medium"
                )}
                onClick={() => {
                  const parsed = parseDateKey(preset.date)
                  if (parsed) {
                    setViewYear(parsed.year)
                    setViewMonth(parsed.month)
                  }
                  onChange?.(preset.date)
                }}
              >
                {preset.label}
              </Button>
            )
          })}
        </div>
      )}

      {/* Header: Month & Year Navigator */}
      <div className="flex items-center justify-between gap-1">
        <div className="flex items-center gap-0.5">
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={handlePrevYear}
            aria-label={t("calendar.prevYear")}
            title={t("calendar.prevYear")}
            className="size-7 text-muted-foreground hover:text-foreground"
          >
            <ChevronsLeft className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={handlePrevMonth}
            aria-label={t("calendar.prevMonth")}
            title={t("calendar.prevMonth")}
            className="size-7 text-muted-foreground hover:text-foreground"
          >
            <ChevronLeft className="size-3.5" />
          </Button>
        </div>

        {/* Month & Year Selectors */}
        <div className="flex items-center gap-1">
          <select
            value={viewYear}
            onChange={(e) => setViewYear(Number(e.target.value))}
            aria-label={t("calendar.year", "Year")}
            className="h-7 rounded-md bg-transparent px-1.5 py-0 text-xs font-semibold text-foreground hover:bg-muted/80 cursor-pointer border-none outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            {yearOptions.map((year) => (
              <option key={year} value={year} className="bg-popover text-popover-foreground">
                {year}
              </option>
            ))}
          </select>

          <select
            value={viewMonth}
            onChange={(e) => setViewMonth(Number(e.target.value))}
            aria-label={t("calendar.month", "Month")}
            className="h-7 rounded-md bg-transparent px-1.5 py-0 text-xs font-semibold text-foreground hover:bg-muted/80 cursor-pointer border-none outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            {monthNames.map((name, idx) => (
              <option key={idx} value={idx} className="bg-popover text-popover-foreground">
                {name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex items-center gap-0.5">
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={handleNextMonth}
            aria-label={t("calendar.nextMonth")}
            title={t("calendar.nextMonth")}
            className="size-7 text-muted-foreground hover:text-foreground"
          >
            <ChevronRight className="size-3.5" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            onClick={handleNextYear}
            aria-label={t("calendar.nextYear")}
            title={t("calendar.nextYear")}
            className="size-7 text-muted-foreground hover:text-foreground"
          >
            <ChevronsRight className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* Weekday headers */}
      <div className="grid grid-cols-7 gap-1 text-center" role="row">
        {weekdayHeaders.map((w) => (
          <div
            key={w.key}
            className="text-[11px] font-medium text-muted-foreground/80 py-1"
            role="columnheader"
          >
            {w.label}
          </div>
        ))}
      </div>

      {/* Calendar Grid */}
      <div className="grid grid-cols-7 gap-1" role="grid">
        {cells.map((cell) => (
          <button
            key={cell.key}
            type="button"
            disabled={cell.isDisabled}
            aria-selected={cell.isSelected}
            aria-current={cell.isToday ? "date" : undefined}
            onClick={() => !cell.isDisabled && handleSelectDate(cell.dateKey, cell.month, cell.year)}
            className={cn(
              "relative size-8 p-0 text-xs rounded-md flex items-center justify-center transition-colors cursor-pointer outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1",
              cell.isSelected &&
                "bg-primary text-primary-foreground font-semibold hover:bg-primary hover:text-primary-foreground shadow-xs",
              !cell.isSelected &&
                cell.isToday &&
                "border border-primary font-semibold text-primary",
              !cell.isSelected &&
                !cell.isToday &&
                cell.isCurrentMonth &&
                "text-foreground hover:bg-accent hover:text-accent-foreground",
              !cell.isSelected &&
                !cell.isCurrentMonth &&
                "text-muted-foreground/40 hover:bg-muted/40 hover:text-muted-foreground",
              cell.isDisabled && "opacity-30 cursor-not-allowed pointer-events-none"
            )}
          >
            {cell.dayNum}
          </button>
        ))}
      </div>
    </div>
  )
}
