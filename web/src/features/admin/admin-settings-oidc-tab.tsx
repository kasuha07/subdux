import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsOIDCSection from "./admin-settings-oidc-section"
import type { AdminSettingsOIDCTabProps } from "./admin-settings-types"

export default function AdminSettingsOIDCTab({
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
  onSave,
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
}: AdminSettingsOIDCTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="auth" className="space-y-6">
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
    </TabsContent>
  )
}
