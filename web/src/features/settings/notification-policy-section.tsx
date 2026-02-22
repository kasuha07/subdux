import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import type { NotificationPolicy, UpdateNotificationPolicyInput } from "@/types"

interface Props {
  onSave: (input: UpdateNotificationPolicyInput) => void | Promise<void>
  policy: NotificationPolicy
  saving: boolean
}

const MAX_NOTIFICATION_DAYS_BEFORE = 10

export function NotificationPolicySection({ onSave, policy, saving }: Props) {
  const { t } = useTranslation()
  const [daysBefore, setDaysBefore] = useState(policy.days_before.toString())
  const [notifyOnDueDay, setNotifyOnDueDay] = useState(policy.notify_on_due_day)

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const parsed = parseInt(daysBefore, 10)
    const normalized = isNaN(parsed) ? 3 : Math.min(MAX_NOTIFICATION_DAYS_BEFORE, Math.max(0, parsed))
    void onSave({
      days_before: normalized,
      notify_on_due_day: notifyOnDueDay,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.notifications.policy.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.notifications.policy.description")}
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="days-before">{t("settings.notifications.policy.daysBefore")}</Label>
        <Input
          id="days-before"
          type="number"
          min="0"
          max={MAX_NOTIFICATION_DAYS_BEFORE}
          value={daysBefore}
          onChange={(e) => setDaysBefore(e.target.value)}
        />
        <p className="text-xs text-muted-foreground">
          {t("settings.notifications.policy.daysBeforeHint")}
        </p>
      </div>

      <div className="flex items-center gap-3">
        <Switch
          id="on-due-day"
          checked={notifyOnDueDay}
          onCheckedChange={setNotifyOnDueDay}
        />
        <Label htmlFor="on-due-day" className="cursor-pointer">
          {t("settings.notifications.policy.notifyOnDueDay")}
        </Label>
      </div>

      <Button type="submit" size="sm" disabled={saving}>
        {t("settings.notifications.channels.save")}
      </Button>
    </form>
  )
}
