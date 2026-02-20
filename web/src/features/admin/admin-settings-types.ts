export interface AdminSettingsTabProps {
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

export type AdminSettingsBasicSectionProps = Pick<
  AdminSettingsTabProps,
  | "maxIconFileSize"
  | "onMaxIconFileSizeChange"
  | "onRegistrationEnabledChange"
  | "onSiteNameChange"
  | "onSiteUrlChange"
  | "registrationEnabled"
  | "siteName"
  | "siteUrl"
>

export type AdminSettingsOIDCSectionProps = Pick<
  AdminSettingsTabProps,
  | "onOIDCAutoCreateUserChange"
  | "onOIDCAudienceChange"
  | "onOIDCAuthorizationEndpointChange"
  | "onOIDCClientIDChange"
  | "onOIDCClientSecretChange"
  | "onOIDCEnabledChange"
  | "onOIDCExtraAuthParamsChange"
  | "onOIDCIssuerURLChange"
  | "onOIDCProviderNameChange"
  | "onOIDCRedirectURLChange"
  | "onOIDCResourceChange"
  | "onOIDCScopesChange"
  | "onOIDCTokenEndpointChange"
  | "onOIDCUserinfoEndpointChange"
  | "oidcAutoCreateUser"
  | "oidcAudience"
  | "oidcAuthorizationEndpoint"
  | "oidcClientID"
  | "oidcClientSecret"
  | "oidcClientSecretConfigured"
  | "oidcEnabled"
  | "oidcExtraAuthParams"
  | "oidcIssuerURL"
  | "oidcProviderName"
  | "oidcRedirectURL"
  | "oidcResource"
  | "oidcScopes"
  | "oidcTokenEndpoint"
  | "oidcUserinfoEndpoint"
>

export type AdminSettingsOIDCAdvancedFieldsProps = Pick<
  AdminSettingsOIDCSectionProps,
  | "onOIDCAudienceChange"
  | "onOIDCAuthorizationEndpointChange"
  | "onOIDCExtraAuthParamsChange"
  | "onOIDCResourceChange"
  | "onOIDCTokenEndpointChange"
  | "onOIDCUserinfoEndpointChange"
  | "oidcAudience"
  | "oidcAuthorizationEndpoint"
  | "oidcExtraAuthParams"
  | "oidcResource"
  | "oidcTokenEndpoint"
  | "oidcUserinfoEndpoint"
>
