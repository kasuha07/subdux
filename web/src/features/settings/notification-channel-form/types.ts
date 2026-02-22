import type { NotificationChannel } from "@/types"

export type ChannelType =
  | "smtp"
  | "resend"
  | "telegram"
  | "webhook"
  | "gotify"
  | "ntfy"
  | "bark"
  | "serverchan"
  | "feishu"
  | "wecom"
  | "dingtalk"
  | "pushdeer"
  | "pushplus"
  | "pushover"
  | "napcat"

export type WebhookMethod = "GET" | "POST" | "PUT"

export interface NotificationChannelFormProps {
  channel: NotificationChannel | null
  onClose: () => void
  onSave: (type: string, config: string) => void | Promise<void>
  open: boolean
  saving: boolean
}

export interface NotificationChannelFormValues {
  smtpHost: string
  smtpPort: string
  smtpUsername: string
  smtpPassword: string
  smtpFromEmail: string
  smtpFromName: string
  smtpToEmail: string
  smtpEncryption: string

  apiKey: string
  fromEmail: string
  resendToEmail: string

  botToken: string
  chatId: string

  webhookUrl: string
  webhookSecret: string
  webhookMethod: WebhookMethod
  webhookHeaders: string

  pushdeerPushKey: string
  pushdeerServerUrl: string

  pushplusToken: string
  pushplusTopic: string
  pushplusEndpoint: string
  pushplusTemplate: string
  pushplusChannel: string

  pushoverToken: string
  pushoverUser: string
  pushoverDevice: string
  pushoverPriority: string
  pushoverSound: string
  pushoverEndpoint: string

  gotifyUrl: string
  gotifyToken: string

  ntfyUrl: string
  ntfyTopic: string
  ntfyToken: string
  ntfyPriority: string
  ntfyTags: string
  ntfyClick: string
  ntfyIcon: string

  barkUrl: string
  barkDeviceKey: string

  serverChanSendKey: string

  feishuWebhookUrl: string
  feishuSecret: string

  wecomWebhookUrl: string

  dingtalkWebhookUrl: string
  dingtalkSecret: string

  napcatUrl: string
  napcatAccessToken: string
  napcatMessageType: "private" | "group"
  napcatUserId: string
  napcatGroupId: string
}
