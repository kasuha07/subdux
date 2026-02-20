import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsGeneralSection from "./admin-settings-general-section"
import AdminSettingsOIDCSection from "./admin-settings-oidc-section"
import type { AdminSettingsTabProps } from "./admin-settings-types"

export default function AdminSettingsTab({
  maxIconFileSize,
  onMaxIconFileSizeChange,
  onOIDCAutoCreateUserChange,
  onOIDCAudienceChange,
  onOIDCAuthorizationEndpointChange,
  onOIDCClientIDChange,
  onOIDCClientSecretChange,
  onOIDCEnabledChange,
  onOIDCExtraAuthParamsChange,
  onOIDCIssuerURLChange,
  onOIDCProviderNameChange,
  onOIDCRedirectURLChange,
  onOIDCResourceChange,
  onOIDCScopesChange,
  onOIDCTokenEndpointChange,
  onOIDCUserinfoEndpointChange,
  onRegistrationEnabledChange,
  onSave,
  onSiteNameChange,
  onSiteUrlChange,
  oidcAutoCreateUser,
  oidcAudience,
  oidcAuthorizationEndpoint,
  oidcClientID,
  oidcClientSecret,
  oidcClientSecretConfigured,
  oidcEnabled,
  oidcExtraAuthParams,
  oidcIssuerURL,
  oidcProviderName,
  oidcRedirectURL,
  oidcResource,
  oidcScopes,
  oidcTokenEndpoint,
  oidcUserinfoEndpoint,
  registrationEnabled,
  siteName,
  siteUrl,
}: AdminSettingsTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="settings">
      <Card>
        <CardContent className="space-y-6 p-6">
          <AdminSettingsGeneralSection
            maxIconFileSize={maxIconFileSize}
            onMaxIconFileSizeChange={onMaxIconFileSizeChange}
            onRegistrationEnabledChange={onRegistrationEnabledChange}
            onSiteNameChange={onSiteNameChange}
            onSiteUrlChange={onSiteUrlChange}
            registrationEnabled={registrationEnabled}
            siteName={siteName}
            siteUrl={siteUrl}
          />

          <Separator />

          <AdminSettingsOIDCSection
            onOIDCAutoCreateUserChange={onOIDCAutoCreateUserChange}
            onOIDCAudienceChange={onOIDCAudienceChange}
            onOIDCAuthorizationEndpointChange={onOIDCAuthorizationEndpointChange}
            onOIDCClientIDChange={onOIDCClientIDChange}
            onOIDCClientSecretChange={onOIDCClientSecretChange}
            onOIDCEnabledChange={onOIDCEnabledChange}
            onOIDCExtraAuthParamsChange={onOIDCExtraAuthParamsChange}
            onOIDCIssuerURLChange={onOIDCIssuerURLChange}
            onOIDCProviderNameChange={onOIDCProviderNameChange}
            onOIDCRedirectURLChange={onOIDCRedirectURLChange}
            onOIDCResourceChange={onOIDCResourceChange}
            onOIDCScopesChange={onOIDCScopesChange}
            onOIDCTokenEndpointChange={onOIDCTokenEndpointChange}
            onOIDCUserinfoEndpointChange={onOIDCUserinfoEndpointChange}
            oidcAutoCreateUser={oidcAutoCreateUser}
            oidcAudience={oidcAudience}
            oidcAuthorizationEndpoint={oidcAuthorizationEndpoint}
            oidcClientID={oidcClientID}
            oidcClientSecret={oidcClientSecret}
            oidcClientSecretConfigured={oidcClientSecretConfigured}
            oidcEnabled={oidcEnabled}
            oidcExtraAuthParams={oidcExtraAuthParams}
            oidcIssuerURL={oidcIssuerURL}
            oidcProviderName={oidcProviderName}
            oidcRedirectURL={oidcRedirectURL}
            oidcResource={oidcResource}
            oidcScopes={oidcScopes}
            oidcTokenEndpoint={oidcTokenEndpoint}
            oidcUserinfoEndpoint={oidcUserinfoEndpoint}
          />

          <Separator />

          <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
        </CardContent>
      </Card>
    </TabsContent>
  )
}
