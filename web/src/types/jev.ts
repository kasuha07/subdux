export type JevConnectionStatus =
  | "not_configured"
  | "not_tested"
  | "available"
  | "invalid_credentials"
  | "rate_limited"
  | "unavailable"
  | "incompatible"

export interface JevSettings {
  revision: number
  enabled: boolean
  api_key_configured: boolean
  connection_status: JevConnectionStatus
  last_checked_at: string | null
  last_success_at: string | null
  classification_requests: number
  classification_suggestions: number
}

export interface JevCategorySuggestion {
  category_id: number | null
}
