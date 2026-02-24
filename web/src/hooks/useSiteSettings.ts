import { useEffect, useState } from "react"
import { api, isAdmin } from "@/lib/api"
import type { SystemSettings } from "@/types"

export function useSiteSettings() {
  const [settings, setSettings] = useState<SystemSettings | null>(null)

  useEffect(() => {
    const token = localStorage.getItem("token")
    if (!token) return

    // Fetch site name for all users via public endpoint
    api.get<{ site_name: string }>("/site-info")
      .then((data) => {
        if (data?.site_name) {
          document.title = data.site_name
        }
      })
      .catch(() => void 0)

    // Fetch full settings for admin users
    if (!isAdmin()) return
    api.get<SystemSettings>("/admin/settings")
      .then((data) => {
        setSettings(data)
      })
      .catch(() => void 0)
  }, [])

  return settings
}

export function updateSiteTitle(siteName: string) {
  document.title = siteName || "Subdux"
}
