export interface User {
  id: number
  username: string
  email: string
  role: "admin" | "user"
  status: "active" | "disabled"
  totp_enabled: boolean
}

export interface AdminUser {
  id: number
  email: string
  role: "admin" | "user"
  status: "active" | "disabled"
  created_at: string
}

export interface Subscription {
  id: number
  name: string
  amount: number
  currency: string
  enabled: boolean
  billing_type: "recurring" | "one_time"
  recurrence_type: "interval" | "monthly_date" | "yearly_date" | ""
  interval_count: number | null
  interval_unit: "day" | "week" | "month" | "year" | ""
  billing_anchor_date: string | null
  monthly_day: number | null
  yearly_month: number | null
  yearly_day: number | null
  trial_enabled: boolean
  trial_start_date: string | null
  trial_end_date: string | null
  next_billing_date: string | null
  category: string
  category_id: number | null
  payment_method_id: number | null
  icon: string
  url: string
  notes: string
  created_at: string
}

export interface DashboardSummary {
  total_monthly: number
  total_yearly: number
  enabled_count: number
  upcoming_renewal_count: number
  currency: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface RegistrationConfig {
  registration_enabled: boolean
  email_verification_enabled: boolean
}

export interface OIDCConfig {
  enabled: boolean
  provider_name: string
  auto_create_user: boolean
}

export interface OIDCStartResponse {
  authorization_url: string
}

export interface OIDCConnection {
  id: number
  provider: string
  email: string
  created_at: string
  updated_at: string
}

export interface OIDCSessionResult {
  purpose: "login" | "connect"
  token?: string
  user?: User
  connected?: boolean
  connection?: OIDCConnection
  error?: string
}

export interface PasskeyCredential {
  id: number
  name: string
  last_used_at: string | null
  created_at: string
}

export interface PasskeyBeginResult<TOptions = unknown> {
  session_id: string
  options: TOptions
}

export interface TotpSetupResponse {
  otpauth_uri: string
  secret: string
}

export interface TotpConfirmResponse {
  backup_codes: string[]
}

export interface TotpRequiredResponse {
  requires_totp: true
  totp_token: string
}

export type LoginResponse = AuthResponse | TotpRequiredResponse

export interface VerifyTotpInput {
  totp_token: string
  code: string
}

export interface DisableTotpInput {
  password: string
  code: string
}

export interface CreateSubscriptionInput {
  name: string
  amount: number
  currency: string
  enabled?: boolean
  billing_type: string
  recurrence_type: string
  interval_count: number | null
  interval_unit: string
  billing_anchor_date: string
  monthly_day: number | null
  yearly_month: number | null
  yearly_day: number | null
  trial_enabled: boolean
  trial_start_date: string
  trial_end_date: string
  category: string
  category_id: number | null
  payment_method_id: number | null
  icon: string
  url: string
  notes: string
}

export interface UpdateSubscriptionInput {
  name?: string
  amount?: number
  currency?: string
  enabled?: boolean
  billing_type?: string
  recurrence_type?: string
  interval_count?: number | null
  interval_unit?: string
  billing_anchor_date?: string
  monthly_day?: number | null
  yearly_month?: number | null
  yearly_day?: number | null
  trial_enabled?: boolean
  trial_start_date?: string
  trial_end_date?: string
  category?: string
  category_id?: number | null
  payment_method_id?: number | null
  icon?: string
  url?: string
  notes?: string
}

export interface ChangePasswordInput {
  current_password: string
  new_password: string
}

export interface ForgotPasswordInput {
  email: string
}

export interface ResetPasswordInput {
  email: string
  verification_code: string
  new_password: string
}

export interface SendEmailChangeCodeInput {
  new_email: string
  password: string
}

export interface ConfirmEmailChangeInput {
  new_email: string
  verification_code: string
}

export interface AdminStats {
  total_users: number
  total_subscriptions: number
  total_monthly_spend: number
}

export interface SystemSettings {
  registration_enabled: boolean
  registration_email_verification_enabled: boolean
  site_name: string
  site_url: string
  currencyapi_key: string
  exchange_rate_source: string
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
  site_name?: string
  site_url?: string
  currencyapi_key?: string
  exchange_rate_source?: string
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

export interface UserPreference {
  preferred_currency: string
}

export interface ExchangeRateInfo {
  base_currency: string
  target_currency: string
  rate: number
  source: string
  fetched_at: string
}

export interface ExchangeRateStatus {
  last_fetched_at: string | null
  source: string
  rate_count: number
}

export interface UserCurrency {
  id: number
  code: string
  symbol: string
  alias: string
  sort_order: number
}

export interface CreateCurrencyInput {
  code: string
  symbol: string
  alias: string
  sort_order: number
}

export interface UpdateCurrencyInput {
  symbol?: string
  alias?: string
  sort_order?: number
}

export interface ReorderCurrencyItem {
  id: number
  sort_order: number
}

export interface Category {
  id: number
  name: string
  system_key: string | null
  name_customized: boolean
  display_order: number
}

export interface CreateCategoryInput {
  name: string
  display_order: number
}

export interface UpdateCategoryInput {
  name?: string
  display_order?: number
}

export interface ReorderCategoryItem {
  id: number
  sort_order: number
}

export interface PaymentMethod {
  id: number
  name: string
  system_key: string | null
  name_customized: boolean
  icon: string
  sort_order: number
}

export interface CreatePaymentMethodInput {
  name: string
  icon: string
  sort_order: number
}

export interface UpdatePaymentMethodInput {
  name?: string
  icon?: string
  sort_order?: number
}

export interface ReorderPaymentMethodItem {
  id: number
  sort_order: number
}

export interface UploadIconResponse {
  icon: string
}
