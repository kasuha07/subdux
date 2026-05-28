import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

interface SubscriptionRecurrenceFieldsProps {
  intervalCount: string
  intervalUnit: string
  monthlyDay: string
  onIntervalCountChange: (value: string) => void
  onIntervalUnitChange: (value: string) => void
  onMonthlyDayChange: (value: string) => void
  onRecurrenceTypeChange: (value: string) => void
  onYearlyDayChange: (value: string) => void
  onYearlyMonthChange: (value: string) => void
  recurrenceType: string
  yearlyDay: string
  yearlyMonth: string
}

export default function SubscriptionRecurrenceFields({
  intervalCount,
  intervalUnit,
  monthlyDay,
  onIntervalCountChange,
  onIntervalUnitChange,
  onMonthlyDayChange,
  onRecurrenceTypeChange,
  onYearlyDayChange,
  onYearlyMonthChange,
  recurrenceType,
  yearlyDay,
  yearlyMonth,
}: SubscriptionRecurrenceFieldsProps) {
  const { t } = useTranslation()

  return (
    <>
      <div className="space-y-2">
        <div className="grid grid-cols-[minmax(0,1.15fr)_minmax(4.75rem,0.55fr)_minmax(5.75rem,0.8fr)] items-start gap-2 sm:grid-cols-[11rem_6rem_minmax(0,1fr)] sm:gap-3">
          <div className="min-w-0 space-y-1">
            <Label className="flex h-4 items-center text-xs" htmlFor="recurrence-type">
              {t("subscription.form.recurrenceTypeLabel")}
            </Label>
            <Select value={recurrenceType} onValueChange={onRecurrenceTypeChange}>
              <SelectTrigger id="recurrence-type" className="w-full min-w-0">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="interval">{t("subscription.form.recurrenceType.interval")}</SelectItem>
                <SelectItem value="monthly_date">
                  {t("subscription.form.recurrenceType.monthly_date")}
                </SelectItem>
                <SelectItem value="yearly_date">
                  {t("subscription.form.recurrenceType.yearly_date")}
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          {recurrenceType === "interval" && (
            <>
              <div className="min-w-0 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="interval-count">
                  {t("subscription.form.intervalCountLabel")}
                </Label>
                <Input
                  id="interval-count"
                  type="number"
                  min="1"
                  step="1"
                  value={intervalCount}
                  onChange={(event) => onIntervalCountChange(event.target.value)}
                  required
                />
              </div>
              <div className="min-w-0 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="interval-unit">
                  {t("subscription.form.intervalUnitLabel")}
                </Label>
                <Select value={intervalUnit} onValueChange={onIntervalUnitChange}>
                  <SelectTrigger id="interval-unit" className="w-full min-w-0">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="day">{t("subscription.form.intervalUnit.day")}</SelectItem>
                    <SelectItem value="week">{t("subscription.form.intervalUnit.week")}</SelectItem>
                    <SelectItem value="month">{t("subscription.form.intervalUnit.month")}</SelectItem>
                    <SelectItem value="year">{t("subscription.form.intervalUnit.year")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </>
          )}

          {recurrenceType === "monthly_date" && (
            <div className="col-span-2 w-full space-y-1 sm:w-32">
              <Label className="flex h-4 items-center text-xs" htmlFor="monthly-day">
                {t("subscription.form.monthlyDayLabel")}
              </Label>
              <Input
                id="monthly-day"
                className="w-full sm:w-32"
                type="number"
                min="1"
                max="31"
                step="1"
                value={monthlyDay}
                onChange={(event) => onMonthlyDayChange(event.target.value)}
                required
              />
            </div>
          )}

          {recurrenceType === "yearly_date" && (
            <>
              <div className="min-w-0 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="yearly-month">
                  {t("subscription.form.yearlyMonthLabel")}
                </Label>
                <Input
                  id="yearly-month"
                  type="number"
                  min="1"
                  max="12"
                  step="1"
                  value={yearlyMonth}
                  onChange={(event) => onYearlyMonthChange(event.target.value)}
                  required
                />
              </div>
              <div className="min-w-0 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="yearly-day">
                  {t("subscription.form.yearlyDayLabel")}
                </Label>
                <Input
                  id="yearly-day"
                  type="number"
                  min="1"
                  max="31"
                  step="1"
                  value={yearlyDay}
                  onChange={(event) => onYearlyDayChange(event.target.value)}
                  required
                />
              </div>
            </>
          )}
        </div>
      </div>
    </>
  )
}
