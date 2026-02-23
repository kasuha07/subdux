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

import type { SubscriptionNotifySetting } from "@/features/subscriptions/hooks/use-subscription-form-state"

interface SubscriptionNotificationFieldsProps {
  notifyDaysBefore: string
  notifyEnabled: SubscriptionNotifySetting
  onNotifyDaysBeforeChange: (value: string) => void
  onNotifyEnabledChange: (value: SubscriptionNotifySetting) => void
}

const MAX_NOTIFICATION_DAYS_BEFORE = 10

export default function SubscriptionNotificationFields({
  notifyDaysBefore,
  notifyEnabled,
  onNotifyDaysBeforeChange,
  onNotifyEnabledChange,
}: SubscriptionNotificationFieldsProps) {
  const { t } = useTranslation()
  const showDaysBeforeOverride = notifyEnabled === "enabled"

  return (
    <div className={showDaysBeforeOverride ? "grid grid-cols-1 gap-3 sm:grid-cols-2" : "space-y-0"}>
      <div className="space-y-2">
        <Label>{t("settings.notifications.subscription.title")}</Label>
        <Select value={notifyEnabled} onValueChange={onNotifyEnabledChange}>
          <SelectTrigger>
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="default">
              {t("settings.notifications.subscription.useDefault")}
            </SelectItem>
            <SelectItem value="enabled">
              {t("settings.notifications.subscription.enabled")}
            </SelectItem>
            <SelectItem value="disabled">
              {t("settings.notifications.subscription.disabled")}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      {showDaysBeforeOverride && (
        <div className="space-y-2">
          <Label htmlFor="notify-days">{t("settings.notifications.subscription.daysBeforeOverride")}</Label>
          <Input
            id="notify-days"
            type="number"
            min="0"
            max={MAX_NOTIFICATION_DAYS_BEFORE}
            placeholder="e.g., 7"
            value={notifyDaysBefore}
            onChange={(event) => onNotifyDaysBeforeChange(event.target.value)}
          />
        </div>
      )}
    </div>
  )
}
