export interface AdminSettingsTabProps {
  allowImageUpload: boolean
  iconProxyEnabled: boolean
  iconProxyDomainWhitelist: string
  maxIconFileSize: number
  mcpEnabled: boolean
  onAllowImageUploadChange: (enabled: boolean) => void
  onIconProxyEnabledChange: (enabled: boolean) => void
  onIconProxyDomainWhitelistChange: (value: string) => void
  onMaxIconFileSizeChange: (value: number) => void
  onMCPEnabledChange: (enabled: boolean) => void
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
  onEmailDomainWhitelistChange: (value: string) => void
  onRegistrationEnabledChange: (enabled: boolean) => void
  onRegistrationEmailVerificationEnabledChange: (enabled: boolean) => void
  onSMTPAuthMethodChange: (value: string) => void
  onSMTPEnabledChange: (enabled: boolean) => void
  onSMTPEncryptionChange: (value: string) => void
  onSMTPFromEmailChange: (value: string) => void
  onSMTPFromNameChange: (value: string) => void
  onSMTPHeloNameChange: (value: string) => void
  onSMTPHostChange: (value: string) => void
  onSMTPPasswordChange: (value: string) => void
  onSMTPSkipTLSVerifyChange: (enabled: boolean) => void
  onSMTPPortChange: (value: number) => void
  onSMTPRateLimitSecondsChange: (value: number) => void
  onSMTPTestRecipientChange: (value: string) => void
  onSMTPTest: () => void | Promise<void>
  onSMTPTimeoutSecondsChange: (value: number) => void
  onSMTPUsernameChange: (value: string) => void
  onSystemProxyEnabledChange: (enabled: boolean) => void
  onSystemProxyTypeChange: (value: string) => void
  onSystemProxyUrlChange: (value: string) => void
  onSave: () => void | Promise<void>
  onSiteNameChange: (value: string) => void
  onSiteUrlChange: (value: string) => void
  emailDomainWhitelist: string
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
  registrationEmailVerificationEnabled: boolean
  smtpAuthMethod: string
  smtpEnabled: boolean
  smtpEncryption: string
  smtpFromEmail: string
  smtpFromName: string
  smtpHeloName: string
  smtpHost: string
  smtpPassword: string
  smtpPasswordConfigured: boolean
  smtpPort: number
  smtpRateLimitSeconds: number
  smtpSkipTLSVerify: boolean
  smtpTestRecipient: string
  smtpTesting: boolean
  smtpTimeoutSeconds: number
  smtpUsername: string
  systemProxyEnabled: boolean
  systemProxyType: string
  systemProxyUrl: string
  systemProxyUrlConfigured: boolean
  siteName: string
  siteUrl: string
}

export type AdminSettingsBasicSectionProps = Pick<
  AdminSettingsTabProps,
  | "allowImageUpload"
  | "emailDomainWhitelist"
  | "iconProxyDomainWhitelist"
  | "iconProxyEnabled"
  | "maxIconFileSize"
  | "mcpEnabled"
  | "onEmailDomainWhitelistChange"
  | "onAllowImageUploadChange"
  | "onIconProxyDomainWhitelistChange"
  | "onIconProxyEnabledChange"
  | "onMaxIconFileSizeChange"
  | "onMCPEnabledChange"
  | "onRegistrationEnabledChange"
  | "onRegistrationEmailVerificationEnabledChange"
  | "onSiteNameChange"
  | "onSiteUrlChange"
  | "registrationEnabled"
  | "registrationEmailVerificationEnabled"
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

export type AdminSettingsSMTPSectionProps = Pick<
  AdminSettingsTabProps,
  | "onSMTPAuthMethodChange"
  | "onSMTPEnabledChange"
  | "onSMTPEncryptionChange"
  | "onSMTPFromEmailChange"
  | "onSMTPFromNameChange"
  | "onSMTPHeloNameChange"
  | "onSMTPHostChange"
  | "onSMTPPasswordChange"
  | "onSMTPSkipTLSVerifyChange"
  | "onSMTPPortChange"
  | "onSMTPRateLimitSecondsChange"
  | "onSMTPTestRecipientChange"
  | "onSMTPTest"
  | "onSMTPTimeoutSecondsChange"
  | "onSMTPUsernameChange"
  | "smtpAuthMethod"
  | "smtpEnabled"
  | "smtpEncryption"
  | "smtpFromEmail"
  | "smtpFromName"
  | "smtpHeloName"
  | "smtpHost"
  | "smtpPassword"
  | "smtpPasswordConfigured"
  | "smtpPort"
  | "smtpRateLimitSeconds"
  | "smtpSkipTLSVerify"
  | "smtpTestRecipient"
  | "smtpTesting"
  | "smtpTimeoutSeconds"
  | "smtpUsername"
>

export type AdminSettingsProxySectionProps = Pick<
  AdminSettingsTabProps,
  | "onSystemProxyEnabledChange"
  | "onSystemProxyTypeChange"
  | "onSystemProxyUrlChange"
  | "systemProxyEnabled"
  | "systemProxyType"
  | "systemProxyUrl"
  | "systemProxyUrlConfigured"
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

export type AdminSettingsSMTPAdvancedFieldsProps = Pick<
  AdminSettingsSMTPSectionProps,
  | "onSMTPAuthMethodChange"
  | "onSMTPEncryptionChange"
  | "onSMTPHeloNameChange"
  | "onSMTPSkipTLSVerifyChange"
  | "onSMTPTimeoutSecondsChange"
  | "onSMTPRateLimitSecondsChange"
  | "smtpAuthMethod"
  | "smtpEncryption"
  | "smtpHeloName"
  | "smtpSkipTLSVerify"
  | "smtpTimeoutSeconds"
  | "smtpRateLimitSeconds"
>

export interface AdminSettingsSaveProps {
  onSave: () => void | Promise<void>
}

export type AdminSettingsGeneralTabProps = AdminSettingsBasicSectionProps &
  AdminSettingsSaveProps

export type AdminSettingsSMTPTabProps = AdminSettingsSMTPSectionProps &
  AdminSettingsSaveProps

export type AdminSettingsProxyTabProps = AdminSettingsProxySectionProps &
  AdminSettingsSaveProps

export type AdminSettingsOIDCTabProps = AdminSettingsOIDCSectionProps &
  AdminSettingsSaveProps
