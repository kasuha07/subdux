import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

import AdminSettingsSMTPAdvancedFields from "./admin-settings-smtp-advanced-fields"
import type { AdminSettingsSMTPSectionProps } from "./admin-settings-types"

export default function AdminSettingsSMTPSection({
  onSMTPAuthMethodChange,
  onSMTPEnabledChange,
  onSMTPEncryptionChange,
  onSMTPFromEmailChange,
  onSMTPFromNameChange,
  onSMTPHeloNameChange,
  onSMTPHostChange,
  onSMTPPasswordChange,
  onSMTPSkipTLSVerifyChange,
  onSMTPPortChange,
  onSMTPTestRecipientChange,
  onSMTPTest,
  onSMTPTimeoutSecondsChange,
  onSMTPUsernameChange,
  smtpAuthMethod,
  smtpEnabled,
  smtpEncryption,
  smtpFromEmail,
  smtpFromName,
  smtpHeloName,
  smtpHost,
  smtpPassword,
  smtpPasswordConfigured,
  smtpPort,
  smtpSkipTLSVerify,
  smtpTestRecipient,
  smtpTesting,
  smtpTimeoutSeconds,
  smtpUsername,
}: AdminSettingsSMTPSectionProps) {
  const { t } = useTranslation()
  const [editingSMTPPassword, setEditingSMTPPassword] = useState(false)
  const configuredMaskValue = "••••••••"
  const smtpPasswordDisplayValue = editingSMTPPassword
    ? smtpPassword
    : smtpPassword || (smtpPasswordConfigured ? configuredMaskValue : "")

  return (
    <>
      <div className="space-y-2">
        <h3 className="text-sm font-medium">{t("admin.settings.smtpTitle")}</h3>
        <p className="text-xs text-muted-foreground">{t("admin.settings.smtpDescription")}</p>
      </div>

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="smtp-enabled">{t("admin.settings.smtpEnabled")}</Label>
          <p className="text-sm text-muted-foreground">{t("admin.settings.smtpEnabledDescription")}</p>
        </div>
        <Switch id="smtp-enabled" checked={smtpEnabled} onCheckedChange={onSMTPEnabledChange} />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-test-recipient">{t("admin.settings.smtpTestRecipient")}</Label>
        <div className="flex items-center gap-2">
          <Input
            id="smtp-test-recipient"
            type="email"
            className="flex-1"
            value={smtpTestRecipient}
            onChange={(event) => onSMTPTestRecipientChange(event.target.value)}
            placeholder={t("admin.settings.smtpTestRecipientPlaceholder")}
          />
          <Button
            variant="outline"
            onClick={() => void onSMTPTest()}
            disabled={smtpTesting || !smtpEnabled}
          >
            {smtpTesting ? t("admin.settings.smtpTesting") : t("admin.settings.smtpTestButton")}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">{t("admin.settings.smtpTestDescription")}</p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-host">{t("admin.settings.smtpHost")}</Label>
        <Input
          id="smtp-host"
          value={smtpHost}
          onChange={(event) => onSMTPHostChange(event.target.value)}
          placeholder={t("admin.settings.smtpHostPlaceholder")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-port">{t("admin.settings.smtpPort")}</Label>
        <Input
          id="smtp-port"
          type="number"
          min={1}
          max={65535}
          step={1}
          className="w-32"
          value={smtpPort}
          onChange={(event) => {
            const next = parseInt(event.target.value, 10)
            if (!Number.isNaN(next) && next >= 1) {
              onSMTPPortChange(next)
            }
          }}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-username">{t("admin.settings.smtpUsername")}</Label>
        <Input
          id="smtp-username"
          value={smtpUsername}
          onChange={(event) => onSMTPUsernameChange(event.target.value)}
          placeholder={t("admin.settings.smtpUsernamePlaceholder")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-password">{t("admin.settings.smtpPassword")}</Label>
        <Input
          id="smtp-password"
          type="password"
          value={smtpPasswordDisplayValue}
          onFocus={() => setEditingSMTPPassword(true)}
          onBlur={() => setEditingSMTPPassword(false)}
          onChange={(event) => onSMTPPasswordChange(event.target.value)}
          placeholder={t("admin.settings.smtpPasswordPlaceholder")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-from-email">{t("admin.settings.smtpFromEmail")}</Label>
        <Input
          id="smtp-from-email"
          type="email"
          value={smtpFromEmail}
          onChange={(event) => onSMTPFromEmailChange(event.target.value)}
          placeholder={t("admin.settings.smtpFromEmailPlaceholder")}
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="smtp-from-name">{t("admin.settings.smtpFromName")}</Label>
        <Input
          id="smtp-from-name"
          value={smtpFromName}
          onChange={(event) => onSMTPFromNameChange(event.target.value)}
          placeholder={t("admin.settings.smtpFromNamePlaceholder")}
        />
      </div>

      <AdminSettingsSMTPAdvancedFields
        onSMTPAuthMethodChange={onSMTPAuthMethodChange}
        onSMTPEncryptionChange={onSMTPEncryptionChange}
        onSMTPHeloNameChange={onSMTPHeloNameChange}
        onSMTPSkipTLSVerifyChange={onSMTPSkipTLSVerifyChange}
        onSMTPTimeoutSecondsChange={onSMTPTimeoutSecondsChange}
        smtpAuthMethod={smtpAuthMethod}
        smtpEncryption={smtpEncryption}
        smtpHeloName={smtpHeloName}
        smtpSkipTLSVerify={smtpSkipTLSVerify}
        smtpTimeoutSeconds={smtpTimeoutSeconds}
      />
    </>
  )
}
