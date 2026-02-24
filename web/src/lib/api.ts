import { toast } from "sonner"
import i18n from "@/i18n"
import type { AuthResponse, User } from "@/types"

const API_BASE = "/api"
const ACCESS_TOKEN_KEY = "token"
const REFRESH_TOKEN_KEY = "refresh_token"

let refreshRequest: Promise<boolean> | null = null
const BACKEND_ERROR_TRANSLATIONS: Record<string, string> = {
  "you can enable at most 3 notification channels": "common.backendErrors.maxNotificationChannels",
}

function getToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, token)
}

export function setRefreshToken(token: string): void {
  localStorage.setItem(REFRESH_TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem("user")
}

export function isAuthenticated(): boolean {
  return !!getToken()
}

export function setUser(user: User): void {
  localStorage.setItem("user", JSON.stringify(user))
}

export function getUser(): User | null {
  const raw = localStorage.getItem("user")
  if (!raw) return null
  try {
    return JSON.parse(raw) as User
  } catch {
    return null
  }
}

export function isAdmin(): boolean {
  return getUser()?.role === "admin"
}

export function setAuth(token: string, user: User, refreshToken?: string): void {
  setToken(token)
  if (refreshToken) {
    setRefreshToken(refreshToken)
  } else {
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  }
  setUser(user)
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
  return hasAccessToken && path !== "/auth/refresh" && !!getRefreshToken()
}

function localizeBackendError(error: unknown): string {
  if (typeof error !== "string" || !error) {
    return i18n.t("common.requestFailed")
  }

  const translationKey = BACKEND_ERROR_TRANSLATIONS[error]
  return translationKey ? i18n.t(translationKey) : error
}

async function performRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false

  try {
    const res = await fetch(`${API_BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })

    if (!res.ok) {
      return false
    }

    const data = await res.json() as AuthResponse
    const accessToken = resolveAccessToken(data)
    if (!accessToken || !data.user) {
      return false
    }

    setAuth(accessToken, data.user, data.refresh_token)
    return true
  } catch {
    return false
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

async function request<T>(
  path: string,
  options: RequestInit = {},
  retryOnUnauthorized = true
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })

  if (res.status === 401) {
    if (retryOnUnauthorized && canRefresh(path, !!token) && (await refreshSession())) {
      return request<T>(path, options, false)
    }
    return handleUnauthorized()
  }

  if (res.status === 204) {
    return undefined as T
  }

  const data = await res.json()

  if (!res.ok) {
    const errorMsg = localizeBackendError(data.error)
    toast.error(errorMsg)
    throw new Error(errorMsg)
  }

  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "POST", body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  uploadFile: async <T>(path: string, formData: FormData, retryOnUnauthorized = true): Promise<T> => {
    const token = getToken()
    const headers: Record<string, string> = {}
    if (token) {
      headers["Authorization"] = `Bearer ${token}`
    }
    const res = await fetch(`${API_BASE}${path}`, {
      method: "POST",
      headers,
      body: formData,
    })
    if (res.status === 401) {
      if (retryOnUnauthorized && canRefresh(path, !!token) && (await refreshSession())) {
        return api.uploadFile<T>(path, formData, false)
      }
      return handleUnauthorized()
    }
    if (res.status === 204) {
      return undefined as T
    }
    const data = await res.json()
    if (!res.ok) {
      const errorMsg = localizeBackendError(data.error)
      toast.error(errorMsg)
      throw new Error(errorMsg)
    }
    return data as T
  },
}
