import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsGeneralSection from "./admin-settings-general-section"
import type { AdminSettingsGeneralTabProps } from "./admin-settings-types"

export default function AdminSettingsTab({
  allowImageUpload,
  emailDomainWhitelist,
  maxIconFileSize,
  onAllowImageUploadChange,
  onEmailDomainWhitelistChange,
  onMaxIconFileSizeChange,
  onRegistrationEmailVerificationEnabledChange,
  onRegistrationEnabledChange,
  onSave,
  onSiteNameChange,
  onSiteUrlChange,
  registrationEmailVerificationEnabled,
  registrationEnabled,
  siteName,
  siteUrl,
}: AdminSettingsGeneralTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="settings" className="space-y-6">
      <AdminSettingsGeneralSection
        allowImageUpload={allowImageUpload}
        emailDomainWhitelist={emailDomainWhitelist}
        maxIconFileSize={maxIconFileSize}
        onAllowImageUploadChange={onAllowImageUploadChange}
        onEmailDomainWhitelistChange={onEmailDomainWhitelistChange}
        onMaxIconFileSizeChange={onMaxIconFileSizeChange}
        onRegistrationEmailVerificationEnabledChange={onRegistrationEmailVerificationEnabledChange}
        onRegistrationEnabledChange={onRegistrationEnabledChange}
        onSiteNameChange={onSiteNameChange}
        onSiteUrlChange={onSiteUrlChange}
        registrationEmailVerificationEnabled={registrationEmailVerificationEnabled}
        registrationEnabled={registrationEnabled}
        siteName={siteName}
        siteUrl={siteUrl}
      />

      <Separator />

      <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
    </TabsContent>
  )
}
