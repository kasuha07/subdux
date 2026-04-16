import { useEffect, useState } from "react"
import { api, isAdmin, isAuthenticated } from "@/lib/api"
import type { SystemSettings } from "@/types"

export function useSiteSettings() {
  const [settings, setSettings] = useState<SystemSettings | null>(null)
  const authenticated = isAuthenticated()
  const admin = isAdmin()

  useEffect(() => {
    if (!authenticated) {
      return
    }

    // Fetch site name for all users via public endpoint
    api.get<{ site_name: string }>("/site-info")
      .then((data) => {
        if (data?.site_name) {
          document.title = data.site_name
        }
      })
      .catch(() => void 0)

    // Fetch full settings for admin users
    if (!admin) {
      return
    }
    api.get<SystemSettings>("/admin/settings")
      .then((data) => {
        setSettings(data)
      })
      .catch(() => void 0)
  }, [admin, authenticated])

  if (!authenticated || !admin) {
    return null
  }

  return settings
}

export function updateSiteTitle(siteName: string) {
  document.title = siteName || "Subdux"
}
