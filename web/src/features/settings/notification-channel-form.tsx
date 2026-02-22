import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import type { NotificationChannel } from "@/types"

interface Props {
  channel: NotificationChannel | null
  onClose: () => void
  onSave: (type: string, config: string) => void | Promise<void>
  open: boolean
  saving: boolean
}

type ChannelType = "smtp" | "resend" | "telegram" | "webhook" | "gotify" | "ntfy" | "bark" | "serverchan" | "feishu" | "wecom" | "dingtalk" | "pushdeer" | "pushplus" | "pushover" | "napcat"
type WebhookMethod = "GET" | "POST" | "PUT"

const WEBHOOK_HEADERS_PARSE_ERROR = "WEBHOOK_HEADERS_PARSE_ERROR"
const PUSHOVER_SOUND_DEVICE_DEFAULT = "__device_default__"
const PUSHOVER_SOUND_OPTIONS: Array<{ value: string; i18nKey: string }> = [
  { value: PUSHOVER_SOUND_DEVICE_DEFAULT, i18nKey: "pushoverSoundOptionDeviceDefault" },
  { value: "pushover", i18nKey: "pushoverSoundOptionPushover" },
  { value: "vibrate", i18nKey: "pushoverSoundOptionVibrate" },
  { value: "none", i18nKey: "pushoverSoundOptionNone" },
  { value: "bike", i18nKey: "pushoverSoundOptionBike" },
  { value: "bugle", i18nKey: "pushoverSoundOptionBugle" },
  { value: "cashregister", i18nKey: "pushoverSoundOptionCashregister" },
  { value: "classical", i18nKey: "pushoverSoundOptionClassical" },
  { value: "cosmic", i18nKey: "pushoverSoundOptionCosmic" },
  { value: "falling", i18nKey: "pushoverSoundOptionFalling" },
  { value: "gamelan", i18nKey: "pushoverSoundOptionGamelan" },
  { value: "incoming", i18nKey: "pushoverSoundOptionIncoming" },
  { value: "intermission", i18nKey: "pushoverSoundOptionIntermission" },
  { value: "magic", i18nKey: "pushoverSoundOptionMagic" },
  { value: "mechanical", i18nKey: "pushoverSoundOptionMechanical" },
  { value: "pianobar", i18nKey: "pushoverSoundOptionPianobar" },
  { value: "siren", i18nKey: "pushoverSoundOptionSiren" },
  { value: "spacealarm", i18nKey: "pushoverSoundOptionSpacealarm" },
  { value: "tugboat", i18nKey: "pushoverSoundOptionTugboat" },
  { value: "alien", i18nKey: "pushoverSoundOptionAlien" },
  { value: "climb", i18nKey: "pushoverSoundOptionClimb" },
  { value: "persistent", i18nKey: "pushoverSoundOptionPersistent" },
  { value: "echo", i18nKey: "pushoverSoundOptionEcho" },
  { value: "updown", i18nKey: "pushoverSoundOptionUpdown" },
]

function parseConfig(raw: string): Record<string, string> {
  try {
    return JSON.parse(raw) as Record<string, string>
  } catch {
    return {}
  }
}

