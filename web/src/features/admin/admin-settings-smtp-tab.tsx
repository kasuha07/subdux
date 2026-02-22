import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsSMTPSection from "./admin-settings-smtp-section"
import type { AdminSettingsSMTPTabProps } from "./admin-settings-types"

export default function AdminSettingsSMTPTab({
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
  onSave,
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
}: AdminSettingsSMTPTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="smtp" className="space-y-6">
      <AdminSettingsSMTPSection
        onSMTPAuthMethodChange={onSMTPAuthMethodChange}
        onSMTPEnabledChange={onSMTPEnabledChange}
        onSMTPEncryptionChange={onSMTPEncryptionChange}
        onSMTPFromEmailChange={onSMTPFromEmailChange}
        onSMTPFromNameChange={onSMTPFromNameChange}
        onSMTPHeloNameChange={onSMTPHeloNameChange}
        onSMTPHostChange={onSMTPHostChange}
        onSMTPPasswordChange={onSMTPPasswordChange}
        onSMTPSkipTLSVerifyChange={onSMTPSkipTLSVerifyChange}
        onSMTPPortChange={onSMTPPortChange}
        onSMTPTestRecipientChange={onSMTPTestRecipientChange}
        onSMTPTest={onSMTPTest}
        onSMTPTimeoutSecondsChange={onSMTPTimeoutSecondsChange}
        onSMTPUsernameChange={onSMTPUsernameChange}
        smtpAuthMethod={smtpAuthMethod}
        smtpEnabled={smtpEnabled}
        smtpEncryption={smtpEncryption}
        smtpFromEmail={smtpFromEmail}
        smtpFromName={smtpFromName}
        smtpHeloName={smtpHeloName}
        smtpHost={smtpHost}
        smtpPassword={smtpPassword}
        smtpPasswordConfigured={smtpPasswordConfigured}
        smtpPort={smtpPort}
        smtpSkipTLSVerify={smtpSkipTLSVerify}
        smtpTestRecipient={smtpTestRecipient}
        smtpTesting={smtpTesting}
        smtpTimeoutSeconds={smtpTimeoutSeconds}
        smtpUsername={smtpUsername}
      />

      <Separator />

      <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
    </TabsContent>
  )
}
