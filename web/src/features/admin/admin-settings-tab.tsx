import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsGeneralSection from "./admin-settings-general-section"
import AdminSettingsProxySection from "./admin-settings-proxy-section"
import AdminSettingsSSRFSection from "./admin-settings-ssrf-section"
import type { AdminSettingsGeneralTabProps } from "./admin-settings-types"

export default function AdminSettingsTab({
  allowImageUpload,
  emailDomainWhitelist,
  iconProxyDomainWhitelist,
  iconProxyEnabled,
  maxIconFileSize,
  mcpEnabled,
  auditEnabled,
  onAllowImageUploadChange,
  onEmailDomainWhitelistChange,
  onIconProxyDomainWhitelistChange,
  onIconProxyEnabledChange,
  onMaxIconFileSizeChange,
  onMCPEnabledChange,
  onAuditEnabledChange,
  onRegistrationEmailVerificationEnabledChange,
  onRegistrationEnabledChange,
  onSave,
  onSiteNameChange,
  onSiteUrlChange,
  onSSRFAllowPrivateIPChange,
  onSSRFDomainFilterListChange,
  onSSRFDomainFilterModeChange,
  onSSRFFilterResolvedIPsChange,
  onSSRFIPFilterListChange,
  onSSRFIPFilterModeChange,
  onSSRFProtectionEnabledChange,
  onSSRFTest,
  onSSRFTestTargetChange,
  onSystemProxyEnabledChange,
  onSystemProxyTypeChange,
  onSystemProxyUrlChange,
  registrationEmailVerificationEnabled,
  registrationEnabled,
  siteName,
  siteUrl,
  ssrfAllowPrivateIP,
  ssrfDomainFilterList,
  ssrfDomainFilterMode,
  ssrfFilterResolvedIPs,
  ssrfIPFilterList,
  ssrfIPFilterMode,
  ssrfProtectionEnabled,
  ssrfTestResult,
  ssrfTestTarget,
  ssrfTesting,
  systemProxyEnabled,
  systemProxyType,
  systemProxyUrl,
  systemProxyUrlConfigured,
}: AdminSettingsGeneralTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="settings" className="space-y-6">
      <AdminSettingsGeneralSection
        allowImageUpload={allowImageUpload}
        emailDomainWhitelist={emailDomainWhitelist}
        iconProxyDomainWhitelist={iconProxyDomainWhitelist}
        iconProxyEnabled={iconProxyEnabled}
        maxIconFileSize={maxIconFileSize}
        mcpEnabled={mcpEnabled}
        auditEnabled={auditEnabled}
        onAllowImageUploadChange={onAllowImageUploadChange}
        onEmailDomainWhitelistChange={onEmailDomainWhitelistChange}
        onIconProxyDomainWhitelistChange={onIconProxyDomainWhitelistChange}
        onIconProxyEnabledChange={onIconProxyEnabledChange}
        onMaxIconFileSizeChange={onMaxIconFileSizeChange}
        onMCPEnabledChange={onMCPEnabledChange}
        onAuditEnabledChange={onAuditEnabledChange}
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

      <AdminSettingsProxySection
        onSystemProxyEnabledChange={onSystemProxyEnabledChange}
        onSystemProxyTypeChange={onSystemProxyTypeChange}
        onSystemProxyUrlChange={onSystemProxyUrlChange}
        systemProxyEnabled={systemProxyEnabled}
        systemProxyType={systemProxyType}
        systemProxyUrl={systemProxyUrl}
        systemProxyUrlConfigured={systemProxyUrlConfigured}
      />

      <Separator />

      <AdminSettingsSSRFSection
        onSSRFAllowPrivateIPChange={onSSRFAllowPrivateIPChange}
        onSSRFDomainFilterListChange={onSSRFDomainFilterListChange}
        onSSRFDomainFilterModeChange={onSSRFDomainFilterModeChange}
        onSSRFFilterResolvedIPsChange={onSSRFFilterResolvedIPsChange}
        onSSRFIPFilterListChange={onSSRFIPFilterListChange}
        onSSRFIPFilterModeChange={onSSRFIPFilterModeChange}
        onSSRFProtectionEnabledChange={onSSRFProtectionEnabledChange}
        onSSRFTest={onSSRFTest}
        onSSRFTestTargetChange={onSSRFTestTargetChange}
        ssrfAllowPrivateIP={ssrfAllowPrivateIP}
        ssrfDomainFilterList={ssrfDomainFilterList}
        ssrfDomainFilterMode={ssrfDomainFilterMode}
        ssrfFilterResolvedIPs={ssrfFilterResolvedIPs}
        ssrfIPFilterList={ssrfIPFilterList}
        ssrfIPFilterMode={ssrfIPFilterMode}
        ssrfProtectionEnabled={ssrfProtectionEnabled}
        ssrfTestResult={ssrfTestResult}
        ssrfTestTarget={ssrfTestTarget}
        ssrfTesting={ssrfTesting}
      />

      <Separator />

      <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
    </TabsContent>
  )
}
