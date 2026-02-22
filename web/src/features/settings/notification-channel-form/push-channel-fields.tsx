import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import {
  NTFY_PRIORITY_OPTIONS,
  NTFY_PRIORITY_UNSET,
  NTFY_TAG_PRESETS,
  PUSHOVER_SOUND_DEVICE_DEFAULT,
  PUSHOVER_SOUND_OPTIONS,
} from "./constants"
import type { BaseChannelConfigFieldProps } from "./field-props"

export function PushdeerConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="pd-key">{t("settings.notifications.channels.configFields.pushdeerPushKey")}</Label>
        <Input
          id="pd-key"
          placeholder={t("settings.notifications.channels.configFields.pushdeerPushKeyPlaceholder")}
          value={values.pushdeerPushKey}
          onChange={(e) => onValueChange("pushdeerPushKey", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="pd-server-url">{t("settings.notifications.channels.configFields.pushdeerServerUrl")}</Label>
        <Input
          id="pd-server-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.pushdeerServerUrlPlaceholder")}
          value={values.pushdeerServerUrl}
          onChange={(e) => onValueChange("pushdeerServerUrl", e.target.value)}
        />
      </div>
    </>
  )
}

export function PushplusConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="pp-token">{t("settings.notifications.channels.configFields.pushplusToken")}</Label>
        <Input
          id="pp-token"
          placeholder={t("settings.notifications.channels.configFields.pushplusTokenPlaceholder")}
          value={values.pushplusToken}
          onChange={(e) => onValueChange("pushplusToken", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="pp-topic">{t("settings.notifications.channels.configFields.pushplusTopic")}</Label>
        <Input
          id="pp-topic"
          placeholder={t("settings.notifications.channels.configFields.pushplusTopicPlaceholder")}
          value={values.pushplusTopic}
          onChange={(e) => onValueChange("pushplusTopic", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="pp-endpoint">{t("settings.notifications.channels.configFields.pushplusEndpoint")}</Label>
        <Input
          id="pp-endpoint"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.pushplusEndpointPlaceholder")}
          value={values.pushplusEndpoint}
          onChange={(e) => onValueChange("pushplusEndpoint", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.configFields.pushplusTemplate")}</Label>
        <Select value={values.pushplusTemplate} onValueChange={(value) => onValueChange("pushplusTemplate", value)}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="markdown">Markdown</SelectItem>
            <SelectItem value="html">HTML</SelectItem>
            <SelectItem value="txt">Text</SelectItem>
            <SelectItem value="json">JSON</SelectItem>
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="pp-channel">{t("settings.notifications.channels.configFields.pushplusChannel")}</Label>
        <Input
          id="pp-channel"
          placeholder={t("settings.notifications.channels.configFields.pushplusChannelPlaceholder")}
          value={values.pushplusChannel}
          onChange={(e) => onValueChange("pushplusChannel", e.target.value)}
        />
      </div>
    </>
  )
}

export function PushoverConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="po-token">{t("settings.notifications.channels.configFields.pushoverToken")}</Label>
        <Input
          id="po-token"
          placeholder={t("settings.notifications.channels.configFields.pushoverTokenPlaceholder")}
          value={values.pushoverToken}
          onChange={(e) => onValueChange("pushoverToken", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="po-user">{t("settings.notifications.channels.configFields.pushoverUser")}</Label>
        <Input
          id="po-user"
          placeholder={t("settings.notifications.channels.configFields.pushoverUserPlaceholder")}
          value={values.pushoverUser}
          onChange={(e) => onValueChange("pushoverUser", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="po-device">{t("settings.notifications.channels.configFields.pushoverDevice")}</Label>
        <Input
          id="po-device"
          placeholder={t("settings.notifications.channels.configFields.pushoverDevicePlaceholder")}
          value={values.pushoverDevice}
          onChange={(e) => onValueChange("pushoverDevice", e.target.value)}
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label>{t("settings.notifications.channels.configFields.pushoverPriority")}</Label>
          <Select value={values.pushoverPriority} onValueChange={(value) => onValueChange("pushoverPriority", value)}>
            <SelectTrigger id="po-priority" className="w-full max-w-full">
              <SelectValue className="max-w-[20ch] truncate" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="-2">{t("settings.notifications.channels.configFields.pushoverPriorityOptionLowest")}</SelectItem>
              <SelectItem value="-1">{t("settings.notifications.channels.configFields.pushoverPriorityOptionLow")}</SelectItem>
              <SelectItem value="0">{t("settings.notifications.channels.configFields.pushoverPriorityOptionNormal")}</SelectItem>
              <SelectItem value="1">{t("settings.notifications.channels.configFields.pushoverPriorityOptionHigh")}</SelectItem>
              <SelectItem value="2">{t("settings.notifications.channels.configFields.pushoverPriorityOptionEmergency")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-2">
          <Label>{t("settings.notifications.channels.configFields.pushoverSound")}</Label>
          <Select
            value={values.pushoverSound || PUSHOVER_SOUND_DEVICE_DEFAULT}
            onValueChange={(value) =>
              onValueChange("pushoverSound", value === PUSHOVER_SOUND_DEVICE_DEFAULT ? "" : value)
            }
          >
            <SelectTrigger id="po-sound" className="w-full max-w-full">
              <SelectValue className="max-w-[20ch] truncate" />
            </SelectTrigger>
            <SelectContent>
              {PUSHOVER_SOUND_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {t(`settings.notifications.channels.configFields.${option.i18nKey}`)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="po-endpoint">{t("settings.notifications.channels.configFields.pushoverEndpoint")}</Label>
        <Input
          id="po-endpoint"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.pushoverEndpointPlaceholder")}
          value={values.pushoverEndpoint}
          onChange={(e) => onValueChange("pushoverEndpoint", e.target.value)}
        />
      </div>
    </>
  )
}

export function GotifyConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="gotify-url">{t("settings.notifications.channels.configFields.gotifyUrl")}</Label>
        <Input
          id="gotify-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.gotifyUrlPlaceholder")}
          value={values.gotifyUrl}
          onChange={(e) => onValueChange("gotifyUrl", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="gotify-token">{t("settings.notifications.channels.configFields.gotifyToken")}</Label>
        <Input
          id="gotify-token"
          placeholder={t("settings.notifications.channels.configFields.gotifyTokenPlaceholder")}
          value={values.gotifyToken}
          onChange={(e) => onValueChange("gotifyToken", e.target.value)}
          required
        />
      </div>
    </>
  )
}

export function NtfyConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  const appendNtfyTagPreset = (tag: string) => {
    const currentTags = values.ntfyTags
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean)

    if (currentTags.includes(tag)) {
      return
    }

    onValueChange("ntfyTags", [...currentTags, tag].join(","))
  }

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="ntfy-url">{t("settings.notifications.channels.configFields.ntfyUrl")}</Label>
        <Input
          id="ntfy-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.ntfyUrlPlaceholder")}
          value={values.ntfyUrl}
          onChange={(e) => onValueChange("ntfyUrl", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ntfy-topic">{t("settings.notifications.channels.configFields.ntfyTopic")}</Label>
        <Input
          id="ntfy-topic"
          placeholder={t("settings.notifications.channels.configFields.ntfyTopicPlaceholder")}
          value={values.ntfyTopic}
          onChange={(e) => onValueChange("ntfyTopic", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ntfy-token">{t("settings.notifications.channels.configFields.ntfyToken")}</Label>
        <Input
          id="ntfy-token"
          placeholder={t("settings.notifications.channels.configFields.ntfyTokenPlaceholder")}
          value={values.ntfyToken}
          onChange={(e) => onValueChange("ntfyToken", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.configFields.ntfyPriority")}</Label>
        <Select
          value={values.ntfyPriority || NTFY_PRIORITY_UNSET}
          onValueChange={(value) => onValueChange("ntfyPriority", value === NTFY_PRIORITY_UNSET ? "" : value)}
        >
          <SelectTrigger id="ntfy-priority" className="w-full max-w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {NTFY_PRIORITY_OPTIONS.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {t(`settings.notifications.channels.configFields.${option.i18nKey}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="ntfy-tags">{t("settings.notifications.channels.configFields.ntfyTags")}</Label>
        <Input
          id="ntfy-tags"
          placeholder={t("settings.notifications.channels.configFields.ntfyTagsPlaceholder")}
          value={values.ntfyTags}
          onChange={(e) => onValueChange("ntfyTags", e.target.value)}
        />
        <div className="flex flex-wrap gap-2">
          {NTFY_TAG_PRESETS.map((tag) => (
            <Button
              key={tag}
              type="button"
              size="sm"
              variant="outline"
              className="h-7 px-2 text-xs"
              onClick={() => appendNtfyTagPreset(tag)}
            >
              :{tag}:
            </Button>
          ))}
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="ntfy-click">{t("settings.notifications.channels.configFields.ntfyClick")}</Label>
        <Input
          id="ntfy-click"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.ntfyClickPlaceholder")}
          value={values.ntfyClick}
          onChange={(e) => onValueChange("ntfyClick", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="ntfy-icon">{t("settings.notifications.channels.configFields.ntfyIcon")}</Label>
        <Input
          id="ntfy-icon"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.ntfyIconPlaceholder")}
          value={values.ntfyIcon}
          onChange={(e) => onValueChange("ntfyIcon", e.target.value)}
        />
      </div>
    </>
  )
}

export function BarkConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="bark-url">{t("settings.notifications.channels.configFields.barkUrl")}</Label>
        <Input
          id="bark-url"
          type="url"
          placeholder={t("settings.notifications.channels.configFields.barkUrlPlaceholder")}
          value={values.barkUrl}
          onChange={(e) => onValueChange("barkUrl", e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="bark-key">{t("settings.notifications.channels.configFields.barkDeviceKey")}</Label>
        <Input
          id="bark-key"
          placeholder={t("settings.notifications.channels.configFields.barkDeviceKeyPlaceholder")}
          value={values.barkDeviceKey}
          onChange={(e) => onValueChange("barkDeviceKey", e.target.value)}
          required
        />
      </div>
    </>
  )
}

export function ServerChanConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <div className="space-y-2">
      <Label htmlFor="sc-key">{t("settings.notifications.channels.configFields.serverChanSendKey")}</Label>
      <Input
        id="sc-key"
        placeholder={t("settings.notifications.channels.configFields.serverChanSendKeyPlaceholder")}
        value={values.serverChanSendKey}
        onChange={(e) => onValueChange("serverChanSendKey", e.target.value)}
        required
      />
    </div>
  )
}
