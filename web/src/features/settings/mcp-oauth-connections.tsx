import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { api } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { CopyButton } from "@/components/copy-button"

interface OAuthGrant {
  id: number
  client_id: string
  client_name: string
  scope: string
  expires_at: string
  last_used_at: string | null
}

export default function MCPOAuthConnections({ active }: { active: boolean }) {
  const { t } = useTranslation()
  const [grants, setGrants] = useState<OAuthGrant[]>([])
  const [info, setInfo] = useState<{ enabled: boolean; configured: boolean; url: string } | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(false)
  const [revoking, setRevoking] = useState<number | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(false)
    try {
      const [connections, config] = await Promise.all([api.get<OAuthGrant[]>("/mcp/oauth/grants"), api.get<{ enabled: boolean; configured: boolean; url: string }>("/mcp/oauth/info")])
      setGrants(connections)
      setInfo(config)
    } catch { setError(true) }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { if (active) void load() }, [active, load])

  async function revoke(id: number) {
    if (revoking !== null || !window.confirm(t("settings.mcpOAuth.revokeConfirm"))) return
    setRevoking(id)
    try {
      await api.delete(`/mcp/oauth/grants/${id}`, { errorHandling: "toast" })
      setGrants((prev) => prev.filter((grant) => grant.id !== id))
    } catch { /* API helper displays the error. */ }
    finally { setRevoking(null) }
  }

  return (
    <section className="space-y-3 rounded-md border p-4">
      <h3 className="text-sm font-semibold">{t("settings.mcpOAuth.title")}</h3>
      <p className="text-sm text-muted-foreground">{t("settings.mcpOAuth.description")}</p>
      {loading ? <p role="status" className="text-sm">{t("common.loading")}</p> : error ? <div role="alert" className="space-y-2">
        <p className="text-sm text-destructive">{t("settings.mcpOAuth.loadFailed")}</p>
        <Button size="sm" variant="outline" onClick={() => void load()}>{t("settings.mcpOAuth.retry")}</Button>
      </div> : <>
        {info?.enabled && info.configured ? <div className="flex items-center gap-2 rounded-md bg-muted p-2">
          <code className="min-w-0 flex-1 break-all text-xs">{info.url}</code><CopyButton text={info.url} label={t("common.copy")} copiedLabel={t("common.copied")} />
        </div> : <p className="text-sm text-muted-foreground">{t("settings.mcpOAuth.configureHint")}</p>}
        {grants.length === 0 && <p className="text-sm text-muted-foreground">{t("settings.mcpOAuth.empty")}</p>}
        {grants.map((grant) => <div key={grant.id} className="flex items-start justify-between gap-3 rounded-md border p-3">
          <div className="min-w-0 space-y-1 text-sm">
            <p className="font-medium break-words">{grant.client_name}</p>
            <p className="break-all text-xs text-muted-foreground">{grant.client_id}</p>
            <p>{grant.scope.split(" ").includes("write") ? t("settings.apiKeys.scopeReadWrite") : t("settings.apiKeys.scopeReadOnly")}</p>
            <p className="text-xs text-muted-foreground">{t("settings.mcpOAuth.lastUsed", { date: grant.last_used_at ? new Date(grant.last_used_at).toLocaleString() : t("settings.apiKeys.never") })}</p>
            <p className="text-xs text-muted-foreground">{t("settings.mcpOAuth.expires", { date: new Date(grant.expires_at).toLocaleString() })}</p>
          </div>
          <Button size="sm" variant="outline" disabled={revoking !== null} onClick={() => void revoke(grant.id)}>{t("settings.mcpOAuth.revoke")}</Button>
        </div>)}
      </>}
    </section>
  )
}
