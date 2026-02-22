import type { NotificationChannel } from "@/types"

import { WEBHOOK_HEADERS_PARSE_ERROR } from "./constants"
import type { ChannelType, NotificationChannelFormValues, WebhookMethod } from "./types"

function normalizeNtfyPriority(raw: unknown): string {
  if (raw == null) {
    return ""
  }

  const normalized = String(raw).trim().toLowerCase()
  if (normalized === "") {
    return ""
  }

  if (normalized === "1" || normalized === "2" || normalized === "3" || normalized === "4" || normalized === "5") {
    return normalized
  }

  if (normalized === "min") {
    return "1"
  }
  if (normalized === "low") {
    return "2"
  }
  if (normalized === "default") {
    return "3"
  }
  if (normalized === "high") {
    return "4"
  }
  if (normalized === "max" || normalized === "urgent") {
    return "5"
  }

  return ""
}

function parseConfig(raw: string): Record<string, string> {
  try {
    return JSON.parse(raw) as Record<string, string>
  } catch {
    return {}
  }
}

export function parseWebhookMethod(raw: string): WebhookMethod {
  const method = raw.trim().toUpperCase()
  if (method === "GET" || method === "PUT") {
    return method
  }
  return "POST"
}

function parseWebhookHeadersDisplay(rawConfig: string): string {
  try {
    const parsed = JSON.parse(rawConfig) as { headers?: unknown }
    if (!parsed.headers || typeof parsed.headers !== "object" || Array.isArray(parsed.headers)) {
      return ""
    }

    const normalized: Record<string, string> = {}
    for (const [key, value] of Object.entries(parsed.headers as Record<string, unknown>)) {
      if (typeof value === "string") {
        normalized[key] = value
      }
    }

    if (Object.keys(normalized).length === 0) {
      return ""
    }

    return JSON.stringify(normalized, null, 2)
  } catch {
    return ""
  }
}

function parseWebhookHeadersInput(raw: string): Record<string, string> {
  const parsed = JSON.parse(raw) as unknown
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(WEBHOOK_HEADERS_PARSE_ERROR)
  }

  const headers: Record<string, string> = {}
  for (const [key, value] of Object.entries(parsed as Record<string, unknown>)) {
    const trimmedKey = key.trim()
    if (trimmedKey === "" || typeof value !== "string") {
      throw new Error(WEBHOOK_HEADERS_PARSE_ERROR)
    }
    headers[trimmedKey] = value
  }

  return headers
}

export function createInitialValues(channel: NotificationChannel | null): NotificationChannelFormValues {
  const initCfg = channel ? parseConfig(channel.config) : {}

  return {
    smtpHost: initCfg.host ?? "",
    smtpPort: initCfg.port ?? "587",
    smtpUsername: initCfg.username ?? "",
    smtpPassword: initCfg.password ?? "",
    smtpFromEmail: initCfg.from_email ?? "",
    smtpFromName: initCfg.from_name ?? "",
    smtpToEmail: initCfg.to_email ?? "",
    smtpEncryption: initCfg.encryption ?? "starttls",

    apiKey: initCfg.api_key ?? "",
    fromEmail: initCfg.from_email ?? "",
    resendToEmail: initCfg.to_email ?? "",

    botToken: initCfg.bot_token ?? "",
    chatId: initCfg.chat_id ?? "",

    webhookUrl: initCfg.url ?? "",
    webhookSecret: initCfg.secret ?? "",
    webhookMethod: parseWebhookMethod(initCfg.method ?? ""),
    webhookHeaders: channel ? parseWebhookHeadersDisplay(channel.config) : "",

    pushdeerPushKey: initCfg.push_key ?? "",
    pushdeerServerUrl: initCfg.server_url ?? "",

    pushplusToken: initCfg.token ?? "",
    pushplusTopic: initCfg.topic ?? "",
    pushplusEndpoint: initCfg.endpoint ?? "",
    pushplusTemplate: initCfg.template ?? "markdown",
    pushplusChannel: initCfg.channel ?? "",

    pushoverToken: initCfg.token ?? "",
    pushoverUser: initCfg.user ?? "",
    pushoverDevice: initCfg.device ?? "",
    pushoverPriority: initCfg.priority != null ? String(initCfg.priority) : "0",
    pushoverSound: initCfg.sound ?? "",
    pushoverEndpoint: initCfg.endpoint ?? "",

    gotifyUrl: initCfg.url ?? "",
    gotifyToken: initCfg.token ?? "",

    ntfyUrl: initCfg.url ?? "",
    ntfyTopic: initCfg.topic ?? "",
    ntfyToken: initCfg.token ?? "",
    ntfyPriority: normalizeNtfyPriority(initCfg.priority),
    ntfyTags: initCfg.tags ?? "",
    ntfyClick: initCfg.click ?? "",
    ntfyIcon: initCfg.icon ?? "",

    barkUrl: initCfg.url ?? "",
    barkDeviceKey: initCfg.device_key ?? "",

    serverChanSendKey: initCfg.send_key ?? "",

    feishuWebhookUrl: initCfg.webhook_url ?? "",
    feishuSecret: initCfg.secret ?? "",

    wecomWebhookUrl: initCfg.webhook_url ?? "",

    dingtalkWebhookUrl: initCfg.webhook_url ?? "",
    dingtalkSecret: initCfg.secret ?? "",

    napcatUrl: initCfg.url ?? "",
    napcatAccessToken: initCfg.access_token ?? "",
    napcatMessageType: (initCfg.message_type as "private" | "group") ?? "private",
    napcatUserId: initCfg.user_id ?? "",
    napcatGroupId: initCfg.group_id ?? "",
  }
}

