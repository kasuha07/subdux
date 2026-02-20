import { useMemo } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { TabsContent } from "@/components/ui/tabs"

interface AdminSettingsTabProps {
  maxIconFileSize: number
  onMaxIconFileSizeChange: (value: number) => void
  onOIDCAutoCreateUserChange: (enabled: boolean) => void
  onOIDCAudienceChange: (value: string) => void
  onOIDCAuthorizationEndpointChange: (value: string) => void
  onOIDCClientIDChange: (value: string) => void
  onOIDCClientSecretChange: (value: string) => void
  onOIDCEnabledChange: (enabled: boolean) => void
  onOIDCExtraAuthParamsChange: (value: string) => void
  onOIDCIssuerURLChange: (value: string) => void
  onOIDCProviderNameChange: (value: string) => void
  onOIDCRedirectURLChange: (value: string) => void
  onOIDCResourceChange: (value: string) => void
  onOIDCScopesChange: (value: string) => void
  onOIDCTokenEndpointChange: (value: string) => void
  onOIDCUserinfoEndpointChange: (value: string) => void
  onRegistrationEnabledChange: (enabled: boolean) => void
  onSave: () => void | Promise<void>
  onSiteNameChange: (value: string) => void
  onSiteUrlChange: (value: string) => void
  oidcAutoCreateUser: boolean
  oidcAudience: string
  oidcAuthorizationEndpoint: string
  oidcClientID: string
  oidcClientSecret: string
  oidcClientSecretConfigured: boolean
  oidcEnabled: boolean
  oidcExtraAuthParams: string
  oidcIssuerURL: string
  oidcProviderName: string
  oidcRedirectURL: string
  oidcResource: string
  oidcScopes: string
  oidcTokenEndpoint: string
  oidcUserinfoEndpoint: string
  registrationEnabled: boolean
  siteName: string
  siteUrl: string
}

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
  const suggestedRedirectURL = useMemo(() => {
    if (typeof window === "undefined" || !window.location.origin) {
      return "/api/auth/oidc/callback"
    }
    return `${window.location.origin}/api/auth/oidc/callback`
  }, [])

  return (
    <TabsContent value="settings">
      <Card>
        <CardContent className="space-y-6 p-6">
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
              <p className="text-sm text-muted-foreground">
                {t("admin.settings.registrationDescription")}
              </p>
            </div>
            <Switch
              id="registration"
              checked={registrationEnabled}
              onCheckedChange={onRegistrationEnabledChange}
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
            <p className="text-xs text-muted-foreground">
              {t("admin.settings.maxIconFileSizeDescription")}
            </p>
          </div>

          <Separator />

          <div className="space-y-2">
            <h3 className="text-sm font-medium">{t("admin.settings.oidcTitle")}</h3>
            <p className="text-xs text-muted-foreground">{t("admin.settings.oidcDescription")}</p>
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="oidc-enabled">{t("admin.settings.oidcEnabled")}</Label>
              <p className="text-sm text-muted-foreground">
                {t("admin.settings.oidcEnabledDescription")}
              </p>
            </div>
            <Switch
              id="oidc-enabled"
              checked={oidcEnabled}
              onCheckedChange={onOIDCEnabledChange}
            />
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

          <details className="rounded-md border p-3">
            <summary className="cursor-pointer text-sm font-medium">
              {t("admin.settings.oidcAdvancedTitle")}
            </summary>
            <p className="mt-2 text-xs text-muted-foreground">
              {t("admin.settings.oidcAdvancedDescription")}
            </p>

            <div className="mt-3 space-y-3">
              <div className="space-y-1">
                <Label htmlFor="oidc-auth-endpoint">{t("admin.settings.oidcAuthorizationEndpoint")}</Label>
                <Input
                  id="oidc-auth-endpoint"
                  type="url"
                  value={oidcAuthorizationEndpoint}
                  onChange={(event) => onOIDCAuthorizationEndpointChange(event.target.value)}
                  placeholder="https://provider.example.com/oauth2/authorize"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="oidc-token-endpoint">{t("admin.settings.oidcTokenEndpoint")}</Label>
                <Input
                  id="oidc-token-endpoint"
                  type="url"
                  value={oidcTokenEndpoint}
                  onChange={(event) => onOIDCTokenEndpointChange(event.target.value)}
                  placeholder="https://provider.example.com/oauth2/token"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="oidc-userinfo-endpoint">{t("admin.settings.oidcUserinfoEndpoint")}</Label>
                <Input
                  id="oidc-userinfo-endpoint"
                  type="url"
                  value={oidcUserinfoEndpoint}
                  onChange={(event) => onOIDCUserinfoEndpointChange(event.target.value)}
                  placeholder="https://provider.example.com/userinfo"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="oidc-audience">{t("admin.settings.oidcAudience")}</Label>
                <Input
                  id="oidc-audience"
                  value={oidcAudience}
                  onChange={(event) => onOIDCAudienceChange(event.target.value)}
                  placeholder="api://default"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="oidc-resource">{t("admin.settings.oidcResource")}</Label>
                <Input
                  id="oidc-resource"
                  value={oidcResource}
                  onChange={(event) => onOIDCResourceChange(event.target.value)}
                  placeholder="https://api.example.com/"
                />
              </div>

              <div className="space-y-1">
                <Label htmlFor="oidc-extra-auth-params">{t("admin.settings.oidcExtraAuthParams")}</Label>
                <Input
                  id="oidc-extra-auth-params"
                  value={oidcExtraAuthParams}
                  onChange={(event) => onOIDCExtraAuthParamsChange(event.target.value)}
                  placeholder="prompt=consent&access_type=offline"
                />
                <p className="text-xs text-muted-foreground">
                  {t("admin.settings.oidcExtraAuthParamsDescription")}
                </p>
              </div>
            </div>
          </details>

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

          <Separator />

          <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
        </CardContent>
      </Card>
    </TabsContent>
  )
}
