import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

import type { AdminSettingsSMTPAdvancedFieldsProps } from "./admin-settings-types"

export default function AdminSettingsSMTPAdvancedFields({
  onSMTPAuthMethodChange,
  onSMTPEncryptionChange,
  onSMTPHeloNameChange,
  onSMTPSkipTLSVerifyChange,
  onSMTPTimeoutSecondsChange,
  smtpAuthMethod,
  smtpEncryption,
  smtpHeloName,
  smtpSkipTLSVerify,
  smtpTimeoutSeconds,
}: AdminSettingsSMTPAdvancedFieldsProps) {
  const { t } = useTranslation()

  return (
    <details className="rounded-md border p-3">
      <summary className="cursor-pointer text-sm font-medium">
        {t("admin.settings.smtpAdvancedTitle")}
      </summary>
      <p className="mt-2 text-xs text-muted-foreground">{t("admin.settings.smtpAdvancedDescription")}</p>

      <div className="mt-3 space-y-3">
        <div className="space-y-1">
          <Label htmlFor="smtp-encryption">{t("admin.settings.smtpEncryption")}</Label>
          <Select value={smtpEncryption} onValueChange={onSMTPEncryptionChange}>
            <SelectTrigger id="smtp-encryption" className="w-64">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="starttls">{t("admin.settings.smtpEncryptionStartTLS")}</SelectItem>
              <SelectItem value="ssl_tls">{t("admin.settings.smtpEncryptionSSLTLS")}</SelectItem>
              <SelectItem value="none">{t("admin.settings.smtpEncryptionNone")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1">
          <Label htmlFor="smtp-auth-method">{t("admin.settings.smtpAuthMethod")}</Label>
          <Select value={smtpAuthMethod} onValueChange={onSMTPAuthMethodChange}>
            <SelectTrigger id="smtp-auth-method" className="w-64">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auto">{t("admin.settings.smtpAuthMethodAuto")}</SelectItem>
              <SelectItem value="plain">{t("admin.settings.smtpAuthMethodPlain")}</SelectItem>
              <SelectItem value="login">{t("admin.settings.smtpAuthMethodLogin")}</SelectItem>
              <SelectItem value="cram_md5">{t("admin.settings.smtpAuthMethodCramMD5")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1">
          <Label htmlFor="smtp-helo-name">{t("admin.settings.smtpHeloName")}</Label>
          <Input
            id="smtp-helo-name"
            value={smtpHeloName}
            onChange={(event) => onSMTPHeloNameChange(event.target.value)}
            placeholder={t("admin.settings.smtpHeloNamePlaceholder")}
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="smtp-timeout-seconds">{t("admin.settings.smtpTimeoutSeconds")}</Label>
          <div className="flex items-center gap-2">
            <Input
              id="smtp-timeout-seconds"
              type="number"
              min={1}
              max={600}
              step={1}
              className="w-32"
              value={smtpTimeoutSeconds}
              onChange={(event) => {
                const next = parseInt(event.target.value, 10)
                if (!Number.isNaN(next) && next >= 1) {
                  onSMTPTimeoutSecondsChange(next)
                }
              }}
            />
            <span className="text-sm text-muted-foreground">
              {t("admin.settings.smtpTimeoutSecondsUnit")}
            </span>
          </div>
        </div>

        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label htmlFor="smtp-skip-tls-verify">{t("admin.settings.smtpSkipTLSVerify")}</Label>
            <p className="text-sm text-muted-foreground">
              {t("admin.settings.smtpSkipTLSVerifyDescription")}
            </p>
          </div>
          <Switch
            id="smtp-skip-tls-verify"
            checked={smtpSkipTLSVerify}
            onCheckedChange={onSMTPSkipTLSVerifyChange}
          />
        </div>
      </div>
    </details>
  )
}
