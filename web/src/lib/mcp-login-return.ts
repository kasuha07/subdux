const RETURN_KEY = "subdux-mcp-login-return"

// OAuth may cross an OIDC login redirect. Store only a bounded, local consent
// path, never a client redirect URL or any access/refresh token.
export function safeMCPReturnPath(raw: string | null): string | null {
  if (!raw || !/^\/connect\/mcp\?request=[A-Za-z0-9_-]{43}$/.test(raw)) return null
  return raw
}

export function getLoginReturnPath(): string {
  const next = safeMCPReturnPath(new URLSearchParams(window.location.search).get("next"))
  if (next) return next
  try { return safeMCPReturnPath(sessionStorage.getItem(RETURN_KEY)) ?? "/" } catch { return "/" }
}

export function rememberLoginReturnPath(): void {
  const path = getLoginReturnPath()
  try {
    if (path !== "/") sessionStorage.setItem(RETURN_KEY, path)
    else sessionStorage.removeItem(RETURN_KEY)
  } catch { /* Login still works when browser storage is unavailable. */ }
}

export function clearLoginReturnPath(): void {
  try { sessionStorage.removeItem(RETURN_KEY) } catch { /* No persisted return path. */ }
}
