import { useEffect, useState } from "react"
import { Link, useSearchParams } from "react-router"
import { useTranslation } from "react-i18next"
import { api, getUser } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Label } from "@/components/ui/label"

interface ConsentInfo {
  client_id: string
  client_name: string
  redirect_uri: string
  scopes: string[]
}

export default function MCPConsentPage() {
  const [params] = useSearchParams()
  const request = params.get("request") ?? ""
  // A new interaction must start with fresh client details and unchecked write
  // permission, including when the router changes the URL without a reload.
  return <MCPConsentRequest key={request} request={request} />
}

function MCPConsentRequest({ request }: { request: string }) {
  const { t } = useTranslation()
  const [info, setInfo] = useState<ConsentInfo | null>(null)
  const [error, setError] = useState("")
  const [busy, setBusy] = useState(false)
  const [allowWrite, setAllowWrite] = useState(false)

  useEffect(() => {
    let cancelled = false
    if (!/^[A-Za-z0-9_-]{43}$/.test(request)) {
      return
    }
    api.get<ConsentInfo>(`/mcp/oauth/requests/${request}`)
      .then((result) => { if (!cancelled) setInfo(result) })
      .catch(() => { if (!cancelled) setError(t("settings.mcpOAuth.invalidRequest")) })
    return () => { cancelled = true }
  }, [request, t])

  async function decide(approve: boolean) {
    if (busy || !info) return
    setBusy(true)
    setError("")
    try {
      const result = await api.post<{ redirect_uri: string }>(`/mcp/oauth/requests/${request}`, { approve, allow_write: allowWrite })
      // The backend binds this URI to validated client metadata and the
      // displayed interaction. Tokens are exchanged by the client, not here.
      window.location.replace(result.redirect_uri)
    } catch {
      setError(t("settings.mcpOAuth.invalidRequest"))
      setBusy(false)
    }
  }

  const invalid = !/^[A-Za-z0-9_-]{43}$/.test(request)
  return (
    <div className="flex min-h-screen items-center justify-center px-4 py-8">
      <Card className="w-full max-w-md">
        <CardHeader>
          <CardTitle>{t("settings.mcpOAuth.consentTitle")}</CardTitle>
          <CardDescription>{t("settings.mcpOAuth.consentDescription")}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {(error || invalid) && <p role="alert" className="text-sm text-destructive">{error || t("settings.mcpOAuth.invalidRequest")}</p>}
          {!info && !error && !invalid && <p role="status">{t("common.loading")}</p>}
          {info && <>
            <div className="space-y-1 text-sm">
              <p className="font-medium break-words">{info.client_name}</p>
              <p className="break-all text-muted-foreground">{info.client_id}</p>
              <p>{t("settings.mcpOAuth.signedInAs", { name: getUser()?.username ?? "" })}</p>
            </div>
            <p className="text-sm">{t("settings.mcpOAuth.readPermission")}</p>
            {info.scopes.includes("write") && <div className="flex items-start gap-2">
              <Checkbox id="mcp-write" checked={allowWrite} disabled={busy} onCheckedChange={(checked) => setAllowWrite(checked === true)} />
              <Label htmlFor="mcp-write" className="text-sm leading-relaxed">{t("settings.mcpOAuth.writePermission")}</Label>
            </div>}
            <p className="text-xs text-muted-foreground">{t("settings.mcpOAuth.accessDescription")}</p>
            <p className="break-all text-xs text-muted-foreground">{t("settings.mcpOAuth.redirectTo", { uri: info.redirect_uri })}</p>
            <div className="flex gap-2">
              <Button className="flex-1" disabled={busy} onClick={() => void decide(true)}>{busy ? t("common.loading") : t("settings.mcpOAuth.authorize")}</Button>
              <Button variant="outline" disabled={busy} onClick={() => void decide(false)}>{t("settings.mcpOAuth.deny")}</Button>
            </div>
          </>}
          <Button variant="ghost" asChild><Link to="/">{t("settings.mcpOAuth.back")}</Link></Button>
        </CardContent>
      </Card>
    </div>
  )
}
