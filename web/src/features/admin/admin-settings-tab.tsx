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
  iconProxyDomainWhitelist,
  iconProxyEnabled,
  maxIconFileSize,
  mcpEnabled,
  auditEnabled,
  onAllowImageUploadChange,
  onIconProxyDomainWhitelistChange,
  onIconProxyEnabledChange,
  onMaxIconFileSizeChange,
  onMCPEnabledChange,
  onAuditEnabledChange,
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
    <TabsContent value="settings" className="space-y-6 select-none">
      <AdminSettingsGeneralSection
        allowImageUpload={allowImageUpload}
        iconProxyDomainWhitelist={iconProxyDomainWhitelist}
        iconProxyEnabled={iconProxyEnabled}
        maxIconFileSize={maxIconFileSize}
        mcpEnabled={mcpEnabled}
        auditEnabled={auditEnabled}
        onAllowImageUploadChange={onAllowImageUploadChange}
        onIconProxyDomainWhitelistChange={onIconProxyDomainWhitelistChange}
        onIconProxyEnabledChange={onIconProxyEnabledChange}
        onMaxIconFileSizeChange={onMaxIconFileSizeChange}
        onMCPEnabledChange={onMCPEnabledChange}
        onAuditEnabledChange={onAuditEnabledChange}
        onSiteNameChange={onSiteNameChange}
        onSiteUrlChange={onSiteUrlChange}
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
