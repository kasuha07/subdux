import { useEffect } from "react"

export function useSiteTitle() {
  useEffect(() => {
    fetch("/api/site-info", { credentials: "include" })
      .then(async (response) => {
        if (!response.ok) {
          return
        }
        const data = (await response.json()) as { site_name?: string }
        if (data?.site_name) {
          document.title = data.site_name
        }
      })
      .catch(() => void 0)
  }, [])

  return null
}

export function updateSiteTitle(siteName: string) {
  document.title = siteName || "Subdux"
}
