import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { SystemSettings } from "@/types"

export function useSiteSettings() {
  const [settings, setSettings] = useState<SystemSettings | null>(null)

  useEffect(() => {
    const token = localStorage.getItem("token")
    if (!token) return

    api.get<SystemSettings>("/admin/settings")
      .then((data) => {
        setSettings(data)
        if (data?.site_name) {
          document.title = data.site_name
        }
      })
      .catch(() => void 0)
  }, [])

  return settings
}

export function updateSiteTitle(siteName: string) {
  document.title = siteName || "Subdux"
}
