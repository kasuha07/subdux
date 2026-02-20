import i18n from "@/i18n"
import type { User } from "@/types"

const API_BASE = "/api"

function getToken(): string | null {
  return localStorage.getItem("token")
}

export function setToken(token: string): void {
  localStorage.setItem("token", token)
}

export function clearToken(): void {
  localStorage.removeItem("token")
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

export function setAuth(token: string, user: User): void {
  setToken(token)
  setUser(user)
}

async function request<T>(
  path: string,
  options: RequestInit = {}
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
    clearToken()
    window.location.href = "/login"
    throw new Error(i18n.t("common.unauthorized"))
  }

  if (res.status === 204) {
    return undefined as T
  }

  const data = await res.json()

  if (!res.ok) {
    throw new Error(data.error || i18n.t("common.requestFailed"))
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
}
