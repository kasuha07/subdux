export type IconProxyProvider = "google" | "icon-horse"

const managedIconProxyPathPrefix = "/api/icon-proxy/"
const allowedIconProxyProviders = new Set<IconProxyProvider>(["google", "icon-horse"])

export function buildIconProxySuggestionURL(provider: IconProxyProvider, domain: string): string {
  const params = new URLSearchParams({ domain })
  return `${managedIconProxyPathPrefix}${provider}?${params.toString()}`
}

export function isManagedIconProxyURL(value: string): boolean {
  if (!value.startsWith(managedIconProxyPathPrefix)) {
    return false
  }

  try {
    const parsed = new URL(value, "https://subdux.local")
    if (parsed.origin !== "https://subdux.local") {
      return false
    }

    const provider = parsed.pathname.slice(managedIconProxyPathPrefix.length) as IconProxyProvider
    if (!allowedIconProxyProviders.has(provider)) {
      return false
    }

    const domain = parsed.searchParams.get("domain")?.trim().toLowerCase() ?? ""
    if (!domain || domain.includes("://") || domain.includes("/")) {
      return false
    }

    return domain.includes(".")
  } catch {
    return false
  }
}

export function isDirectExternalImageURL(value: string): boolean {
  return value.startsWith("http://") || value.startsWith("https://")
}

export function isRenderableImageURL(value: string): boolean {
  return isDirectExternalImageURL(value) || isManagedIconProxyURL(value)
}
