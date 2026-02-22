import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import type { BaseChannelConfigFieldProps } from "./field-props"

export function FeishuConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="feishu-url">{t("settings.notifications.channels.configFields.feishuWebhookUrl")}</Label>
        <Input
          id="feishu-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.feishuWebhookUrlPlaceholder")}
          value={values.feishuWebhookUrl}
          onChange={(e) => onValueChange("feishuWebhookUrl", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="feishu-secret">{t("settings.notifications.channels.configFields.feishuSecret")}</Label>
        <Input
          id="feishu-secret"
          placeholder={t("settings.notifications.channels.configFields.feishuSecretPlaceholder")}
          value={values.feishuSecret}
          onChange={(e) => onValueChange("feishuSecret", e.target.value)}
        />
      </div>
    </>
  )
}

export function WecomConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="wecom-url">{t("settings.notifications.channels.configFields.wecomWebhookUrl")}</Label>
      <Input
        id="wecom-url"
        type="url"
        placeholder={t("settings.notifications.channels.configFields.wecomWebhookUrlPlaceholder")}
        value={values.wecomWebhookUrl}
        onChange={(e) => onValueChange("wecomWebhookUrl", e.target.value)}
        required
      />
    </div>
  )
}

export function DingtalkConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="dt-url">{t("settings.notifications.channels.configFields.dingtalkWebhookUrl")}</Label>
        <Input
          id="dt-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.dingtalkWebhookUrlPlaceholder")}
          value={values.dingtalkWebhookUrl}
          onChange={(e) => onValueChange("dingtalkWebhookUrl", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dt-secret">{t("settings.notifications.channels.configFields.dingtalkSecret")}</Label>
        <Input
          id="dt-secret"
          placeholder={t("settings.notifications.channels.configFields.dingtalkSecretPlaceholder")}
          value={values.dingtalkSecret}
          onChange={(e) => onValueChange("dingtalkSecret", e.target.value)}
        />
      </div>
    </>
  )
}
