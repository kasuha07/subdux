import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsGeneralSection from "./admin-settings-general-section"
import type { AdminSettingsGeneralTabProps } from "./admin-settings-types"

export default function AdminSettingsTab({
  maxIconFileSize,
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
    <TabsContent value="settings">
      <Card>
        <CardContent className="space-y-6 p-6">
          <AdminSettingsGeneralSection
            maxIconFileSize={maxIconFileSize}
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
        </CardContent>
      </Card>
    </TabsContent>
  )
}
