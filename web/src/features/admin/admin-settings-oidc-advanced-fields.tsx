import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"

import type { AdminSettingsOIDCAdvancedFieldsProps } from "./admin-settings-types"

export default function AdminSettingsOIDCAdvancedFields({
  onOIDCAudienceChange,
  onOIDCAuthorizationEndpointChange,
  onOIDCExtraAuthParamsChange,
  onOIDCResourceChange,
  onOIDCTokenEndpointChange,
  onOIDCUserinfoEndpointChange,
  oidcAudience,
  oidcAuthorizationEndpoint,
  oidcExtraAuthParams,
  oidcResource,
  oidcTokenEndpoint,
  oidcUserinfoEndpoint,
}: AdminSettingsOIDCAdvancedFieldsProps) {
  const { t } = useTranslation()

  return (
    <details className="rounded-md border p-3">
      <summary className="cursor-pointer text-sm font-medium">
        {t("admin.settings.oidcAdvancedTitle")}
      </summary>
      <p className="mt-2 text-xs text-muted-foreground">{t("admin.settings.oidcAdvancedDescription")}</p>

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
  )
}
