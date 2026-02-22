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

export function SmtpConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <div className="space-y-3">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label htmlFor="smtp-host">{t("settings.notifications.channels.configFields.smtpHost")}</Label>
          <Input
            id="smtp-host"
            placeholder={t("settings.notifications.channels.configFields.smtpHostPlaceholder")}
            value={values.smtpHost}
            onChange={(e) => onValueChange("smtpHost", e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp-port">{t("settings.notifications.channels.configFields.smtpPort")}</Label>
          <Input
            id="smtp-port"
            type="number"
            placeholder="587"
            value={values.smtpPort}
            onChange={(e) => onValueChange("smtpPort", e.target.value)}
          />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label htmlFor="smtp-user">{t("settings.notifications.channels.configFields.smtpUsername")}</Label>
          <Input
            id="smtp-user"
            placeholder={t("settings.notifications.channels.configFields.smtpUsernamePlaceholder")}
            value={values.smtpUsername}
            onChange={(e) => onValueChange("smtpUsername", e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp-pass">{t("settings.notifications.channels.configFields.smtpPassword")}</Label>
          <Input
            id="smtp-pass"
            type="password"
            placeholder={t("settings.notifications.channels.configFields.smtpPasswordPlaceholder")}
            value={values.smtpPassword}
            onChange={(e) => onValueChange("smtpPassword", e.target.value)}
          />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-2">
          <Label htmlFor="smtp-from">{t("settings.notifications.channels.configFields.smtpFromEmail")}</Label>
          <Input
            id="smtp-from"
            type="email"
            placeholder={t("settings.notifications.channels.configFields.smtpFromEmailPlaceholder")}
            value={values.smtpFromEmail}
            onChange={(e) => onValueChange("smtpFromEmail", e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="smtp-from-name">{t("settings.notifications.channels.configFields.smtpFromName")}</Label>
          <Input
            id="smtp-from-name"
            placeholder={t("settings.notifications.channels.configFields.smtpFromNamePlaceholder")}
            value={values.smtpFromName}
            onChange={(e) => onValueChange("smtpFromName", e.target.value)}
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label htmlFor="smtp-to">{t("settings.notifications.channels.configFields.toEmail")}</Label>
        <Input
          id="smtp-to"
          type="email"
          placeholder={t("settings.notifications.channels.configFields.toEmailPlaceholder")}
          value={values.smtpToEmail}
          onChange={(e) => onValueChange("smtpToEmail", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label>{t("settings.notifications.channels.configFields.smtpEncryption")}</Label>
        <Select value={values.smtpEncryption} onValueChange={(value) => onValueChange("smtpEncryption", value)}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="starttls">{t("settings.notifications.channels.configFields.smtpEncryptionStartTLS")}</SelectItem>
            <SelectItem value="ssl_tls">{t("settings.notifications.channels.configFields.smtpEncryptionSSLTLS")}</SelectItem>
            <SelectItem value="none">{t("settings.notifications.channels.configFields.smtpEncryptionNone")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

export function ResendConfigFields({ onValueChange, t, values }: BaseChannelConfigFieldProps) {
  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="resend-key">{t("settings.notifications.channels.configFields.apiKey")}</Label>
        <Input
          id="resend-key"
          placeholder={t("settings.notifications.channels.configFields.apiKeyPlaceholder")}
          value={values.apiKey}
          onChange={(e) => onValueChange("apiKey", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="resend-from">{t("settings.notifications.channels.configFields.fromEmail")}</Label>
        <Input
          id="resend-from"
          type="email"
          placeholder={t("settings.notifications.channels.configFields.fromEmailPlaceholder")}
          value={values.fromEmail}
          onChange={(e) => onValueChange("fromEmail", e.target.value)}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="resend-to">{t("settings.notifications.channels.configFields.toEmail")}</Label>
        <Input
          id="resend-to"
          type="email"
          placeholder={t("settings.notifications.channels.configFields.toEmailPlaceholder")}
          value={values.resendToEmail}
          onChange={(e) => onValueChange("resendToEmail", e.target.value)}
          required
        />
      </div>
    </>
  )
}
