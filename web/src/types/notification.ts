export type NotificationChannelType =
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

export interface NotificationChannel {
  id: number
  type: NotificationChannelType
  enabled: boolean
  config: string
  configured_secret_fields?: string[]
  configured_webhook_header_keys?: string[]
  created_at: string
  updated_at: string
}

export interface SMTPChannelConfig {
  to_email?: string
}

export interface ResendChannelConfig {
  api_key: string
  from_email: string
  to_email: string
}

export interface TelegramChannelConfig {
  bot_token: string
  chat_id: string
}

export interface WebhookChannelConfig {
  url: string
  secret?: string
  method?: WebhookMethod
  headers?: Record<string, string>
}

export interface PushDeerChannelConfig {
  push_key: string
  server_url?: string
}

export interface PushplusChannelConfig {
  token: string
  endpoint?: string
  template?: string
  channel?: string
  topic?: string
}

export interface PushoverChannelConfig {
  token: string
  user: string
  device?: string
  priority?: number
  sound?: string
  endpoint?: string
}

export interface GotifyChannelConfig {
  url: string
  token: string
}

export interface NtfyChannelConfig {
  url?: string
  topic: string
  token?: string
  username?: string
  password?: string
  priority?: string
  tags?: string
  click?: string
  icon?: string
}

export interface BarkChannelConfig {
  url?: string
  device_key: string
}

export interface ServerChanChannelConfig {
  send_key: string
}

export interface FeishuChannelConfig {
  webhook_url: string
  secret?: string
}

export interface WeComChannelConfig {
  webhook_url: string
}

export interface DingTalkChannelConfig {
  webhook_url: string
  secret?: string
}

export interface NapCatChannelConfig {
  url: string
  access_token?: string
  message_type?: "private" | "group"
  user_id?: string
  group_id?: string
}

export type ChannelConfig =
  | SMTPChannelConfig
  | ResendChannelConfig
  | TelegramChannelConfig
  | WebhookChannelConfig
  | PushDeerChannelConfig
  | PushplusChannelConfig
  | PushoverChannelConfig
  | GotifyChannelConfig
  | NtfyChannelConfig
  | BarkChannelConfig
  | ServerChanChannelConfig
  | FeishuChannelConfig
  | WeComChannelConfig
  | DingTalkChannelConfig
  | NapCatChannelConfig

export interface CreateNotificationChannelInput {
  type: string
  enabled: boolean
  config: string
}

export interface UpdateNotificationChannelInput {
  enabled?: boolean
  config?: string
}

export interface NotificationPolicy {
  days_before: number
  notify_on_due_day: boolean
}

export interface UpdateNotificationPolicyInput {
  days_before?: number
  notify_on_due_day?: boolean
}

export interface NotificationLog {
  id: number
  subscription_id: number
  channel_type: string
  notify_date: string
  status: string
  error: string
  sent_at: string
}

export interface NotificationTemplate {
  id: number
  user_id: number
  channel_type: string | null
  format: string
  template: string
  created_at: string
  updated_at: string
}

export interface CreateTemplateInput {
  channel_type?: string | null
  format: string
  template: string
}

export interface UpdateTemplateInput {
  format?: string
  template?: string
}

export interface PreviewTemplateInput {
  format: string
  template: string
}
