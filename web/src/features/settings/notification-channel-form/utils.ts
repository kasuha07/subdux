import type { NotificationChannel } from "@/types"

import { WEBHOOK_HEADERS_PARSE_ERROR } from "./constants"
import type { ChannelType, NotificationChannelFormValues, WebhookMethod } from "./types"

const CONFIGURED_SECRET_MASK = "••••••••"

const CHANNEL_SECRET_FIELD_TO_FORM_FIELD_MAP: Record<string, Partial<Record<string, keyof NotificationChannelFormValues>>> = {
  smtp: {
    password: "smtpPassword",
  },
  resend: {
    api_key: "apiKey",
  },
  telegram: {
    bot_token: "botToken",
  },
  webhook: {
    secret: "webhookSecret",
  },
  gotify: {
    token: "gotifyToken",
  },
  ntfy: {
    token: "ntfyToken",
  },
  bark: {
    device_key: "barkDeviceKey",
  },
  serverchan: {
    send_key: "serverChanSendKey",
  },
  feishu: {
    webhook_url: "feishuWebhookUrl",
    secret: "feishuSecret",
  },
  wecom: {
    webhook_url: "wecomWebhookUrl",
  },
  dingtalk: {
    webhook_url: "dingtalkWebhookUrl",
    secret: "dingtalkSecret",
  },
  pushdeer: {
    push_key: "pushdeerPushKey",
  },
  pushplus: {
    token: "pushplusToken",
  },
  pushover: {
    token: "pushoverToken",
    user: "pushoverUser",
  },
  napcat: {
    access_token: "napcatAccessToken",
  },
}

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

