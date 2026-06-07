export interface User {
  id: number
  username: string
  email: string
  role: "admin" | "user"
  status: "active" | "disabled"
  totp_enabled: boolean
}

export interface AuthResponse {
  token: string
  access_token?: string
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
  access_token?: string
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