function parseWebhookMethod(raw: string): WebhookMethod {
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

export function NotificationChannelForm({ channel, onClose, onSave, open, saving }: Props) {
  const { t } = useTranslation()
  const isEditing = !!channel

  const initCfg = channel ? parseConfig(channel.config) : {}
  const [type, setType] = useState<ChannelType>(channel?.type ?? "smtp")

  const [smtpHost, setSmtpHost] = useState(initCfg.host ?? "")
  const [smtpPort, setSmtpPort] = useState(initCfg.port ?? "587")
  const [smtpUsername, setSmtpUsername] = useState(initCfg.username ?? "")
  const [smtpPassword, setSmtpPassword] = useState(initCfg.password ?? "")
  const [smtpFromEmail, setSmtpFromEmail] = useState(initCfg.from_email ?? "")
  const [smtpFromName, setSmtpFromName] = useState(initCfg.from_name ?? "")
  const [smtpToEmail, setSmtpToEmail] = useState(initCfg.to_email ?? "")
  const [smtpEncryption, setSmtpEncryption] = useState(initCfg.encryption ?? "starttls")

  const [apiKey, setApiKey] = useState(initCfg.api_key ?? "")
  const [fromEmail, setFromEmail] = useState(initCfg.from_email ?? "")
  const [resendToEmail, setResendToEmail] = useState(initCfg.to_email ?? "")
  const [botToken, setBotToken] = useState(initCfg.bot_token ?? "")
  const [chatId, setChatId] = useState(initCfg.chat_id ?? "")
  const [webhookUrl, setWebhookUrl] = useState(initCfg.url ?? "")
  const [webhookSecret, setWebhookSecret] = useState(initCfg.secret ?? "")
  const [webhookMethod, setWebhookMethod] = useState<WebhookMethod>(parseWebhookMethod(initCfg.method ?? ""))
  const [webhookHeaders, setWebhookHeaders] = useState(channel ? parseWebhookHeadersDisplay(channel.config) : "")

  const [pushdeerPushKey, setPushdeerPushKey] = useState(initCfg.push_key ?? "")
  const [pushdeerServerUrl, setPushdeerServerUrl] = useState(initCfg.server_url ?? "")

  const [pushplusToken, setPushplusToken] = useState(initCfg.token ?? "")
  const [pushplusTopic, setPushplusTopic] = useState(initCfg.topic ?? "")
  const [pushplusEndpoint, setPushplusEndpoint] = useState(initCfg.endpoint ?? "")
  const [pushplusTemplate, setPushplusTemplate] = useState(initCfg.template ?? "markdown")
  const [pushplusChannel, setPushplusChannel] = useState(initCfg.channel ?? "")
  const [pushoverToken, setPushoverToken] = useState(initCfg.token ?? "")
  const [pushoverUser, setPushoverUser] = useState(initCfg.user ?? "")
  const [pushoverDevice, setPushoverDevice] = useState(initCfg.device ?? "")
  const [pushoverPriority, setPushoverPriority] = useState(initCfg.priority != null ? String(initCfg.priority) : "0")
  const [pushoverSound, setPushoverSound] = useState(initCfg.sound ?? "")
  const [pushoverEndpoint, setPushoverEndpoint] = useState(initCfg.endpoint ?? "")

  const [gotifyUrl, setGotifyUrl] = useState(initCfg.url ?? "")
  const [gotifyToken, setGotifyToken] = useState(initCfg.token ?? "")

  const [ntfyUrl, setNtfyUrl] = useState(initCfg.url ?? "")
  const [ntfyTopic, setNtfyTopic] = useState(initCfg.topic ?? "")
  const [ntfyToken, setNtfyToken] = useState(initCfg.token ?? "")

  const [barkUrl, setBarkUrl] = useState(initCfg.url ?? "")
  const [barkDeviceKey, setBarkDeviceKey] = useState(initCfg.device_key ?? "")

  const [serverChanSendKey, setServerChanSendKey] = useState(initCfg.send_key ?? "")

  const [feishuWebhookUrl, setFeishuWebhookUrl] = useState(initCfg.webhook_url ?? "")
  const [feishuSecret, setFeishuSecret] = useState(initCfg.secret ?? "")

  const [wecomWebhookUrl, setWecomWebhookUrl] = useState(initCfg.webhook_url ?? "")

  const [dingtalkWebhookUrl, setDingtalkWebhookUrl] = useState(initCfg.webhook_url ?? "")
  const [dingtalkSecret, setDingtalkSecret] = useState(initCfg.secret ?? "")

  const [napcatUrl, setNapcatUrl] = useState(initCfg.url ?? "")
  const [napcatAccessToken, setNapcatAccessToken] = useState(initCfg.access_token ?? "")
  const [napcatMessageType, setNapcatMessageType] = useState<"private" | "group">((initCfg.message_type as "private" | "group") ?? "private")
  const [napcatUserId, setNapcatUserId] = useState(initCfg.user_id ?? "")
  const [napcatGroupId, setNapcatGroupId] = useState(initCfg.group_id ?? "")

  function buildConfig(): string {
    switch (type) {
      case "smtp":
        return JSON.stringify({
          host: smtpHost.trim(),
          port: parseInt(smtpPort, 10) || 587,
          username: smtpUsername.trim(),
          password: smtpPassword,
          from_email: smtpFromEmail.trim(),
          from_name: smtpFromName.trim(),
          to_email: smtpToEmail.trim(),
          encryption: smtpEncryption,
        })
      case "resend":
        return JSON.stringify({ api_key: apiKey.trim(), from_email: fromEmail.trim(), to_email: resendToEmail.trim() })
      case "telegram":
        return JSON.stringify({ bot_token: botToken.trim(), chat_id: chatId.trim() })
      case "webhook":
        {
          const webhookConfig: Record<string, unknown> = {
            url: webhookUrl.trim(),
            method: webhookMethod,
          }
          if (webhookSecret.trim()) {
            webhookConfig.secret = webhookSecret.trim()
          }

          if (webhookHeaders.trim()) {
            const headers = parseWebhookHeadersInput(webhookHeaders)
            if (Object.keys(headers).length > 0) {
              webhookConfig.headers = headers
            }
          }

          return JSON.stringify(webhookConfig)
        }
      case "pushdeer":
        return JSON.stringify({
          push_key: pushdeerPushKey.trim(),
          ...(pushdeerServerUrl.trim() ? { server_url: pushdeerServerUrl.trim() } : {}),
        })
      case "pushplus":
        return JSON.stringify({
          token: pushplusToken.trim(),
          ...(pushplusTopic.trim() ? { topic: pushplusTopic.trim() } : {}),
          ...(pushplusEndpoint.trim() ? { endpoint: pushplusEndpoint.trim() } : {}),
          ...(pushplusTemplate.trim() ? { template: pushplusTemplate.trim() } : {}),
          ...(pushplusChannel.trim() ? { channel: pushplusChannel.trim() } : {}),
        })
      case "pushover":
        return JSON.stringify({
          token: pushoverToken.trim(),
          user: pushoverUser.trim(),
          ...(pushoverDevice.trim() ? { device: pushoverDevice.trim() } : {}),
          priority: parseInt(pushoverPriority, 10) || 0,
          ...(pushoverSound.trim() ? { sound: pushoverSound.trim() } : {}),
          ...(pushoverEndpoint.trim() ? { endpoint: pushoverEndpoint.trim() } : {}),
        })
      case "gotify":
        return JSON.stringify({ url: gotifyUrl.trim(), token: gotifyToken.trim() })
      case "ntfy":
        return JSON.stringify({
          ...(ntfyUrl.trim() ? { url: ntfyUrl.trim() } : {}),
          topic: ntfyTopic.trim(),
          ...(ntfyToken.trim() ? { token: ntfyToken.trim() } : {}),
        })
      case "bark":
        return JSON.stringify({
          ...(barkUrl.trim() ? { url: barkUrl.trim() } : {}),
          device_key: barkDeviceKey.trim(),
        })
      case "serverchan":
        return JSON.stringify({ send_key: serverChanSendKey.trim() })
      case "feishu":
        return JSON.stringify({
          webhook_url: feishuWebhookUrl.trim(),
          ...(feishuSecret.trim() ? { secret: feishuSecret.trim() } : {}),
        })
      case "wecom":
        return JSON.stringify({ webhook_url: wecomWebhookUrl.trim() })
      case "dingtalk":
        return JSON.stringify({
          webhook_url: dingtalkWebhookUrl.trim(),
          ...(dingtalkSecret.trim() ? { secret: dingtalkSecret.trim() } : {}),
        })
      case "napcat":
        return JSON.stringify({
          url: napcatUrl.trim(),
          ...(napcatAccessToken.trim() ? { access_token: napcatAccessToken.trim() } : {}),
          message_type: napcatMessageType,
          ...(napcatMessageType === "private" ? { user_id: napcatUserId.trim() } : { group_id: napcatGroupId.trim() }),
        })
    }
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    try {
      const config = buildConfig()
      void onSave(type, config)
    } catch (error) {
      if (error instanceof Error && error.message === WEBHOOK_HEADERS_PARSE_ERROR) {
        window.alert(t("settings.notifications.channels.configFields.headersInvalid"))
        return
      }
      window.alert(t("settings.notifications.channels.configFields.headersInvalid"))
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditing ? t("settings.notifications.channels.edit") : t("settings.notifications.channels.addButton")}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label>{t("settings.notifications.channels.typeLabel")}</Label>
            <Select value={type} onValueChange={(v) => setType(v as ChannelType)} disabled={isEditing}>
              <SelectTrigger><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="smtp">{t("settings.notifications.channels.type.smtp")}</SelectItem>
                <SelectItem value="resend">{t("settings.notifications.channels.type.resend")}</SelectItem>
                <SelectItem value="telegram">{t("settings.notifications.channels.type.telegram")}</SelectItem>
                <SelectItem value="webhook">{t("settings.notifications.channels.type.webhook")}</SelectItem>
                <SelectItem value="pushdeer">{t("settings.notifications.channels.type.pushdeer")}</SelectItem>
                <SelectItem value="pushplus">{t("settings.notifications.channels.type.pushplus")}</SelectItem>
                <SelectItem value="pushover">{t("settings.notifications.channels.type.pushover")}</SelectItem>
                <SelectItem value="gotify">{t("settings.notifications.channels.type.gotify")}</SelectItem>
                <SelectItem value="ntfy">{t("settings.notifications.channels.type.ntfy")}</SelectItem>
                <SelectItem value="bark">{t("settings.notifications.channels.type.bark")}</SelectItem>
                <SelectItem value="serverchan">{t("settings.notifications.channels.type.serverchan")}</SelectItem>
                <SelectItem value="feishu">{t("settings.notifications.channels.type.feishu")}</SelectItem>
                <SelectItem value="wecom">{t("settings.notifications.channels.type.wecom")}</SelectItem>
                <SelectItem value="dingtalk">{t("settings.notifications.channels.type.dingtalk")}</SelectItem>
                <SelectItem value="napcat">{t("settings.notifications.channels.type.napcat")}</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {type === "smtp" && (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="smtp-host">{t("settings.notifications.channels.configFields.smtpHost")}</Label>
                  <Input id="smtp-host" placeholder={t("settings.notifications.channels.configFields.smtpHostPlaceholder")} value={smtpHost} onChange={(e) => setSmtpHost(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-port">{t("settings.notifications.channels.configFields.smtpPort")}</Label>
                  <Input id="smtp-port" type="number" placeholder="587" value={smtpPort} onChange={(e) => setSmtpPort(e.target.value)} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="smtp-user">{t("settings.notifications.channels.configFields.smtpUsername")}</Label>
                  <Input id="smtp-user" placeholder={t("settings.notifications.channels.configFields.smtpUsernamePlaceholder")} value={smtpUsername} onChange={(e) => setSmtpUsername(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-pass">{t("settings.notifications.channels.configFields.smtpPassword")}</Label>
                  <Input id="smtp-pass" type="password" placeholder={t("settings.notifications.channels.configFields.smtpPasswordPlaceholder")} value={smtpPassword} onChange={(e) => setSmtpPassword(e.target.value)} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="smtp-from">{t("settings.notifications.channels.configFields.smtpFromEmail")}</Label>
                  <Input id="smtp-from" type="email" placeholder={t("settings.notifications.channels.configFields.smtpFromEmailPlaceholder")} value={smtpFromEmail} onChange={(e) => setSmtpFromEmail(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="smtp-from-name">{t("settings.notifications.channels.configFields.smtpFromName")}</Label>
                  <Input id="smtp-from-name" placeholder={t("settings.notifications.channels.configFields.smtpFromNamePlaceholder")} value={smtpFromName} onChange={(e) => setSmtpFromName(e.target.value)} />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="smtp-to">{t("settings.notifications.channels.configFields.toEmail")}</Label>
                <Input id="smtp-to" type="email" placeholder={t("settings.notifications.channels.configFields.toEmailPlaceholder")} value={smtpToEmail} onChange={(e) => setSmtpToEmail(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.notifications.channels.configFields.smtpEncryption")}</Label>
                <Select value={smtpEncryption} onValueChange={setSmtpEncryption}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="starttls">{t("settings.notifications.channels.configFields.smtpEncryptionStartTLS")}</SelectItem>
                    <SelectItem value="ssl_tls">{t("settings.notifications.channels.configFields.smtpEncryptionSSLTLS")}</SelectItem>
                    <SelectItem value="none">{t("settings.notifications.channels.configFields.smtpEncryptionNone")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}

          {type === "resend" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="resend-key">{t("settings.notifications.channels.configFields.apiKey")}</Label>
                <Input id="resend-key" placeholder={t("settings.notifications.channels.configFields.apiKeyPlaceholder")} value={apiKey} onChange={(e) => setApiKey(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="resend-from">{t("settings.notifications.channels.configFields.fromEmail")}</Label>
                <Input id="resend-from" type="email" placeholder={t("settings.notifications.channels.configFields.fromEmailPlaceholder")} value={fromEmail} onChange={(e) => setFromEmail(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="resend-to">{t("settings.notifications.channels.configFields.toEmail")}</Label>
                <Input id="resend-to" type="email" placeholder={t("settings.notifications.channels.configFields.toEmailPlaceholder")} value={resendToEmail} onChange={(e) => setResendToEmail(e.target.value)} required />
              </div>
            </>
          )}

          {type === "telegram" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="tg-token">{t("settings.notifications.channels.configFields.botToken")}</Label>
                <Input id="tg-token" placeholder={t("settings.notifications.channels.configFields.botTokenPlaceholder")} value={botToken} onChange={(e) => setBotToken(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tg-chat">{t("settings.notifications.channels.configFields.chatId")}</Label>
                <Input id="tg-chat" placeholder={t("settings.notifications.channels.configFields.chatIdPlaceholder")} value={chatId} onChange={(e) => setChatId(e.target.value)} required />
              </div>
            </>
          )}

          {type === "webhook" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="wh-url">{t("settings.notifications.channels.configFields.url")}</Label>
                <Input id="wh-url" type="url" placeholder={t("settings.notifications.channels.configFields.urlPlaceholder")} value={webhookUrl} onChange={(e) => setWebhookUrl(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.notifications.channels.configFields.method")}</Label>
                <Select value={webhookMethod} onValueChange={(value) => setWebhookMethod(parseWebhookMethod(value))}>
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
                <Input id="wh-secret" placeholder={t("settings.notifications.channels.configFields.secretPlaceholder")} value={webhookSecret} onChange={(e) => setWebhookSecret(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="wh-headers">{t("settings.notifications.channels.configFields.headers")}</Label>
                <textarea
                  id="wh-headers"
                  className="flex min-h-[96px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm font-mono shadow-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50"
                  placeholder={t("settings.notifications.channels.configFields.headersPlaceholder")}
                  value={webhookHeaders}
                  onChange={(e) => setWebhookHeaders(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">{t("settings.notifications.channels.configFields.headersHint")}</p>
              </div>
            </>
          )}

          {type === "pushdeer" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="pd-key">{t("settings.notifications.channels.configFields.pushdeerPushKey")}</Label>
                <Input id="pd-key" placeholder={t("settings.notifications.channels.configFields.pushdeerPushKeyPlaceholder")} value={pushdeerPushKey} onChange={(e) => setPushdeerPushKey(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pd-server-url">{t("settings.notifications.channels.configFields.pushdeerServerUrl")}</Label>
                <Input id="pd-server-url" type="url" placeholder={t("settings.notifications.channels.configFields.pushdeerServerUrlPlaceholder")} value={pushdeerServerUrl} onChange={(e) => setPushdeerServerUrl(e.target.value)} />
              </div>
            </>
          )}

          {type === "pushplus" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="pp-token">{t("settings.notifications.channels.configFields.pushplusToken")}</Label>
                <Input id="pp-token" placeholder={t("settings.notifications.channels.configFields.pushplusTokenPlaceholder")} value={pushplusToken} onChange={(e) => setPushplusToken(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pp-topic">{t("settings.notifications.channels.configFields.pushplusTopic")}</Label>
                <Input id="pp-topic" placeholder={t("settings.notifications.channels.configFields.pushplusTopicPlaceholder")} value={pushplusTopic} onChange={(e) => setPushplusTopic(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="pp-endpoint">{t("settings.notifications.channels.configFields.pushplusEndpoint")}</Label>
                <Input id="pp-endpoint" type="url" placeholder={t("settings.notifications.channels.configFields.pushplusEndpointPlaceholder")} value={pushplusEndpoint} onChange={(e) => setPushplusEndpoint(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.notifications.channels.configFields.pushplusTemplate")}</Label>
                <Select value={pushplusTemplate} onValueChange={setPushplusTemplate}>
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
                <Input id="pp-channel" placeholder={t("settings.notifications.channels.configFields.pushplusChannelPlaceholder")} value={pushplusChannel} onChange={(e) => setPushplusChannel(e.target.value)} />
              </div>
            </>
          )}

          {type === "pushover" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="po-token">{t("settings.notifications.channels.configFields.pushoverToken")}</Label>
                <Input id="po-token" placeholder={t("settings.notifications.channels.configFields.pushoverTokenPlaceholder")} value={pushoverToken} onChange={(e) => setPushoverToken(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="po-user">{t("settings.notifications.channels.configFields.pushoverUser")}</Label>
                <Input id="po-user" placeholder={t("settings.notifications.channels.configFields.pushoverUserPlaceholder")} value={pushoverUser} onChange={(e) => setPushoverUser(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="po-device">{t("settings.notifications.channels.configFields.pushoverDevice")}</Label>
                <Input id="po-device" placeholder={t("settings.notifications.channels.configFields.pushoverDevicePlaceholder")} value={pushoverDevice} onChange={(e) => setPushoverDevice(e.target.value)} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label>{t("settings.notifications.channels.configFields.pushoverPriority")}</Label>
                  <Select value={pushoverPriority} onValueChange={setPushoverPriority}>
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
                    value={pushoverSound || PUSHOVER_SOUND_DEVICE_DEFAULT}
                    onValueChange={(value) => setPushoverSound(value === PUSHOVER_SOUND_DEVICE_DEFAULT ? "" : value)}
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
                <Input id="po-endpoint" type="url" placeholder={t("settings.notifications.channels.configFields.pushoverEndpointPlaceholder")} value={pushoverEndpoint} onChange={(e) => setPushoverEndpoint(e.target.value)} />
              </div>
            </>
          )}

          {type === "gotify" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="gotify-url">{t("settings.notifications.channels.configFields.gotifyUrl")}</Label>
                <Input id="gotify-url" type="url" placeholder={t("settings.notifications.channels.configFields.gotifyUrlPlaceholder")} value={gotifyUrl} onChange={(e) => setGotifyUrl(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="gotify-token">{t("settings.notifications.channels.configFields.gotifyToken")}</Label>
                <Input id="gotify-token" placeholder={t("settings.notifications.channels.configFields.gotifyTokenPlaceholder")} value={gotifyToken} onChange={(e) => setGotifyToken(e.target.value)} required />
              </div>
            </>
          )}

          {type === "ntfy" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="ntfy-url">{t("settings.notifications.channels.configFields.ntfyUrl")}</Label>
                <Input id="ntfy-url" type="url" placeholder={t("settings.notifications.channels.configFields.ntfyUrlPlaceholder")} value={ntfyUrl} onChange={(e) => setNtfyUrl(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ntfy-topic">{t("settings.notifications.channels.configFields.ntfyTopic")}</Label>
                <Input id="ntfy-topic" placeholder={t("settings.notifications.channels.configFields.ntfyTopicPlaceholder")} value={ntfyTopic} onChange={(e) => setNtfyTopic(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ntfy-token">{t("settings.notifications.channels.configFields.ntfyToken")}</Label>
                <Input id="ntfy-token" placeholder={t("settings.notifications.channels.configFields.ntfyTokenPlaceholder")} value={ntfyToken} onChange={(e) => setNtfyToken(e.target.value)} />
              </div>
            </>
          )}

          {type === "bark" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="bark-url">{t("settings.notifications.channels.configFields.barkUrl")}</Label>
                <Input id="bark-url" type="url" placeholder={t("settings.notifications.channels.configFields.barkUrlPlaceholder")} value={barkUrl} onChange={(e) => setBarkUrl(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="bark-key">{t("settings.notifications.channels.configFields.barkDeviceKey")}</Label>
                <Input id="bark-key" placeholder={t("settings.notifications.channels.configFields.barkDeviceKeyPlaceholder")} value={barkDeviceKey} onChange={(e) => setBarkDeviceKey(e.target.value)} required />
              </div>
            </>
          )}

          {type === "serverchan" && (
            <div className="space-y-2">
              <Label htmlFor="sc-key">{t("settings.notifications.channels.configFields.serverChanSendKey")}</Label>
              <Input id="sc-key" placeholder={t("settings.notifications.channels.configFields.serverChanSendKeyPlaceholder")} value={serverChanSendKey} onChange={(e) => setServerChanSendKey(e.target.value)} required />
            </div>
          )}

          {type === "feishu" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="feishu-url">{t("settings.notifications.channels.configFields.feishuWebhookUrl")}</Label>
                <Input id="feishu-url" type="url" placeholder={t("settings.notifications.channels.configFields.feishuWebhookUrlPlaceholder")} value={feishuWebhookUrl} onChange={(e) => setFeishuWebhookUrl(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="feishu-secret">{t("settings.notifications.channels.configFields.feishuSecret")}</Label>
                <Input id="feishu-secret" placeholder={t("settings.notifications.channels.configFields.feishuSecretPlaceholder")} value={feishuSecret} onChange={(e) => setFeishuSecret(e.target.value)} />
              </div>
            </>
          )}

          {type === "wecom" && (
            <div className="space-y-2">
              <Label htmlFor="wecom-url">{t("settings.notifications.channels.configFields.wecomWebhookUrl")}</Label>
              <Input id="wecom-url" type="url" placeholder={t("settings.notifications.channels.configFields.wecomWebhookUrlPlaceholder")} value={wecomWebhookUrl} onChange={(e) => setWecomWebhookUrl(e.target.value)} required />
            </div>
          )}

          {type === "dingtalk" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="dt-url">{t("settings.notifications.channels.configFields.dingtalkWebhookUrl")}</Label>
                <Input id="dt-url" type="url" placeholder={t("settings.notifications.channels.configFields.dingtalkWebhookUrlPlaceholder")} value={dingtalkWebhookUrl} onChange={(e) => setDingtalkWebhookUrl(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="dt-secret">{t("settings.notifications.channels.configFields.dingtalkSecret")}</Label>
                <Input id="dt-secret" placeholder={t("settings.notifications.channels.configFields.dingtalkSecretPlaceholder")} value={dingtalkSecret} onChange={(e) => setDingtalkSecret(e.target.value)} />
              </div>
            </>
          )}

          {type === "napcat" && (
            <>
              <div className="space-y-2">
                <Label htmlFor="nc-url">{t("settings.notifications.channels.configFields.napcatUrl")}</Label>
                <Input id="nc-url" type="url" placeholder={t("settings.notifications.channels.configFields.napcatUrlPlaceholder")} value={napcatUrl} onChange={(e) => setNapcatUrl(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="nc-token">{t("settings.notifications.channels.configFields.napcatAccessToken")}</Label>
                <Input id="nc-token" placeholder={t("settings.notifications.channels.configFields.napcatAccessTokenPlaceholder")} value={napcatAccessToken} onChange={(e) => setNapcatAccessToken(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>{t("settings.notifications.channels.configFields.napcatMessageType")}</Label>
                <Select value={napcatMessageType} onValueChange={(v) => setNapcatMessageType(v as "private" | "group")}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="private">{t("settings.notifications.channels.configFields.napcatMessageTypePrivate")}</SelectItem>
                    <SelectItem value="group">{t("settings.notifications.channels.configFields.napcatMessageTypeGroup")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {napcatMessageType === "private" && (
                <div className="space-y-2">
                  <Label htmlFor="nc-userid">{t("settings.notifications.channels.configFields.napcatUserId")}</Label>
                  <Input id="nc-userid" placeholder={t("settings.notifications.channels.configFields.napcatUserIdPlaceholder")} value={napcatUserId} onChange={(e) => setNapcatUserId(e.target.value)} required />
                </div>
              )}
              {napcatMessageType === "group" && (
                <div className="space-y-2">
                  <Label htmlFor="nc-groupid">{t("settings.notifications.channels.configFields.napcatGroupId")}</Label>
                  <Input id="nc-groupid" placeholder={t("settings.notifications.channels.configFields.napcatGroupIdPlaceholder")} value={napcatGroupId} onChange={(e) => setNapcatGroupId(e.target.value)} required />
                </div>
              )}
            </>
          )}

          <div className="flex gap-2 pt-2">
            <Button type="button" variant="outline" className="flex-1" onClick={onClose}>
              {t("settings.notifications.channels.cancel")}
            </Button>
            <Button type="submit" className="flex-1" disabled={saving}>
              {saving ? t("settings.notifications.channels.adding") : t("settings.notifications.channels.save")}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
