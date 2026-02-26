import { Label } from "@/components/ui/label"

import type { BaseChannelConfigFieldProps } from "./field-props"
import { SecretInput } from "./secret-input"

export function FeishuConfigFields({ isSecretFieldConfigured, onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="feishu-url">{t("settings.notifications.channels.configFields.feishuWebhookUrl")}</Label>
        <SecretInput
          id="feishu-url"
          type="password"
          placeholder={t("settings.notifications.channels.configFields.feishuWebhookUrlPlaceholder")}
          value={values.feishuWebhookUrl}
          configured={isSecretFieldConfigured("feishuWebhookUrl")}
          onValueChange={(value) => onValueChange("feishuWebhookUrl", value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="feishu-secret">{t("settings.notifications.channels.configFields.feishuSecret")}</Label>
        <SecretInput
          id="feishu-secret"
          type="password"
          placeholder={t("settings.notifications.channels.configFields.feishuSecretPlaceholder")}
          value={values.feishuSecret}
          configured={isSecretFieldConfigured("feishuSecret")}
          onValueChange={(value) => onValueChange("feishuSecret", value)}
        />
      </div>
    </>
  )
}

export function WecomConfigFields({ isSecretFieldConfigured, onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="wecom-url">{t("settings.notifications.channels.configFields.wecomWebhookUrl")}</Label>
      <SecretInput
        id="wecom-url"
        type="password"
        placeholder={t("settings.notifications.channels.configFields.wecomWebhookUrlPlaceholder")}
        value={values.wecomWebhookUrl}
        configured={isSecretFieldConfigured("wecomWebhookUrl")}
        onValueChange={(value) => onValueChange("wecomWebhookUrl", value)}
        required
      />
    </div>
  )
}

export function DingtalkConfigFields({ isSecretFieldConfigured, onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="dt-url">{t("settings.notifications.channels.configFields.dingtalkWebhookUrl")}</Label>
        <SecretInput
          id="dt-url"
          type="password"
          placeholder={t("settings.notifications.channels.configFields.dingtalkWebhookUrlPlaceholder")}
          value={values.dingtalkWebhookUrl}
          configured={isSecretFieldConfigured("dingtalkWebhookUrl")}
          onValueChange={(value) => onValueChange("dingtalkWebhookUrl", value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="dt-secret">{t("settings.notifications.channels.configFields.dingtalkSecret")}</Label>
        <SecretInput
          id="dt-secret"
          type="password"
          placeholder={t("settings.notifications.channels.configFields.dingtalkSecretPlaceholder")}
          value={values.dingtalkSecret}
          configured={isSecretFieldConfigured("dingtalkSecret")}
          onValueChange={(value) => onValueChange("dingtalkSecret", value)}
        />
      </div>
    </>
  )
}
