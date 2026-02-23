import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"

import type { AdminSettingsBasicSectionProps } from "./admin-settings-types"

export default function AdminSettingsGeneralSection({
  allowImageUpload,
  maxIconFileSize,
  onAllowImageUploadChange,
  onMaxIconFileSizeChange,
  onRegistrationEmailVerificationEnabledChange,
  onRegistrationEnabledChange,
  onSiteNameChange,
  onSiteUrlChange,
  registrationEmailVerificationEnabled,
  registrationEnabled,
  siteName,
  siteUrl,
}: AdminSettingsBasicSectionProps) {
  const { t } = useTranslation()

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="site-name">{t("admin.settings.siteName")}</Label>
        <Input
          id="site-name"
          value={siteName}
          onChange={(event) => onSiteNameChange(event.target.value)}
          placeholder="Subdux"
        />
        <p className="text-xs text-muted-foreground">{t("admin.settings.siteNameDescription")}</p>
      </div>

      <Separator />

      <div className="space-y-2">
        <Label htmlFor="site-url">{t("admin.settings.siteUrl")}</Label>
        <Input
          id="site-url"
          type="url"
          value={siteUrl}
          onChange={(event) => onSiteUrlChange(event.target.value)}
          placeholder="https://example.com"
        />
        <p className="text-xs text-muted-foreground">{t("admin.settings.siteUrlDescription")}</p>
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="registration">{t("admin.settings.registrationEnabled")}</Label>
          <p className="text-sm text-muted-foreground">{t("admin.settings.registrationDescription")}</p>
        </div>
        <Switch
          id="registration"
          checked={registrationEnabled}
          onCheckedChange={onRegistrationEnabledChange}
        />
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="registration-email-verification">
            {t("admin.settings.registrationEmailVerificationEnabled")}
          </Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.registrationEmailVerificationDescription")}
          </p>
        </div>
        <Switch
          id="registration-email-verification"
          checked={registrationEmailVerificationEnabled}
          onCheckedChange={onRegistrationEmailVerificationEnabledChange}
        />
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="allow-image-upload">{t("admin.settings.allowImageUpload")}</Label>
          <p className="text-sm text-muted-foreground">{t("admin.settings.allowImageUploadDescription")}</p>
        </div>
        <Switch
          id="allow-image-upload"
          checked={allowImageUpload}
          onCheckedChange={onAllowImageUploadChange}
        />
      </div>

      <Separator />

      <div className="space-y-2">
        <Label htmlFor="max-icon-size">{t("admin.settings.maxIconFileSize")}</Label>
        <div className="flex items-center gap-2">
          <Input
            id="max-icon-size"
            type="number"
            min={1}
            max={10240}
            step={1}
            className="w-32"
            value={maxIconFileSize}
            onChange={(event) => {
              const next = parseInt(event.target.value, 10)
              if (!Number.isNaN(next) && next >= 1) {
                onMaxIconFileSizeChange(next)
              }
            }}
          />
          <span className="text-sm text-muted-foreground">
            {t("admin.settings.maxIconFileSizeUnit")}
          </span>
        </div>
        <p className="text-xs text-muted-foreground">{t("admin.settings.maxIconFileSizeDescription")}</p>
      </div>
    </>
  )
}
