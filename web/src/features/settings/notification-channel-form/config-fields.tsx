import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import {
  NapcatConfigFields,
  TelegramConfigFields,
  WebhookConfigFields,
} from "./chat-channel-fields"
import { CHANNEL_TYPE_OPTIONS } from "./constants"
import {
  DingtalkConfigFields,
  FeishuConfigFields,
  WecomConfigFields,
} from "./enterprise-channel-fields"
import {
  ResendConfigFields,
  SmtpConfigFields,
} from "./email-channel-fields"
import type { BaseChannelConfigFieldProps } from "./field-props"
import {
  BarkConfigFields,
  GotifyConfigFields,
  NtfyConfigFields,
  PushdeerConfigFields,
  PushplusConfigFields,
  PushoverConfigFields,
  ServerChanConfigFields,
} from "./push-channel-fields"
import type { ChannelType } from "./types"

interface Props extends BaseChannelConfigFieldProps {
  isEditing: boolean
  type: ChannelType
  onTypeChange: (value: ChannelType) => void
}

export function NotificationChannelConfigFields({
  isEditing,
  onTypeChange,
  onValueChange,
  t,
  type,
  values,
}: Props) {
  const fieldProps: BaseChannelConfigFieldProps = {
    onValueChange,
    t,
    values,
  }

  return (
    <>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.typeLabel")}</Label>
        <Select value={type} onValueChange={(value) => onTypeChange(value as ChannelType)} disabled={isEditing}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            {CHANNEL_TYPE_OPTIONS.map((option) => (
              <SelectItem key={option} value={option}>
                {t(`settings.notifications.channels.type.${option}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {renderChannelConfigFields(type, fieldProps)}
    </>
  )
}

function renderChannelConfigFields(type: ChannelType, props: BaseChannelConfigFieldProps) {
  switch (type) {
    case "smtp":
      return <SmtpConfigFields {...props} />
    case "resend":
      return <ResendConfigFields {...props} />
    case "telegram":
      return <TelegramConfigFields {...props} />
    case "webhook":
      return <WebhookConfigFields {...props} />
    case "pushdeer":
      return <PushdeerConfigFields {...props} />
    case "pushplus":
      return <PushplusConfigFields {...props} />
    case "pushover":
      return <PushoverConfigFields {...props} />
    case "gotify":
      return <GotifyConfigFields {...props} />
    case "ntfy":
      return <NtfyConfigFields {...props} />
    case "bark":
      return <BarkConfigFields {...props} />
    case "serverchan":
      return <ServerChanConfigFields {...props} />
    case "feishu":
      return <FeishuConfigFields {...props} />
    case "wecom":
      return <WecomConfigFields {...props} />
    case "dingtalk":
      return <DingtalkConfigFields {...props} />
    case "napcat":
      return <NapcatConfigFields {...props} />
    default:
      return null
  }
}
