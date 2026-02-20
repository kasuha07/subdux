import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"

import AdminSettingsOIDCAdvancedFields from "./admin-settings-oidc-advanced-fields"
import type { AdminSettingsOIDCSectionProps } from "./admin-settings-types"

export default function AdminSettingsOIDCSection({
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
}: AdminSettingsOIDCSectionProps) {
  const { t } = useTranslation()
  const suggestedRedirectURL = useMemo(() => {
    if (typeof window === "undefined" || !window.location.origin) {
      return "/api/auth/oidc/callback"
    }
    return `${window.location.origin}/api/auth/oidc/callback`
  }, [])

  return (
    <>
      <div className="space-y-2">
        <h3 className="text-sm font-medium">{t("admin.settings.oidcTitle")}</h3>
        <p className="text-xs text-muted-foreground">{t("admin.settings.oidcDescription")}</p>
      </div>

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="oidc-enabled">{t("admin.settings.oidcEnabled")}</Label>
          <p className="text-sm text-muted-foreground">{t("admin.settings.oidcEnabledDescription")}</p>
        </div>
        <Switch id="oidc-enabled" checked={oidcEnabled} onCheckedChange={onOIDCEnabledChange} />
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-provider-name">{t("admin.settings.oidcProviderName")}</Label>
        <Input
          id="oidc-provider-name"
          value={oidcProviderName}
          onChange={(event) => onOIDCProviderNameChange(event.target.value)}
          placeholder="Google / Auth0 / Keycloak"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-issuer-url">{t("admin.settings.oidcIssuerURL")}</Label>
        <Input
          id="oidc-issuer-url"
          type="url"
          value={oidcIssuerURL}
          onChange={(event) => onOIDCIssuerURLChange(event.target.value)}
          placeholder="https://accounts.example.com"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-client-id">{t("admin.settings.oidcClientID")}</Label>
        <Input
          id="oidc-client-id"
          value={oidcClientID}
          onChange={(event) => onOIDCClientIDChange(event.target.value)}
          placeholder="client-id"
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-client-secret">{t("admin.settings.oidcClientSecret")}</Label>
        <Input
          id="oidc-client-secret"
          type="password"
          value={oidcClientSecret}
          onChange={(event) => onOIDCClientSecretChange(event.target.value)}
          placeholder={t("admin.settings.oidcClientSecretPlaceholder")}
        />
        <p className="text-xs text-muted-foreground">
          {oidcClientSecretConfigured
            ? t("admin.settings.oidcClientSecretConfigured")
            : t("admin.settings.oidcClientSecretNotConfigured")}
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-redirect-url">{t("admin.settings.oidcRedirectURL")}</Label>
        <Input
          id="oidc-redirect-url"
          type="url"
          value={oidcRedirectURL}
          onChange={(event) => onOIDCRedirectURLChange(event.target.value)}
          placeholder={suggestedRedirectURL}
        />
        <p className="text-xs text-muted-foreground">
          {t("admin.settings.oidcRedirectURLDescription", { url: suggestedRedirectURL })}
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="oidc-scopes">{t("admin.settings.oidcScopes")}</Label>
        <Input
          id="oidc-scopes"
          value={oidcScopes}
          onChange={(event) => onOIDCScopesChange(event.target.value)}
          placeholder="openid profile email"
        />
      </div>

      <AdminSettingsOIDCAdvancedFields
        onOIDCAudienceChange={onOIDCAudienceChange}
        onOIDCAuthorizationEndpointChange={onOIDCAuthorizationEndpointChange}
        onOIDCExtraAuthParamsChange={onOIDCExtraAuthParamsChange}
        onOIDCResourceChange={onOIDCResourceChange}
        onOIDCTokenEndpointChange={onOIDCTokenEndpointChange}
        onOIDCUserinfoEndpointChange={onOIDCUserinfoEndpointChange}
        oidcAudience={oidcAudience}
        oidcAuthorizationEndpoint={oidcAuthorizationEndpoint}
        oidcExtraAuthParams={oidcExtraAuthParams}
        oidcResource={oidcResource}
        oidcTokenEndpoint={oidcTokenEndpoint}
        oidcUserinfoEndpoint={oidcUserinfoEndpoint}
      />

      <div className="flex items-center justify-between">
        <div className="space-y-0.5">
          <Label htmlFor="oidc-auto-create">{t("admin.settings.oidcAutoCreateUser")}</Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.oidcAutoCreateUserDescription")}
          </p>
        </div>
        <Switch
          id="oidc-auto-create"
          checked={oidcAutoCreateUser}
          onCheckedChange={onOIDCAutoCreateUserChange}
        />
      </div>
    </>
  )
}
