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
import { Switch } from "@/components/ui/switch"

interface SubscriptionRecurrenceFieldsProps {
  billingType: string
  intervalCount: string
  intervalUnit: string
  monthlyDay: string
  onIntervalCountChange: (value: string) => void
  onIntervalUnitChange: (value: string) => void
  onMonthlyDayChange: (value: string) => void
  onRecurrenceTypeChange: (value: string) => void
  onTrialEnabledChange: (value: boolean) => void
  onTrialEndDateChange: (value: string) => void
  onTrialStartDateChange: (value: string) => void
  onYearlyDayChange: (value: string) => void
  onYearlyMonthChange: (value: string) => void
  recurrenceType: string
  trialEnabled: boolean
  trialEndDate: string
  trialStartDate: string
  yearlyDay: string
  yearlyMonth: string
}

export default function SubscriptionRecurrenceFields({
  billingType,
  intervalCount,
  intervalUnit,
  monthlyDay,
  onIntervalCountChange,
  onIntervalUnitChange,
  onMonthlyDayChange,
  onRecurrenceTypeChange,
  onTrialEnabledChange,
  onTrialEndDateChange,
  onTrialStartDateChange,
  onYearlyDayChange,
  onYearlyMonthChange,
  recurrenceType,
  trialEnabled,
  trialEndDate,
  trialStartDate,
  yearlyDay,
  yearlyMonth,
}: SubscriptionRecurrenceFieldsProps) {
  const { t } = useTranslation()

  if (billingType !== "recurring") {
    return null
  }

  return (
    <>
      <div className="space-y-2">
        <div className="grid grid-cols-[11rem_minmax(0,1fr)] items-start gap-2">
          <div className="space-y-1">
            <Label className="flex h-4 items-center text-xs" htmlFor="recurrence-type">
              {t("subscription.form.recurrenceTypeLabel")}
            </Label>
            <Select value={recurrenceType} onValueChange={onRecurrenceTypeChange}>
              <SelectTrigger id="recurrence-type">
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
            <div className="flex min-w-0 items-start gap-2">
              <div className="w-24 shrink-0 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="interval-count">
                  {t("subscription.form.intervalCountLabel")}
                </Label>
                <Input
                  id="interval-count"
                  className="w-24 shrink-0"
                  type="number"
                  min="1"
                  step="1"
                  value={intervalCount}
                  onChange={(event) => onIntervalCountChange(event.target.value)}
                  required
                />
              </div>
              <div className="min-w-[132px] flex-1 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="interval-unit">
                  {t("subscription.form.intervalUnitLabel")}
                </Label>
                <Select value={intervalUnit} onValueChange={onIntervalUnitChange}>
                  <SelectTrigger id="interval-unit">
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
            </div>
          )}

          {recurrenceType === "monthly_date" && (
            <div className="w-32 space-y-1">
              <Label className="flex h-4 items-center text-xs" htmlFor="monthly-day">
                {t("subscription.form.monthlyDayLabel")}
              </Label>
              <Input
                id="monthly-day"
                className="w-32"
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
            <div className="flex items-start gap-2">
              <div className="w-24 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="yearly-month">
                  {t("subscription.form.yearlyMonthLabel")}
                </Label>
                <Input
                  id="yearly-month"
                  className="w-24"
                  type="number"
                  min="1"
                  max="12"
                  step="1"
                  value={yearlyMonth}
                  onChange={(event) => onYearlyMonthChange(event.target.value)}
                  required
                />
              </div>
              <div className="w-24 space-y-1">
                <Label className="flex h-4 items-center text-xs" htmlFor="yearly-day">
                  {t("subscription.form.yearlyDayLabel")}
                </Label>
                <Input
                  id="yearly-day"
                  className="w-24"
                  type="number"
                  min="1"
                  max="31"
                  step="1"
                  value={yearlyDay}
                  onChange={(event) => onYearlyDayChange(event.target.value)}
                  required
                />
              </div>
            </div>
          )}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="trial-enabled">{t("subscription.form.trialLabel")}</Label>
        <div className="inline-flex h-9 w-fit items-center rounded-md border px-3">
          <Switch id="trial-enabled" checked={trialEnabled} onCheckedChange={onTrialEnabledChange} />
          <span className="ml-2 text-sm text-muted-foreground">
            {trialEnabled ? t("subscription.form.enabled") : t("subscription.form.disabled")}
          </span>
        </div>
      </div>

      {trialEnabled && (
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-2">
            <Label htmlFor="trial-start">{t("subscription.form.trialStartLabel")}</Label>
            <Input
              id="trial-start"
              type="date"
              value={trialStartDate}
              onChange={(event) => onTrialStartDateChange(event.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="trial-end">{t("subscription.form.trialEndLabel")}</Label>
            <Input
              id="trial-end"
              type="date"
              value={trialEndDate}
              onChange={(event) => onTrialEndDateChange(event.target.value)}
              required
            />
          </div>
        </div>
      )}
    </>
  )
}
