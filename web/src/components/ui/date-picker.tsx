import * as React from "react"
import { useState, useId } from "react"
import { useTranslation } from "react-i18next"
import { CalendarDays, X } from "lucide-react"

import { Calendar, type CalendarPreset } from "@/components/ui/calendar"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { cn, formatDate } from "@/lib/utils"

export interface DatePickerProps {
  id?: string
  name?: string
  value?: string // YYYY-MM-DD
  onChange?: (date: string) => void
  placeholder?: string
  disabled?: boolean
  required?: boolean
  minDate?: string
  maxDate?: string
  className?: string
  triggerClassName?: string
  align?: "start" | "center" | "end"
  showPresets?: boolean
  presets?: CalendarPreset[]
  clearable?: boolean
  autoClose?: boolean
  firstDayOfWeek?: number
  "aria-label"?: string
}

export function DatePicker({
  id,
  name,
  value,
  onChange,
  placeholder,
  disabled = false,
  required = false,
  minDate,
  maxDate,
  className,
  triggerClassName,
  align = "start",
  showPresets = true,
  presets,
  clearable,
  autoClose = true,
  firstDayOfWeek,
  "aria-label": ariaLabelProp,
}: DatePickerProps) {
  const { t, i18n } = useTranslation()
  const [open, setOpen] = useState(false)
  const generatedId = useId()
  const effectiveId = id || generatedId

  // Default clearable if not required and clearable not explicitly false
  const isClearable = (clearable ?? !required) && Boolean(value) && !disabled

  const displayDate = value ? formatDate(value, i18n.language) : ""
  const effectivePlaceholder = placeholder || t("calendar.selectDate")

  function handleClear(e: React.MouseEvent | React.KeyboardEvent) {
    e.stopPropagation()
    e.preventDefault()
    onChange?.("")
  }

  return (
    <div className={cn("relative inline-block w-full", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            id={effectiveId}
            disabled={disabled}
            aria-haspopup="dialog"
            aria-expanded={open}
            aria-label={ariaLabelProp || effectivePlaceholder}
            className={cn(
              "group h-9 w-full min-w-0 rounded-md border border-input bg-transparent dark:bg-input/30 px-3 py-1 text-sm shadow-xs transition-[color,box-shadow] outline-none flex items-center justify-between text-left",
              "focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]",
              "disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50",
              !value && "text-muted-foreground",
              triggerClassName
            )}
          >
            <div className="flex items-center gap-2 min-w-0 flex-1 truncate">
              <CalendarDays className="size-4 shrink-0 text-muted-foreground group-hover:text-foreground transition-colors" />
              <span className="truncate">
                {displayDate || effectivePlaceholder}
              </span>
            </div>
            {isClearable && (
              <span
                role="button"
                tabIndex={0}
                aria-label={t("common.clear")}
                onClick={handleClear}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    handleClear(e)
                  }
                }}
                className="-mr-1 rounded-sm p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted/80 transition-colors focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-ring"
              >
                <X className="size-3.5" />
              </span>
            )}
          </button>
        </PopoverTrigger>

        {name && (
          <input
            type="hidden"
            name={name}
            value={value || ""}
            required={required}
            tabIndex={-1}
            aria-hidden="true"
          />
        )}

        <PopoverContent
          className="w-auto p-3 motion-surface"
          align={align}
          sideOffset={4}
        >
          <Calendar
            value={value}
            onChange={(newDate) => {
              onChange?.(newDate)
              if (autoClose) {
                setOpen(false)
              }
            }}
            minDate={minDate}
            maxDate={maxDate}
            showQuickPresets={showPresets}
            presets={presets}
            firstDayOfWeek={firstDayOfWeek}
          />
        </PopoverContent>
      </Popover>
    </div>
  )
}