function parseWebhookHeadersDisplayWithConfiguredKeys(rawConfig: string, configuredHeaderKeys: Set<string>): string {
  try {
    const parsed = JSON.parse(rawConfig) as { headers?: unknown }
    if (!parsed.headers || typeof parsed.headers !== "object" || Array.isArray(parsed.headers)) {
      return ""
    }

    const normalized: Record<string, string> = {}
    for (const [key, value] of Object.entries(parsed.headers as Record<string, unknown>)) {
      if (typeof value === "string") {
        const trimmedValue = value.trim()
        if (trimmedValue !== "") {
          normalized[key] = value
          continue
        }
        if (configuredHeaderKeys.has(key)) {
          normalized[key] = CONFIGURED_SECRET_MASK
        }
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
    headers[trimmedKey] = isConfiguredSecretMask(value) ? "" : value
  }

  return headers
}

function isConfiguredSecretMask(value: string): boolean {
  return value.trim() === CONFIGURED_SECRET_MASK
}

function sanitizeSecretInput(value: string): string {
  if (isConfiguredSecretMask(value)) {
    return ""
  }
  return value
}

export function createInitialValues(channel: NotificationChannel | null): NotificationChannelFormValues {
  const initCfg = channel ? parseConfig(channel.config) : {}
  const configuredWebhookHeaderKeys = new Set(channel?.configured_webhook_header_keys ?? [])

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
    webhookHeaders: channel ? parseWebhookHeadersDisplayWithConfiguredKeys(channel.config, configuredWebhookHeaderKeys) : "",

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

export function createConfiguredSecretFormFields(channel: NotificationChannel | null): Set<keyof NotificationChannelFormValues> {
  if (!channel) {
    return new Set<keyof NotificationChannelFormValues>()
  }

  const channelType = channel.type
  const mapping = CHANNEL_SECRET_FIELD_TO_FORM_FIELD_MAP[channelType]
  if (!mapping) {
    return new Set<keyof NotificationChannelFormValues>()
  }

  const configuredSecretFields = channel.configured_secret_fields ?? []
  const configuredFormFields = new Set<keyof NotificationChannelFormValues>()
  for (const field of configuredSecretFields) {
    const mapped = mapping[field]
    if (mapped) {
      configuredFormFields.add(mapped)
    }
  }
  return configuredFormFields
}

export function buildConfig(type: ChannelType, values: NotificationChannelFormValues): string {
  switch (type) {
    case "smtp":
      return JSON.stringify({
        host: values.smtpHost.trim(),
        port: parseInt(values.smtpPort, 10) || 587,
        username: values.smtpUsername.trim(),
        password: sanitizeSecretInput(values.smtpPassword),
        from_email: values.smtpFromEmail.trim(),
        from_name: values.smtpFromName.trim(),
        to_email: values.smtpToEmail.trim(),
        encryption: values.smtpEncryption,
      })
    case "resend":
      return JSON.stringify({
        api_key: sanitizeSecretInput(values.apiKey).trim(),
        from_email: values.fromEmail.trim(),
        to_email: values.resendToEmail.trim(),
      })
    case "telegram":
      return JSON.stringify({ bot_token: sanitizeSecretInput(values.botToken).trim(), chat_id: values.chatId.trim() })
    case "webhook": {
      const webhookConfig: Record<string, unknown> = {
        url: values.webhookUrl.trim(),
        method: values.webhookMethod,
      }
      const webhookSecret = sanitizeSecretInput(values.webhookSecret).trim()
      if (webhookSecret) {
        webhookConfig.secret = webhookSecret
      }

      if (values.webhookHeaders.trim()) {
        const headers = parseWebhookHeadersInput(values.webhookHeaders)
        webhookConfig.headers = headers
      }

      return JSON.stringify(webhookConfig)
    }
    case "pushdeer":
      return JSON.stringify({
        push_key: sanitizeSecretInput(values.pushdeerPushKey).trim(),
        ...(values.pushdeerServerUrl.trim() ? { server_url: values.pushdeerServerUrl.trim() } : {}),
      })
    case "pushplus":
      return JSON.stringify({
        token: sanitizeSecretInput(values.pushplusToken).trim(),
        ...(values.pushplusTopic.trim() ? { topic: values.pushplusTopic.trim() } : {}),
        ...(values.pushplusEndpoint.trim() ? { endpoint: values.pushplusEndpoint.trim() } : {}),
        ...(values.pushplusTemplate.trim() ? { template: values.pushplusTemplate.trim() } : {}),
        ...(values.pushplusChannel.trim() ? { channel: values.pushplusChannel.trim() } : {}),
      })
    case "pushover":
      return JSON.stringify({
        token: sanitizeSecretInput(values.pushoverToken).trim(),
        user: sanitizeSecretInput(values.pushoverUser).trim(),
        ...(values.pushoverDevice.trim() ? { device: values.pushoverDevice.trim() } : {}),
        priority: parseInt(values.pushoverPriority, 10) || 0,
        ...(values.pushoverSound.trim() ? { sound: values.pushoverSound.trim() } : {}),
        ...(values.pushoverEndpoint.trim() ? { endpoint: values.pushoverEndpoint.trim() } : {}),
      })
    case "gotify":
      return JSON.stringify({ url: values.gotifyUrl.trim(), token: sanitizeSecretInput(values.gotifyToken).trim() })
    case "ntfy":
      return JSON.stringify({
        ...(values.ntfyUrl.trim() ? { url: values.ntfyUrl.trim() } : {}),
        topic: values.ntfyTopic.trim(),
        ...(sanitizeSecretInput(values.ntfyToken).trim() ? { token: sanitizeSecretInput(values.ntfyToken).trim() } : {}),
        ...(values.ntfyPriority.trim() ? { priority: values.ntfyPriority.trim() } : {}),
        ...(values.ntfyTags.trim() ? { tags: values.ntfyTags.trim() } : {}),
        ...(values.ntfyClick.trim() ? { click: values.ntfyClick.trim() } : {}),
        ...(values.ntfyIcon.trim() ? { icon: values.ntfyIcon.trim() } : {}),
      })
    case "bark":
      return JSON.stringify({
        ...(values.barkUrl.trim() ? { url: values.barkUrl.trim() } : {}),
        device_key: sanitizeSecretInput(values.barkDeviceKey).trim(),
      })
    case "serverchan":
      return JSON.stringify({ send_key: sanitizeSecretInput(values.serverChanSendKey).trim() })
    case "feishu":
      return JSON.stringify({
        webhook_url: sanitizeSecretInput(values.feishuWebhookUrl).trim(),
        ...(sanitizeSecretInput(values.feishuSecret).trim() ? { secret: sanitizeSecretInput(values.feishuSecret).trim() } : {}),
      })
    case "wecom":
      return JSON.stringify({ webhook_url: sanitizeSecretInput(values.wecomWebhookUrl).trim() })
    case "dingtalk":
      return JSON.stringify({
        webhook_url: sanitizeSecretInput(values.dingtalkWebhookUrl).trim(),
        ...(sanitizeSecretInput(values.dingtalkSecret).trim() ? { secret: sanitizeSecretInput(values.dingtalkSecret).trim() } : {}),
      })
    case "napcat":
      return JSON.stringify({
        url: values.napcatUrl.trim(),
        ...(sanitizeSecretInput(values.napcatAccessToken).trim() ? { access_token: sanitizeSecretInput(values.napcatAccessToken).trim() } : {}),
        message_type: values.napcatMessageType,
        ...(values.napcatMessageType === "private"
          ? { user_id: values.napcatUserId.trim() }
          : { group_id: values.napcatGroupId.trim() }),
      })
  }
}
