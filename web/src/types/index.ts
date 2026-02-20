export interface User {
  id: number
  username: string
  email: string
  role: "admin" | "user"
  status: "active" | "disabled"
  totp_enabled: boolean
  created_at: string
  updated_at: string
}

export interface Subscription {
  id: number
  user_id: number
  name: string
  amount: number
  currency: string
  billing_cycle: "weekly" | "monthly" | "yearly"
  next_billing_date: string
  category: string
  icon: string
  url: string
  notes: string
  status: "active" | "paused" | "cancelled"
  color: string
  created_at: string
  updated_at: string
}

export interface DashboardSummary {
  total_monthly: number
  total_yearly: number
  active_count: number
  upcoming_renewals: Subscription[]
  currency: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface PasskeyCredential {
  id: number
  name: string
  credential_id: string
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
  billing_cycle: string
  next_billing_date: string
  category: string
  icon: string
  url: string
  notes: string
  color: string
}

export interface UpdateSubscriptionInput {
  name?: string
  amount?: number
  currency?: string
  billing_cycle?: string
  next_billing_date?: string
  category?: string
  icon?: string
  url?: string
  notes?: string
  status?: string
  color?: string
}

export interface ChangePasswordInput {
  current_password: string
  new_password: string
}

export interface AdminStats {
  total_users: number
  total_subscriptions: number
  total_monthly_spend: number
}

export interface SystemSettings {
  registration_enabled: boolean
  site_name: string
  site_url: string
  currencyapi_key: string
  exchange_rate_source: string
  max_icon_file_size: number
}

export interface UpdateSettingsInput {
  registration_enabled?: boolean
  site_name?: string
  site_url?: string
  currencyapi_key?: string
  exchange_rate_source?: string
  max_icon_file_size?: number
}

export interface UserPreference {
  user_id: number
  preferred_currency: string
  updated_at: string
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
  user_id: number
  code: string
  symbol: string
  alias: string
  sort_order: number
  created_at: string
  updated_at: string
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

export interface UploadIconResponse {
  icon: string
}
