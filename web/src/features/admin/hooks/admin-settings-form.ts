import type { SystemSettings, UpdateSettingsInput } from "@/types"

export type AdminSettingsSaveScope = "general" | "smtp" | "auth" | "exchange-rates" | "backup"

export interface AdminSettingsFormState {
  allowImageUpload: boolean
  backupScheduleEnabled: boolean
  backupTimeOfDay: string
  backupIncludeAssets: boolean
  backupEncryptEnabled: boolean
  backupEncryptionPassword: string
  backupEncryptionPasswordConfigured: boolean
  backupLocalDir: string
  backupRetentionCount: number
  currencyApiKey: string
  currencyApiKeyConfigured: boolean
  emailDomainWhitelist: string
  exchangeRateSource: string
  iconProxyDomainWhitelist: string
  iconProxyEnabled: boolean
  maxIconFileSize: number
  mcpEnabled: boolean
  auditEnabled: boolean
  oidcAudience: string
  oidcAuthorizationEndpoint: string
  oidcAutoCreateUser: boolean
  oidcClientID: string
  oidcClientSecret: string
  oidcClientSecretConfigured: boolean
  oidcEnabled: boolean
  oidcExtraAuthParams: string
  oidcReauthACRMFA: string
  oidcReauthACRPhishingResistant: string
  oidcIssuerURL: string
  oidcProviderName: string
  oidcRedirectURL: string
  oidcResource: string
  oidcScopes: string
  oidcTokenEndpoint: string
  oidcUserinfoEndpoint: string
  registrationEmailVerificationEnabled: boolean
  registrationEnabled: boolean
  siteName: string
  siteUrl: string
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
  smtpTimeoutSeconds: number
  smtpUsername: string
  ssrfAllowPrivateIP: boolean
  ssrfDomainFilterList: string
  ssrfDomainFilterMode: string
  ssrfFilterResolvedIPs: boolean
  ssrfIPFilterList: string
  ssrfIPFilterMode: string
  ssrfProtectionEnabled: boolean
  systemProxyEnabled: boolean
  systemProxyType: string
  systemProxyUrl: string
  systemProxyUrlConfigured: boolean
}

const formFieldsByScope: Record<AdminSettingsSaveScope, readonly (keyof AdminSettingsFormState)[]> = {
  general: [
    "allowImageUpload",
    "auditEnabled",
    "iconProxyDomainWhitelist",
    "iconProxyEnabled",
    "maxIconFileSize",
    "mcpEnabled",
    "siteName",
    "siteUrl",
    "ssrfAllowPrivateIP",
    "ssrfDomainFilterList",
    "ssrfDomainFilterMode",
    "ssrfFilterResolvedIPs",
    "ssrfIPFilterList",
    "ssrfIPFilterMode",
    "ssrfProtectionEnabled",
    "systemProxyEnabled",
    "systemProxyType",
    "systemProxyUrl",
    "systemProxyUrlConfigured",
  ],
  smtp: [
    "smtpAuthMethod",
    "smtpEnabled",
    "smtpEncryption",
    "smtpFromEmail",
    "smtpFromName",
    "smtpHeloName",
    "smtpHost",
    "smtpPassword",
    "smtpPasswordConfigured",
    "smtpPort",
    "smtpRateLimitSeconds",
    "smtpSkipTLSVerify",
    "smtpTimeoutSeconds",
    "smtpUsername",
  ],
  auth: [
    "emailDomainWhitelist",
    "oidcAudience",
    "oidcAuthorizationEndpoint",
    "oidcAutoCreateUser",
    "oidcClientID",
    "oidcClientSecret",
    "oidcClientSecretConfigured",
    "oidcEnabled",
    "oidcExtraAuthParams",
    "oidcReauthACRMFA",
    "oidcReauthACRPhishingResistant",
    "oidcIssuerURL",
    "oidcProviderName",
    "oidcRedirectURL",
    "oidcResource",
    "oidcScopes",
    "oidcTokenEndpoint",
    "oidcUserinfoEndpoint",
    "registrationEmailVerificationEnabled",
    "registrationEnabled",
  ],
  "exchange-rates": ["currencyApiKey", "currencyApiKeyConfigured", "exchangeRateSource"],
  backup: [
    "backupEncryptEnabled",
    "backupEncryptionPassword",
    "backupEncryptionPasswordConfigured",
    "backupIncludeAssets",
    "backupLocalDir",
    "backupRetentionCount",
    "backupScheduleEnabled",
    "backupTimeOfDay",
  ],
}

