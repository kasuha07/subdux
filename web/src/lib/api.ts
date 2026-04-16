import { toast } from "sonner"
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

const BACKEND_ERROR_TRANSLATIONS: Record<string, string> = {
  "you can enable at most 3 notification channels": "common.backendErrors.maxNotificationChannels",
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
    cachedAccessToken = readLocalStorage(ACCESS_TOKEN_KEY)
    accessTokenLoaded = true
  }

  return cachedAccessToken
}

export function setToken(token: string): void {
  cachedAccessToken = token
  accessTokenLoaded = true
  writeLocalStorage(ACCESS_TOKEN_KEY, token)
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
  toast.error(i18n.t("common.unauthorized"))
  window.location.href = "/login"
  throw new Error(i18n.t("common.unauthorized"))
}

function canRefresh(path: string, hasAccessToken: boolean): boolean {
  return hasAccessToken && path !== "/auth/refresh"
}

function localizeBackendError(error: unknown): string {
  if (typeof error !== "string" || !error) {
    return i18n.t("common.requestFailed")
  }

  const translationKey = BACKEND_ERROR_TRANSLATIONS[error]
  return translationKey ? i18n.t(translationKey) : error
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

  const data = await parseJSON<{ error?: unknown }>(res)
  if (!res.ok) {
    throw new Error(localizeBackendError(data?.error))
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
  options: RequestInit = {},
  retryOnUnauthorized = true
): Promise<Response> {
  const token = getToken()
  const headers = buildHeaders(options)

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
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

async function request<T>(
  path: string,
  options: RequestInit = {},
  retryOnUnauthorized = true
): Promise<T> {
  const res = await requestRaw(path, options, retryOnUnauthorized)

  if (res.status === 204) {
    return undefined as T
  }

  const data = await parseJSON<T & { error?: unknown }>(res)

  if (!res.ok) {
    const errorMsg = localizeBackendError(data?.error)
    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  if (!data) {
    const errorMsg = i18n.t("common.requestFailed")
    toast.error(errorMsg)
    throw new Error(errorMsg)
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

export const api = {
  fetch: (path: string, options?: RequestInit) => requestRaw(path, options),
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "POST", body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  uploadFile: async <T>(path: string, formData: FormData, retryOnUnauthorized = true): Promise<T> => {
    const res = await requestRaw(path, { method: "POST", body: formData }, retryOnUnauthorized)

    if (res.status === 204) {
      return undefined as T
    }

    const data = await parseJSON<T & { error?: unknown }>(res)

    if (!res.ok) {
      const errorMsg = localizeBackendError(data?.error)
      toast.error(errorMsg)
      throw new Error(errorMsg)
    }

    if (!data) {
      const errorMsg = i18n.t("common.requestFailed")
      toast.error(errorMsg)
      throw new Error(errorMsg)
    }

    return data as T
  },
}
