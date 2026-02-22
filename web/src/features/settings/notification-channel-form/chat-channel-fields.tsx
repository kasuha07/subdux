import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import type { BaseChannelConfigFieldProps } from "./field-props"
import { parseWebhookMethod } from "./utils"

export function TelegramConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="tg-token">{t("settings.notifications.channels.configFields.botToken")}</Label>
        <Input
          id="tg-token"
          placeholder={t("settings.notifications.channels.configFields.botTokenPlaceholder")}
          value={values.botToken}
          onChange={(e) => onValueChange("botToken", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="tg-chat">{t("settings.notifications.channels.configFields.chatId")}</Label>
        <Input
          id="tg-chat"
          placeholder={t("settings.notifications.channels.configFields.chatIdPlaceholder")}
          value={values.chatId}
          onChange={(e) => onValueChange("chatId", e.target.value)}
          required
        />
      </div>
    </>
  )
}

export function WebhookConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="wh-url">{t("settings.notifications.channels.configFields.url")}</Label>
        <Input
          id="wh-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.urlPlaceholder")}
          value={values.webhookUrl}
          onChange={(e) => onValueChange("webhookUrl", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.configFields.method")}</Label>
        <Select
          value={values.webhookMethod}
          onValueChange={(value) => onValueChange("webhookMethod", parseWebhookMethod(value))}
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="POST">{t("settings.notifications.channels.configFields.methodPost")}</SelectItem>
            <SelectItem value="PUT">{t("settings.notifications.channels.configFields.methodPut")}</SelectItem>
            <SelectItem value="GET">{t("settings.notifications.channels.configFields.methodGet")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="wh-secret">{t("settings.notifications.channels.configFields.secret")}</Label>
        <Input
          id="wh-secret"
          placeholder={t("settings.notifications.channels.configFields.secretPlaceholder")}
          value={values.webhookSecret}
          onChange={(e) => onValueChange("webhookSecret", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="wh-headers">{t("settings.notifications.channels.configFields.headers")}</Label>
        <textarea
          id="wh-headers"
          className="flex min-h-[96px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
          placeholder={t("settings.notifications.channels.configFields.headersPlaceholder")}
          value={values.webhookHeaders}
          onChange={(e) => onValueChange("webhookHeaders", e.target.value)}
        />
        <p className="text-xs text-muted-foreground">{t("settings.notifications.channels.configFields.headersHint")}</p>
      </div>
    </>
  )
}

export function NapcatConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="nc-url">{t("settings.notifications.channels.configFields.napcatUrl")}</Label>
        <Input
          id="nc-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.napcatUrlPlaceholder")}
          value={values.napcatUrl}
          onChange={(e) => onValueChange("napcatUrl", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="nc-token">{t("settings.notifications.channels.configFields.napcatAccessToken")}</Label>
        <Input
          id="nc-token"
          placeholder={t("settings.notifications.channels.configFields.napcatAccessTokenPlaceholder")}
          value={values.napcatAccessToken}
          onChange={(e) => onValueChange("napcatAccessToken", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.configFields.napcatMessageType")}</Label>
        <Select
          value={values.napcatMessageType}
          onValueChange={(value) => onValueChange("napcatMessageType", value as "private" | "group")}
        >
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="private">{t("settings.notifications.channels.configFields.napcatMessageTypePrivate")}</SelectItem>
            <SelectItem value="group">{t("settings.notifications.channels.configFields.napcatMessageTypeGroup")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
      {values.napcatMessageType === "private" && (
        <div className="space-y-2">
          <Label htmlFor="nc-userid">{t("settings.notifications.channels.configFields.napcatUserId")}</Label>
          <Input
            id="nc-userid"
            placeholder={t("settings.notifications.channels.configFields.napcatUserIdPlaceholder")}
            value={values.napcatUserId}
            onChange={(e) => onValueChange("napcatUserId", e.target.value)}
            required
          />
        </div>
      )}
      {values.napcatMessageType === "group" && (
        <div className="space-y-2">
          <Label htmlFor="nc-groupid">{t("settings.notifications.channels.configFields.napcatGroupId")}</Label>
          <Input
            id="nc-groupid"
            placeholder={t("settings.notifications.channels.configFields.napcatGroupIdPlaceholder")}
            value={values.napcatGroupId}
            onChange={(e) => onValueChange("napcatGroupId", e.target.value)}
            required
          />
        </div>
      )}
    </>
  )
}
