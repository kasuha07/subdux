import { useTranslation } from "react-i18next"
import { CheckCircle2 } from "lucide-react"

import { DatePicker } from "@/components/ui/date-picker"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type {
  SubscriptionFormValues,
  SubscriptionIntroMode,
} from "@/features/subscriptions/hooks/use-subscription-form-state"
import { computeIntroPlan } from "@/features/subscriptions/hooks/use-subscription-form-state"
import { cn, formatCurrency, formatDate } from "@/lib/utils"

import { addDaysToDateKey } from "./subscription-pending-price"

interface SubscriptionIntroFieldsProps {
  amountStep: number
  isEditing: boolean
  onChange: <K extends keyof SubscriptionFormValues>(field: K, value: SubscriptionFormValues[K]) => void
  values: SubscriptionFormValues
}

const INTRO_MODES: SubscriptionIntroMode[] = ["none", "trial", "intro"]
const MAX_LISTED_INTRO_DATES = 3

export default function SubscriptionIntroFields({
  amountStep,
  isEditing,
  onChange,
  values,
}: SubscriptionIntroFieldsProps) {
  const { t, i18n } = useTranslation()
  const plan = computeIntroPlan(values)
  const regularPrice = parseFloat(values.amount)
  const hasRegularPrice = !Number.isNaN(regularPrice) && regularPrice >= 0
  const formatAmount = (amount: number) => formatCurrency(amount, values.currency, i18n.language)
  const formatDay = (date: string) => formatDate(date, i18n.language)

  let summary: string | null = null
  if (plan && hasRegularPrice) {
    if (values.introMode === "trial") {
      summary = t("subscription.form.intro.trialSummary", {
        end: formatDay(addDaysToDateKey(plan.regularFrom, -1)),
        start: formatDay(plan.regularFrom),
        amount: formatAmount(regularPrice),
      })
    } else {
      const listed = plan.introDates.slice(0, MAX_LISTED_INTRO_DATES).map(formatDay).join(t("subscription.form.intro.dateSeparator"))
      summary = t(
        plan.introDates.length > MAX_LISTED_INTRO_DATES
          ? "subscription.form.intro.introSummaryMore"
          : "subscription.form.intro.introSummary",
        {
          dates: listed,
          count: plan.introDates.length,
          introAmount: formatAmount(parseFloat(values.introAmount)),
          start: formatDay(plan.regularFrom),
          amount: formatAmount(regularPrice),
        }
      )
    }
  }

  return (
    <div className="space-y-3 rounded-lg border border-dashed p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <Label id="intro-mode-label">{t("subscription.form.intro.label")}</Label>
        <div
          role="radiogroup"
          aria-labelledby="intro-mode-label"
          className="inline-flex rounded-md border bg-muted/40 p-0.5"
        >
          {INTRO_MODES.map((mode) => (
            <button
              key={mode}
              type="button"
              role="radio"
              aria-checked={values.introMode === mode}
              onClick={() => onChange("introMode", mode)}
              className={cn(
                "rounded px-3 py-1 text-xs font-medium text-muted-foreground transition-colors",
                "focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-ring",
                values.introMode === mode && "bg-background text-foreground shadow-sm"
              )}
            >
              {t(`subscription.form.intro.mode.${mode}`)}
            </button>
          ))}
        </div>
      </div>

      {values.introMode === "trial" ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="trial-days">
              {isEditing ? t("subscription.form.intro.trialDaysLeftLabel") : t("subscription.form.intro.trialDaysLabel")}
            </Label>
            <Input
              id="trial-days"
              type="number"
              min="1"
              step="1"
              value={values.trialDays}
              onChange={(event) => onChange("trialDays", event.target.value)}
              required
            />
          </div>
          {isEditing ? null : (
            <div className="space-y-2">
              <Label htmlFor="trial-start">{t("subscription.form.intro.trialStartLabel")}</Label>
              <DatePicker
                id="trial-start"
                value={values.trialStart}
                onChange={(date) => onChange("trialStart", date)}
                required
              />
            </div>
          )}
        </div>
      ) : null}

      {values.introMode === "intro" ? (
        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-2">
            <Label htmlFor="intro-cycles">
              {isEditing ? t("subscription.form.intro.introCyclesLeftLabel") : t("subscription.form.intro.introCyclesLabel")}
            </Label>
            <Input
              id="intro-cycles"
              type="number"
              min="1"
              step="1"
              value={values.introCycles}
              onChange={(event) => onChange("introCycles", event.target.value)}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="intro-amount">{t("subscription.form.intro.introAmountLabel")}</Label>
            <Input
              id="intro-amount"
              type="number"
              min="0"
              step={amountStep}
              value={values.introAmount}
              onChange={(event) => onChange("introAmount", event.target.value)}
              required
            />
          </div>
        </div>
      ) : null}

      {values.introMode !== "none" ? (
        summary ? (
          <p className="flex items-start gap-1.5 text-xs text-emerald-700 dark:text-emerald-400">
            <CheckCircle2 className="mt-px size-3.5 shrink-0" aria-hidden="true" />
            <span>{summary}</span>
          </p>
        ) : (
          <p className="text-xs text-muted-foreground">{t("subscription.form.intro.incomplete")}</p>
        )
      ) : null}
    </div>
  )
}