export function createAdminSettingsForm(settings?: SystemSettings): AdminSettingsFormState {
  return {
    allowImageUpload: settings?.allow_image_upload ?? true,
    backupScheduleEnabled: settings?.backup_schedule_enabled ?? false,
    backupTimeOfDay: settings?.backup_time_of_day || "03:00",
    backupIncludeAssets: settings?.backup_include_assets ?? false,
    backupEncryptEnabled: settings?.backup_encrypt_enabled ?? false,
    backupEncryptionPassword: "",
    backupEncryptionPasswordConfigured: settings?.backup_encryption_password_configured ?? false,
    backupLocalDir: settings?.backup_local_dir || "",
    backupRetentionCount: settings?.backup_retention_count ?? 7,
    currencyApiKey: "",
    currencyApiKeyConfigured: settings?.currencyapi_key_configured ?? false,
    emailDomainWhitelist: settings?.email_domain_whitelist || "",
    exchangeRateSource: settings?.exchange_rate_source || "auto",
    iconProxyDomainWhitelist: settings?.icon_proxy_domain_whitelist || "",
    iconProxyEnabled: settings?.icon_proxy_enabled ?? true,
    maxIconFileSize: settings?.max_icon_file_size
      ? Math.round(settings.max_icon_file_size / 1024)
      : 64,
    mcpEnabled: settings?.mcp_enabled ?? false,
    auditEnabled: settings?.audit_enabled ?? true,
    oidcAudience: settings?.oidc_audience || "",
    oidcAuthorizationEndpoint: settings?.oidc_authorization_endpoint || "",
    oidcAutoCreateUser: settings?.oidc_auto_create_user ?? false,
    oidcClientID: settings?.oidc_client_id || "",
    oidcClientSecret: "",
    oidcClientSecretConfigured: settings?.oidc_client_secret_configured ?? false,
    oidcEnabled: settings?.oidc_enabled ?? false,
    oidcExtraAuthParams: settings?.oidc_extra_auth_params || "",
    oidcReauthACRMFA: settings?.oidc_reauth_acr_mfa || "",
    oidcReauthACRPhishingResistant: settings?.oidc_reauth_acr_phishing_resistant || "",
    oidcIssuerURL: settings?.oidc_issuer_url || "",
    oidcProviderName: settings?.oidc_provider_name || "OIDC",
    oidcRedirectURL: settings?.oidc_redirect_url || "",
    oidcResource: settings?.oidc_resource || "",
    oidcScopes: settings?.oidc_scopes || "openid profile email",
    oidcTokenEndpoint: settings?.oidc_token_endpoint || "",
    oidcUserinfoEndpoint: settings?.oidc_userinfo_endpoint || "",
    registrationEmailVerificationEnabled:
      settings?.registration_email_verification_enabled ?? false,
    registrationEnabled: settings?.registration_enabled ?? false,
    siteName: settings?.site_name || "Subdux",
    siteUrl: settings?.site_url || "",
    smtpAuthMethod: settings?.smtp_auth_method || "auto",
    smtpEnabled: settings?.smtp_enabled ?? false,
    smtpEncryption: settings?.smtp_encryption || "starttls",
    smtpFromEmail: settings?.smtp_from_email || "",
    smtpFromName: settings?.smtp_from_name || "",
    smtpHeloName: settings?.smtp_helo_name || "",
    smtpHost: settings?.smtp_host || "",
    smtpPassword: "",
    smtpPasswordConfigured: settings?.smtp_password_configured ?? false,
    smtpPort: settings?.smtp_port || 587,
    smtpRateLimitSeconds: settings?.smtp_rate_limit_seconds ?? 0,
    smtpSkipTLSVerify: settings?.smtp_skip_tls_verify ?? false,
    smtpTimeoutSeconds: settings?.smtp_timeout_seconds || 10,
    smtpUsername: settings?.smtp_username || "",
    ssrfAllowPrivateIP: settings?.ssrf_allow_private_ip ?? false,
    ssrfDomainFilterList: settings?.ssrf_domain_filter_list || "",
    ssrfDomainFilterMode: settings?.ssrf_domain_filter_mode || "blacklist",
    ssrfFilterResolvedIPs: settings?.ssrf_filter_resolved_ips ?? true,
    ssrfIPFilterList: settings?.ssrf_ip_filter_list || "",
    ssrfIPFilterMode: settings?.ssrf_ip_filter_mode || "blacklist",
    ssrfProtectionEnabled: settings?.ssrf_protection_enabled ?? true,
    systemProxyEnabled: settings?.system_proxy_enabled ?? false,
    systemProxyType: settings?.system_proxy_type || "http",
    systemProxyUrl: "",
    systemProxyUrlConfigured: settings?.system_proxy_url_configured ?? false,
  }
}

export function buildAdminSettingsPayload(
  form: AdminSettingsFormState,
  scope: AdminSettingsSaveScope
): UpdateSettingsInput {
  switch (scope) {
    case "general":
      return buildGeneralSettingsPayload(form)
    case "smtp":
      return buildSMTPSettingsPayload(form)
    case "auth":
      return buildAuthSettingsPayload(form)
    case "exchange-rates":
      return buildExchangeRateSettingsPayload(form)
    case "backup":
      return buildBackupSettingsPayload(form)
  }
}

export function mergeAdminSettingsFormScope(
  current: AdminSettingsFormState,
  fresh: SystemSettings,
  scope: AdminSettingsSaveScope
): AdminSettingsFormState {
  const freshForm = createAdminSettingsForm(fresh)
  const next = { ...current }
  for (const field of formFieldsByScope[scope]) {
    assignFormField(next, freshForm, field)
  }
  return next
}

