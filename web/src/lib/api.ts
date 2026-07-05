import { toast } from "@/lib/toast"
import i18n from "@/i18n"
import type { AuthResponse, User } from "@/types"

const API_BASE = "/api"
const ACCESS_TOKEN_KEY = "token"
const USER_KEY = "user"

let refreshRequest: Promise<boolean> | null = null
let cachedAccessToken: string | null = null
let cachedUser: User | null = null
let accessTokenLoaded = false
let userLoaded = false

export type BackendMessageParams = Record<string, string | number | boolean>

export interface BackendErrorPayload {
  error_code?: unknown
  error_params?: unknown
}

export interface BackendMessagePayload {
  message_code?: unknown
  message_params?: unknown
}

type BackendPayload = BackendErrorPayload & BackendMessagePayload

export interface APIRequestOptions extends RequestInit {
  errorHandling?: "silent" | "toast"
  errorFallbackKey?: string
}

export class BackendAPIError extends Error {
  readonly code?: string
  readonly params?: BackendMessageParams
  readonly status?: number

  constructor(message: string, options: { code?: string; params?: BackendMessageParams; status?: number } = {}) {
    super(message)
    this.name = "BackendAPIError"
    this.code = options.code
    this.params = options.params
    this.status = options.status
  }
}

export function isBackendAPIError(error: unknown): error is BackendAPIError {
  return error instanceof BackendAPIError
}

export function getAPIErrorMessage(error: unknown, fallbackKey = "common.requestFailed"): string {
  if (isBackendAPIError(error)) {
    return error.message
  }

  return i18n.t(fallbackKey)
}

function readLocalStorage(key: string): string | null {
  if (typeof window === "undefined") {
    return null
  }

  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

function writeLocalStorage(key: string, value: string): void {
  if (typeof window === "undefined") {
    return
  }

  try {
    window.localStorage.setItem(key, value)
  } catch {
    void 0
  }
}

function removeLocalStorage(key: string): void {
  if (typeof window === "undefined") {
    return
  }

  try {
    window.localStorage.removeItem(key)
  } catch {
    void 0
  }
}

function getToken(): string | null {
  if (!accessTokenLoaded) {
    removeLocalStorage(ACCESS_TOKEN_KEY)
    cachedAccessToken = null
    accessTokenLoaded = true
  }

  return cachedAccessToken
}

export function setToken(token: string): void {
  cachedAccessToken = token
  accessTokenLoaded = true
  removeLocalStorage(ACCESS_TOKEN_KEY)
}

export function clearToken(): void {
  cachedAccessToken = null
  cachedUser = null
  accessTokenLoaded = true
  userLoaded = true
  removeLocalStorage(ACCESS_TOKEN_KEY)
  removeLocalStorage(USER_KEY)
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function setUser(user: User): void {
  cachedUser = user
  userLoaded = true
  writeLocalStorage(USER_KEY, JSON.stringify(user))
}

export function getUser(): User | null {
  if (!userLoaded) {
    const raw = readLocalStorage(USER_KEY)
    if (!raw) {
      cachedUser = null
      userLoaded = true
      return cachedUser
    }

    try {
      cachedUser = JSON.parse(raw) as User
    } catch {
      cachedUser = null
      removeLocalStorage(USER_KEY)
    }
    userLoaded = true
  }

  return cachedUser
}

export function isAdmin(): boolean {
  return getUser()?.role === "admin"
}

export function setAuth(token: string, user: User): void {
  setToken(token)
  setUser(user)
}

export async function restoreSession(): Promise<boolean> {
  if (getToken()) {
    return true
  }

  const restored = await refreshSession()
  if (!restored) {
    clearToken()
  }

  return restored
}

function resolveAccessToken(data: Partial<AuthResponse>): string | null {
  return data.access_token ?? data.token ?? null
}

function handleUnauthorized(): never {
  clearToken()
  const message = i18n.t("common.unauthorized")
  toast.error(message)
  window.location.href = "/login"
  throw new BackendAPIError(message, { status: 401 })
}

function canRefresh(path: string, hasAccessToken: boolean): boolean {
  return hasAccessToken && path !== "/auth/refresh"
}

function normalizeBackendParams(value: unknown): BackendMessageParams | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined
  }

  const params: BackendMessageParams = {}
  for (const [key, rawValue] of Object.entries(value)) {
    if (
      typeof rawValue === "string" ||
      typeof rawValue === "number" ||
      typeof rawValue === "boolean"
    ) {
      params[key] = rawValue
    }
  }

  return Object.keys(params).length > 0 ? params : undefined
}

function shouldShowErrorToast(options: APIRequestOptions): boolean {
  return options.errorHandling === "toast"
}

function localizeBackendText(
  code: unknown,
  params: unknown,
  fallbackKey = "common.requestFailed"
): string {
  const normalizedCode = typeof code === "string" ? code.trim() : ""
  const normalizedParams = normalizeBackendParams(params)

  if (normalizedCode) {
    const translationKey = `common.backendMessages.${normalizedCode}`
    if (i18n.exists(translationKey)) {
      return normalizedParams ? i18n.t(translationKey, normalizedParams) : i18n.t(translationKey)
    }
  }

  return i18n.t(fallbackKey)
}

export function localizeBackendError(errorCode?: unknown, errorParams?: unknown, fallbackKey = "common.requestFailed"): string {
  return localizeBackendText(errorCode, errorParams, fallbackKey)
}

export function localizeBackendErrorResponse(payload?: BackendErrorPayload | null, fallbackKey = "common.requestFailed"): string {
  return localizeBackendError(payload?.error_code, payload?.error_params, fallbackKey)
}

