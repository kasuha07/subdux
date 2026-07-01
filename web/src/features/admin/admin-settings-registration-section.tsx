import { useTranslation } from "react-i18next"

import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"

import type { AdminSettingsRegistrationSectionProps } from "./admin-settings-types"

export default function AdminSettingsRegistrationSection({
  emailDomainWhitelist,
  onEmailDomainWhitelistChange,
  onRegistrationEmailVerificationEnabledChange,
  onRegistrationEnabledChange,
  registrationEmailVerificationEnabled,
  registrationEnabled,
}: AdminSettingsRegistrationSectionProps) {
  const { t } = useTranslation()

  return (
    <>
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

      <div className="space-y-2">
        <Label htmlFor="email-domain-whitelist">{t("admin.settings.emailDomainWhitelist")}</Label>
        <Textarea
          id="email-domain-whitelist"
          value={emailDomainWhitelist}
          onChange={(event) => onEmailDomainWhitelistChange(event.target.value)}
          placeholder={t("admin.settings.emailDomainWhitelistPlaceholder")}
          rows={4}
        />
        <p className="text-xs text-muted-foreground">
          {t("admin.settings.emailDomainWhitelistDescription")}
        </p>
      </div>
    </>
  )
}