function assignFormField<K extends keyof AdminSettingsFormState>(
  target: AdminSettingsFormState,
  source: AdminSettingsFormState,
  field: K
) {
  target[field] = source[field]
}

function buildGeneralSettingsPayload(form: AdminSettingsFormState): UpdateSettingsInput {
  const payload: UpdateSettingsInput = {
    allow_image_upload: form.allowImageUpload,
    audit_enabled: form.auditEnabled,
    icon_proxy_domain_whitelist: form.iconProxyDomainWhitelist,
    icon_proxy_enabled: form.iconProxyEnabled,
    max_icon_file_size: form.maxIconFileSize * 1024,
    mcp_enabled: form.mcpEnabled,
    site_name: form.siteName,
    site_url: form.siteUrl,
    ssrf_allow_private_ip: form.ssrfAllowPrivateIP,
    ssrf_domain_filter_list: form.ssrfDomainFilterList,
    ssrf_domain_filter_mode: form.ssrfDomainFilterMode,
    ssrf_filter_resolved_ips: form.ssrfFilterResolvedIPs,
    ssrf_ip_filter_list: form.ssrfIPFilterList,
    ssrf_ip_filter_mode: form.ssrfIPFilterMode,
    ssrf_protection_enabled: form.ssrfProtectionEnabled,
    system_proxy_enabled: form.systemProxyEnabled,
    system_proxy_type: form.systemProxyType,
  }
  if (form.systemProxyUrl.trim()) {
    payload.system_proxy_url = form.systemProxyUrl.trim()
  }
  return payload
}

function buildSMTPSettingsPayload(form: AdminSettingsFormState): UpdateSettingsInput {
  const payload: UpdateSettingsInput = {
    smtp_auth_method: form.smtpAuthMethod,
    smtp_enabled: form.smtpEnabled,
    smtp_encryption: form.smtpEncryption,
    smtp_from_email: form.smtpFromEmail,
    smtp_from_name: form.smtpFromName,
    smtp_helo_name: form.smtpHeloName,
    smtp_host: form.smtpHost,
    smtp_port: form.smtpPort,
    smtp_rate_limit_seconds: form.smtpRateLimitSeconds,
    smtp_skip_tls_verify: form.smtpSkipTLSVerify,
    smtp_timeout_seconds: form.smtpTimeoutSeconds,
    smtp_username: form.smtpUsername,
  }
  if (form.smtpPassword.trim()) {
    payload.smtp_password = form.smtpPassword.trim()
  }
  return payload
}

function buildAuthSettingsPayload(form: AdminSettingsFormState): UpdateSettingsInput {
  const payload: UpdateSettingsInput = {
    email_domain_whitelist: form.emailDomainWhitelist,
    oidc_audience: form.oidcAudience,
    oidc_authorization_endpoint: form.oidcAuthorizationEndpoint,
    oidc_auto_create_user: form.oidcAutoCreateUser,
    oidc_client_id: form.oidcClientID,
    oidc_enabled: form.oidcEnabled,
    oidc_extra_auth_params: form.oidcExtraAuthParams,
    oidc_reauth_acr_mfa: form.oidcReauthACRMFA,
    oidc_reauth_acr_phishing_resistant: form.oidcReauthACRPhishingResistant,
    oidc_issuer_url: form.oidcIssuerURL,
    oidc_provider_name: form.oidcProviderName,
    oidc_redirect_url: form.oidcRedirectURL,
    oidc_resource: form.oidcResource,
    oidc_scopes: form.oidcScopes,
    oidc_token_endpoint: form.oidcTokenEndpoint,
    oidc_userinfo_endpoint: form.oidcUserinfoEndpoint,
    registration_email_verification_enabled: form.registrationEmailVerificationEnabled,
    registration_enabled: form.registrationEnabled,
  }
  if (form.oidcClientSecret.trim()) {
    payload.oidc_client_secret = form.oidcClientSecret.trim()
  }
  return payload
}

function buildExchangeRateSettingsPayload(form: AdminSettingsFormState): UpdateSettingsInput {
  const payload: UpdateSettingsInput = {
    exchange_rate_source: form.exchangeRateSource,
  }
  if (form.currencyApiKey.trim()) {
    payload.currencyapi_key = form.currencyApiKey.trim()
  }
  return payload
}

function buildBackupSettingsPayload(form: AdminSettingsFormState): UpdateSettingsInput {
  const payload: UpdateSettingsInput = {
    backup_encrypt_enabled: form.backupEncryptEnabled,
    backup_include_assets: form.backupIncludeAssets,
    backup_local_dir: form.backupLocalDir,
    backup_retention_count: form.backupRetentionCount,
    backup_schedule_enabled: form.backupScheduleEnabled,
    backup_time_of_day: form.backupTimeOfDay,
  }
  if (form.backupEncryptionPassword.trim()) {
    payload.backup_encryption_password = form.backupEncryptionPassword.trim()
  }
  return payload
}
