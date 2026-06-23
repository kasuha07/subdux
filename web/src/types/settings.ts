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

export interface APIKey {
  id: number
  name: string
  prefix: string
  key_kind: "mcp_client" | "api_integration"
  scopes: string[]
  last_used_at: string | null
  expires_at: string | null
  created_at: string
}

export interface CreateAPIKeyInput {
  name: string
  key_kind: "mcp_client" | "api_integration"
  expires_at?: string | null
  scopes?: string[]
}

export interface AuditEvent {
  event_id: string
  occurred_at: string
  user_id: number
  key_id: number
  key_kind: string
  scope_used: string
  transport: string
  tool_name: string
  resource_type: string
  resource_id: string
  action: string
  status: string
  error: string
  latency_ms: number
  client_name: string
  client_version: string
  request_id: string
  request_args_redacted?: unknown
  before_snapshot?: unknown
  after_snapshot?: unknown
}

export interface CreateAPIKeyResponse {
  api_key: APIKey
  key: string
}