export function localizeBackendMessage(
  messageCode?: unknown,
  messageParams?: unknown,
  fallbackKey = "common.requestFailed"
): string {
  return localizeBackendText(messageCode, messageParams, fallbackKey)
}

export function localizeBackendMessageResponse(
  payload?: BackendMessagePayload | null,
  fallbackKey = "common.requestFailed"
): string {
  return localizeBackendMessage(payload?.message_code, payload?.message_params, fallbackKey)
}

function createBackendAPIError(
  payload: BackendErrorPayload | null | undefined,
  status: number,
  fallbackKey = "common.requestFailed"
): BackendAPIError {
  return new BackendAPIError(localizeBackendErrorResponse(payload, fallbackKey), {
    code: typeof payload?.error_code === "string" ? payload.error_code : undefined,
    params: normalizeBackendParams(payload?.error_params),
    status,
  })
}

function buildHeaders(options: RequestInit): Headers {
  const headers = new Headers(options.headers)
  const hasBody = options.body !== undefined && options.body !== null
  const isFormDataBody =
    typeof FormData !== "undefined" && options.body instanceof FormData

  if (hasBody && !isFormDataBody && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const token = getToken()
  if (token) {
    headers.set("Authorization", `Bearer ${token}`)
  }

  return headers
}

async function parseJSON<T>(res: Response): Promise<T | null> {
  const contentType = res.headers.get("content-type") ?? ""
  if (!contentType.toLowerCase().includes("application/json")) {
    return null
  }

  try {
    return (await res.json()) as T
  } catch {
    return null
  }
}

async function performRefresh(): Promise<boolean> {
  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    })

    if (!res.ok) {
      return false
    }

    const data = await parseJSON<AuthResponse>(res)
    if (!data) {
      return false
    }

    const accessToken = resolveAccessToken(data)
    if (!accessToken || !data.user) {
      return false
    }

    setAuth(accessToken, data.user)
    return true
  } catch {
    return false
  }
}

async function performLogout(): Promise<void> {
  const res = await fetch(`${API_BASE}/auth/refresh/logout`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: "{}",
  })

  if (res.status === 204) {
    return
  }

  const data = await parseJSON<BackendErrorPayload>(res)
  if (!res.ok) {
    throw createBackendAPIError(data, res.status)
  }
}

async function performLogoutAll(): Promise<void> {
  const res = await requestRaw("/auth/logout-all", { method: "POST" })

  if (res.status === 204) {
    return
  }

  const data = await parseJSON<BackendErrorPayload>(res)
  if (!res.ok) {
    throw createBackendAPIError(data, res.status)
  }
}

async function refreshSession(): Promise<boolean> {
  if (!refreshRequest) {
    refreshRequest = performRefresh().finally(() => {
      refreshRequest = null
    })
  }
  return refreshRequest
}

async function requestRaw(
  path: string,
  options: APIRequestOptions = {},
  retryOnUnauthorized = true
): Promise<Response> {
  const fetchOptions = { ...options }
  delete fetchOptions.errorHandling
  delete fetchOptions.errorFallbackKey
  const token = getToken()
  const headers = buildHeaders(fetchOptions)

  const res = await fetch(`${API_BASE}${path}`, {
    ...fetchOptions,
    credentials: "include",
    headers,
  })

  if (res.status === 401) {
    if (retryOnUnauthorized && canRefresh(path, !!token) && (await refreshSession())) {
      return requestRaw(path, options, false)
    }
    return handleUnauthorized()
  }

  return res
}

async function requestResponse(
  path: string,
  options: APIRequestOptions = {},
  retryOnUnauthorized = true
): Promise<Response> {
  const res = await requestRaw(path, options, retryOnUnauthorized)

  if (!res.ok) {
    const data = await parseJSON<BackendErrorPayload>(res)
    const error = createBackendAPIError(data, res.status, options.errorFallbackKey)
    if (shouldShowErrorToast(options)) {
      toast.error(error.message)
    }
    throw error
  }

  return res
}

async function request<T>(
  path: string,
  options: APIRequestOptions = {},
  retryOnUnauthorized = true
): Promise<T> {
  const res = await requestResponse(path, options, retryOnUnauthorized)

  if (res.status === 204) {
    return undefined as T
  }

  const data = await parseJSON<T & BackendPayload>(res)

  if (!data) {
    const error = createBackendAPIError(null, res.status, options.errorFallbackKey)
    if (shouldShowErrorToast(options)) {
      toast.error(error.message)
    }
    throw error
  }

  return data as T
}

export async function logout(): Promise<void> {
  try {
    await performLogout()
  } finally {
    clearToken()
  }
}

export async function logoutAll(): Promise<void> {
  await performLogoutAll()
  clearToken()
}

export const api = {
  response: (path: string, options?: APIRequestOptions) => requestResponse(path, options),
  get: <T>(path: string, options: APIRequestOptions = {}) => request<T>(path, options),
  post: <T>(path: string, body: unknown, options: APIRequestOptions = {}) =>
    request<T>(path, { ...options, method: "POST", body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown, options: APIRequestOptions = {}) =>
    request<T>(path, { ...options, method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(path: string, options: APIRequestOptions = {}) =>
    request<T>(path, { ...options, method: "DELETE" }),
  uploadFile: async <T>(
    path: string,
    formData: FormData,
    options: APIRequestOptions = {},
    retryOnUnauthorized = true
  ): Promise<T> => request<T>(path, { ...options, method: "POST", body: formData }, retryOnUnauthorized),
}
