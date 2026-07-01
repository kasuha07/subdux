export interface AdminUser {
  id: number
  email: string
  role: "admin" | "user"
  status: "active" | "disabled"
  created_at: string
}

export interface AdminStats {
  total_users: number
  total_subscriptions: number
  total_monthly_spend: number
}

export interface BackgroundTask {
  key: string
  name: string
  description: string
  interval_seconds: number
  status: "idle" | "running" | "succeeded" | "failed"
  running: boolean
  last_started_at: string | null
  last_finished_at: string | null
  next_run_at: string | null
  last_duration_ms: number
  last_error: string
  success_count: number
  failure_count: number
}

export interface SystemSettings {
  registration_enabled: boolean
  registration_email_verification_enabled: boolean
  email_domain_whitelist: string
  site_name: string
  site_url: string
  icon_proxy_enabled: boolean
  icon_proxy_domain_whitelist: string
  mcp_enabled: boolean
  audit_enabled: boolean
  system_proxy_enabled: boolean
  system_proxy_type: string
  system_proxy_url_configured: boolean
  ssrf_protection_enabled: boolean
  ssrf_allow_private_ip: boolean
  ssrf_domain_filter_mode: string
  ssrf_domain_filter_list: string
  ssrf_ip_filter_mode: string
  ssrf_ip_filter_list: string
  ssrf_filter_resolved_ips: boolean
  currencyapi_key_configured: boolean
  exchange_rate_source: string
  allow_image_upload: boolean
  max_icon_file_size: number
  smtp_enabled: boolean
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password_configured: boolean
  smtp_from_email: string
  smtp_from_name: string
  smtp_encryption: string
  smtp_auth_method: string
  smtp_helo_name: string
  smtp_timeout_seconds: number
  smtp_rate_limit_seconds: number
  smtp_skip_tls_verify: boolean
  oidc_enabled: boolean
  oidc_provider_name: string
  oidc_issuer_url: string
  oidc_client_id: string
  oidc_client_secret_configured: boolean
  oidc_redirect_url: string
  oidc_scopes: string
  oidc_auto_create_user: boolean
  oidc_authorization_endpoint: string
  oidc_token_endpoint: string
  oidc_userinfo_endpoint: string
  oidc_audience: string
  oidc_resource: string
  oidc_extra_auth_params: string
}

export interface UpdateSettingsInput {
  registration_enabled?: boolean
  registration_email_verification_enabled?: boolean
  email_domain_whitelist?: string
  site_name?: string
  site_url?: string
  icon_proxy_enabled?: boolean
  icon_proxy_domain_whitelist?: string
  mcp_enabled?: boolean
  audit_enabled?: boolean
  system_proxy_enabled?: boolean
  system_proxy_type?: string
  system_proxy_url?: string
  ssrf_protection_enabled?: boolean
  ssrf_allow_private_ip?: boolean
  ssrf_domain_filter_mode?: string
  ssrf_domain_filter_list?: string
  ssrf_ip_filter_mode?: string
  ssrf_ip_filter_list?: string
  ssrf_filter_resolved_ips?: boolean
  currencyapi_key?: string
  exchange_rate_source?: string
  allow_image_upload?: boolean
  max_icon_file_size?: number
  smtp_enabled?: boolean
  smtp_host?: string
  smtp_port?: number
  smtp_username?: string
  smtp_password?: string
  smtp_from_email?: string
  smtp_from_name?: string
  smtp_encryption?: string
  smtp_auth_method?: string
  smtp_helo_name?: string
  smtp_timeout_seconds?: number
  smtp_rate_limit_seconds?: number
  smtp_skip_tls_verify?: boolean
  oidc_enabled?: boolean
  oidc_provider_name?: string
  oidc_issuer_url?: string
  oidc_client_id?: string
  oidc_client_secret?: string
  oidc_redirect_url?: string
  oidc_scopes?: string
  oidc_auto_create_user?: boolean
  oidc_authorization_endpoint?: string
  oidc_token_endpoint?: string
  oidc_userinfo_endpoint?: string
  oidc_audience?: string
  oidc_resource?: string
  oidc_extra_auth_params?: string
}

export interface SSRFTestResult {
  target: string
  host: string
  allowed: boolean
  reason: string
  resolved_ips: string[]
  protection_enabled: boolean
  allow_private_ip: boolean
  domain_filter_mode: string
  ip_filter_mode: string
  filter_resolved_ips: boolean
  proxy_mediated: boolean
  resolved_ip_filter_applied: boolean
}
