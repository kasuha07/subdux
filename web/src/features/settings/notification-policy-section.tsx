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
  const [notifyManualRenewDaily, setNotifyManualRenewDaily] = useState(policy.notify_manual_renew_daily)
  const [quietHoursEnabled, setQuietHoursEnabled] = useState(policy.quiet_hours_enabled)
  const [quietHoursStart, setQuietHoursStart] = useState(policy.quiet_hours_start || "22:00")
  const [quietHoursEnd, setQuietHoursEnd] = useState(policy.quiet_hours_end || "08:00")

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const parsed = parseInt(daysBefore, 10)
    const normalized = isNaN(parsed) ? 3 : Math.min(MAX_NOTIFICATION_DAYS_BEFORE, Math.max(0, parsed))
    void onSave({
      days_before: normalized,
      notify_on_due_day: notifyOnDueDay,
      notify_manual_renew_daily: notifyManualRenewDaily,
      quiet_hours_enabled: quietHoursEnabled,
      quiet_hours_start: quietHoursStart,
      quiet_hours_end: quietHoursEnd,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <h2 className="text-base font-semibold tracking-tight select-none">{t("settings.notifications.policy.title")}</h2>
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

      <div className="flex items-center gap-3">
        <Switch
          id="manual-renew-daily"
          checked={notifyManualRenewDaily}
          onCheckedChange={setNotifyManualRenewDaily}
        />
        <Label htmlFor="manual-renew-daily" className="cursor-pointer">
          {t("settings.notifications.policy.notifyManualRenewDaily")}
        </Label>
      </div>

      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <Switch
            id="quiet-hours-enabled"
            checked={quietHoursEnabled}
            onCheckedChange={setQuietHoursEnabled}
          />
          <Label htmlFor="quiet-hours-enabled" className="cursor-pointer">
            {t("settings.notifications.policy.quietHours")}
          </Label>
        </div>
        <p className="text-xs text-muted-foreground">
          {t("settings.notifications.policy.quietHoursHint")}
        </p>
        <div className="flex items-center gap-3">
          <div className="space-y-1">
            <Label htmlFor="quiet-hours-start" className="text-xs">
              {t("settings.notifications.policy.quietHoursStart")}
            </Label>
            <Input
              id="quiet-hours-start"
              type="time"
              value={quietHoursStart}
              disabled={!quietHoursEnabled}
              onChange={(e) => setQuietHoursStart(e.target.value)}
            />
          </div>
          <div className="space-y-1">
            <Label htmlFor="quiet-hours-end" className="text-xs">
              {t("settings.notifications.policy.quietHoursEnd")}
            </Label>
            <Input
              id="quiet-hours-end"
              type="time"
              value={quietHoursEnd}
              disabled={!quietHoursEnabled}
              onChange={(e) => setQuietHoursEnd(e.target.value)}
            />
          </div>
        </div>
      </div>

      <Button type="submit" size="sm" disabled={saving}>
        {t("settings.notifications.channels.save")}
      </Button>
    </form>
  )
}
