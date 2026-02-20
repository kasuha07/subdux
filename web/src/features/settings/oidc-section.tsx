import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api"
import type { OIDCConfig, OIDCConnection, OIDCSessionResult, OIDCStartResponse } from "@/types"

export default function OIDCSection() {
  const { t } = useTranslation()

  const [config, setConfig] = useState<OIDCConfig | null>(null)
  const [connections, setConnections] = useState<OIDCConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [processingCallback, setProcessingCallback] = useState(false)
  const [starting, setStarting] = useState(false)
  const [deletingID, setDeletingID] = useState<number | null>(null)
  const [error, setError] = useState("")

  const providerName = useMemo(() => config?.provider_name || "OIDC", [config?.provider_name])

  useEffect(() => {
    Promise.all([
      api.get<OIDCConfig>("/auth/oidc/config"),
      api.get<OIDCConnection[]>("/auth/oidc/connections"),
    ])
      .then(([configData, connectionData]) => {
        setConfig(configData)
        setConnections(connectionData ?? [])
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const action = params.get("oidc_action")
    const sessionID = params.get("oidc_session")
    if (action !== "connect" || !sessionID) {
      return
    }

    setProcessingCallback(true)
    api.get<OIDCSessionResult>(`/auth/oidc/session/${encodeURIComponent(sessionID)}`)
      .then((result) => {
        if (result.error) {
          setError(result.error)
          return
        }

        if (result.connection) {
          setConnections([result.connection])
          toast.success(t("settings.oidc.connectSuccess"))
        }
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : t("settings.oidc.connectError"))
      })
      .finally(() => {
        const nextURL = new URL(window.location.href)
        nextURL.searchParams.delete("oidc_action")
        nextURL.searchParams.delete("oidc_session")
        const query = nextURL.searchParams.toString()
        const search = query ? `?${query}` : ""
        window.history.replaceState({}, "", `${nextURL.pathname}${search}${nextURL.hash}`)
        setProcessingCallback(false)
      })
  }, [t])

  async function refreshConnections() {
    const latest = await api.get<OIDCConnection[]>("/auth/oidc/connections")
    setConnections(latest ?? [])
  }

  async function handleConnect() {
    setError("")
    setStarting(true)
    try {
      const result = await api.post<OIDCStartResponse>("/auth/oidc/connect/start", {})
      window.location.href = result.authorization_url
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.oidc.connectError"))
      setStarting(false)
    }
  }

  async function handleDisconnect(connection: OIDCConnection) {
    if (!window.confirm(t("settings.oidc.disconnectConfirm"))) {
      return
    }

    setError("")
    setDeletingID(connection.id)
    try {
      await api.delete(`/auth/oidc/connections/${connection.id}`)
      await refreshConnections()
      toast.success(t("settings.oidc.disconnectSuccess"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settings.oidc.disconnectError"))
    } finally {
      setDeletingID(null)
    }
  }

  const connected = connections.length > 0
  const enabled = config?.enabled ?? false

  return (
    <div className="space-y-3">
      <div>
        <div className="flex items-center gap-2">
          <h3 className="text-sm font-medium">{t("settings.oidc.title")}</h3>
          <Badge variant="secondary" className="text-xs">{connections.length}</Badge>
        </div>
        <p className="mt-0.5 text-sm text-muted-foreground">{t("settings.oidc.description")}</p>
      </div>

      {loading && <p className="text-sm text-muted-foreground">{t("common.loading")}</p>}

      {!loading && !enabled && (
        <p className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
          {t("settings.oidc.disabled")}
        </p>
      )}

      {error && (
        <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      {processingCallback && (
        <p className="text-sm text-muted-foreground">{t("settings.oidc.processing")}</p>
      )}

      {enabled && (
        <div>
          <Button
            size="sm"
            variant="outline"
            disabled={starting || processingCallback || connected}
            onClick={() => void handleConnect()}
          >
            {starting
              ? t("settings.oidc.connecting")
              : t("settings.oidc.connectButton", { provider: providerName })}
          </Button>
          {connected && (
            <p className="mt-2 text-xs text-muted-foreground">
              {t("settings.oidc.alreadyConnected")}
            </p>
          )}
        </div>
      )}

      <div className="space-y-2">
        {!loading && connections.length === 0 && (
          <p className="text-sm text-muted-foreground">{t("settings.oidc.empty")}</p>
        )}
        {connections.map((connection) => (
          <div key={connection.id} className="flex items-start justify-between gap-3 rounded-md border bg-card px-3 py-2">
            <div className="min-w-0">
              <p className="text-sm font-medium">{providerName}</p>
              <p className="text-xs text-muted-foreground">{connection.email || "-"}</p>
            </div>
            <Button
              size="sm"
              variant="ghost"
              className="text-destructive hover:text-destructive"
              disabled={deletingID === connection.id}
              onClick={() => void handleDisconnect(connection)}
            >
              {deletingID === connection.id ? t("settings.oidc.disconnecting") : t("settings.oidc.disconnect")}
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}