export function buildConfig(type: ChannelType, values: NotificationChannelFormValues): string {
  switch (type) {
    case "smtp":
      return JSON.stringify({
        host: values.smtpHost.trim(),
        port: parseInt(values.smtpPort, 10) || 587,
        username: values.smtpUsername.trim(),
        password: values.smtpPassword,
        from_email: values.smtpFromEmail.trim(),
        from_name: values.smtpFromName.trim(),
        to_email: values.smtpToEmail.trim(),
        encryption: values.smtpEncryption,
      })
    case "resend":
      return JSON.stringify({
        api_key: values.apiKey.trim(),
        from_email: values.fromEmail.trim(),
        to_email: values.resendToEmail.trim(),
      })
    case "telegram":
      return JSON.stringify({ bot_token: values.botToken.trim(), chat_id: values.chatId.trim() })
    case "webhook": {
      const webhookConfig: Record<string, unknown> = {
        url: values.webhookUrl.trim(),
        method: values.webhookMethod,
      }
      if (values.webhookSecret.trim()) {
        webhookConfig.secret = values.webhookSecret.trim()
      }

      if (values.webhookHeaders.trim()) {
        const headers = parseWebhookHeadersInput(values.webhookHeaders)
        if (Object.keys(headers).length > 0) {
          webhookConfig.headers = headers
        }
      }

      return JSON.stringify(webhookConfig)
    }
    case "pushdeer":
      return JSON.stringify({
        push_key: values.pushdeerPushKey.trim(),
        ...(values.pushdeerServerUrl.trim() ? { server_url: values.pushdeerServerUrl.trim() } : {}),
      })
    case "pushplus":
      return JSON.stringify({
        token: values.pushplusToken.trim(),
        ...(values.pushplusTopic.trim() ? { topic: values.pushplusTopic.trim() } : {}),
        ...(values.pushplusEndpoint.trim() ? { endpoint: values.pushplusEndpoint.trim() } : {}),
        ...(values.pushplusTemplate.trim() ? { template: values.pushplusTemplate.trim() } : {}),
        ...(values.pushplusChannel.trim() ? { channel: values.pushplusChannel.trim() } : {}),
      })
    case "pushover":
      return JSON.stringify({
        token: values.pushoverToken.trim(),
        user: values.pushoverUser.trim(),
        ...(values.pushoverDevice.trim() ? { device: values.pushoverDevice.trim() } : {}),
        priority: parseInt(values.pushoverPriority, 10) || 0,
        ...(values.pushoverSound.trim() ? { sound: values.pushoverSound.trim() } : {}),
        ...(values.pushoverEndpoint.trim() ? { endpoint: values.pushoverEndpoint.trim() } : {}),
      })
    case "gotify":
      return JSON.stringify({ url: values.gotifyUrl.trim(), token: values.gotifyToken.trim() })
    case "ntfy":
      return JSON.stringify({
        ...(values.ntfyUrl.trim() ? { url: values.ntfyUrl.trim() } : {}),
        topic: values.ntfyTopic.trim(),
        ...(values.ntfyToken.trim() ? { token: values.ntfyToken.trim() } : {}),
        ...(values.ntfyPriority.trim() ? { priority: values.ntfyPriority.trim() } : {}),
        ...(values.ntfyTags.trim() ? { tags: values.ntfyTags.trim() } : {}),
        ...(values.ntfyClick.trim() ? { click: values.ntfyClick.trim() } : {}),
        ...(values.ntfyIcon.trim() ? { icon: values.ntfyIcon.trim() } : {}),
      })
    case "bark":
      return JSON.stringify({
        ...(values.barkUrl.trim() ? { url: values.barkUrl.trim() } : {}),
        device_key: values.barkDeviceKey.trim(),
      })
    case "serverchan":
      return JSON.stringify({ send_key: values.serverChanSendKey.trim() })
    case "feishu":
      return JSON.stringify({
        webhook_url: values.feishuWebhookUrl.trim(),
        ...(values.feishuSecret.trim() ? { secret: values.feishuSecret.trim() } : {}),
      })
    case "wecom":
      return JSON.stringify({ webhook_url: values.wecomWebhookUrl.trim() })
    case "dingtalk":
      return JSON.stringify({
        webhook_url: values.dingtalkWebhookUrl.trim(),
        ...(values.dingtalkSecret.trim() ? { secret: values.dingtalkSecret.trim() } : {}),
      })
    case "napcat":
      return JSON.stringify({
        url: values.napcatUrl.trim(),
        ...(values.napcatAccessToken.trim() ? { access_token: values.napcatAccessToken.trim() } : {}),
        message_type: values.napcatMessageType,
        ...(values.napcatMessageType === "private"
          ? { user_id: values.napcatUserId.trim() }
          : { group_id: values.napcatGroupId.trim() }),
      })
  }
}
