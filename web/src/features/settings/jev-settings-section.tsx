import { useEffect, useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { api, getAPIErrorMessage } from "@/lib/api"
import { toast } from "@/lib/toast"
import type { JevSettings } from "@/types/jev"

export function JevSettingsSection() {
  const { t } = useTranslation()
  const [settings, setSettings] = useState<JevSettings | null>(null)
  const [enabled, setEnabled] = useState(false)
  const [apiKey, setAPIKey] = useState("")
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState(false)
  const [reload, setReload] = useState(0)

  useEffect(() => {
    const controller = new AbortController()
    api.get<JevSettings>("/jev/settings", { signal: controller.signal }).then((value) => {
      if (controller.signal.aborted) return
      setSettings(value)
      setEnabled(value.enabled)
      setLoadError(false)
    }).catch(() => {
      if (!controller.signal.aborted) setLoadError(true)
    })
    return () => controller.abort()
  }, [reload])

  async function save(removeAPIKey = false) {
    if (!settings) return
    setSaving(true)
    try {
      const value = await api.put<JevSettings>("/jev/settings", {
        revision: settings.revision,
        enabled: removeAPIKey ? false : enabled,
        api_key: removeAPIKey ? "" : apiKey.trim(),
        remove_api_key: removeAPIKey,
      })
      setSettings(value)
      setEnabled(value.enabled)
      setAPIKey("")
      toast.success(t("settings.jev.saved"))
    } catch (error) {
      toast.error(getAPIErrorMessage(error))
    } finally {
      setSaving(false)
    }
  }

  function submit(event: FormEvent) {
    event.preventDefault()
    void save()
  }

  return (
    <section className="space-y-4">
      <div>
        <h3 className="text-sm font-medium">{t("settings.jev.title")}</h3>
        <p className="mt-1 text-sm text-muted-foreground">{t("settings.jev.description")}</p>
      </div>
      {loadError ? (
        <div role="alert" className="flex items-center gap-3 text-sm">
          <span>{t("settings.jev.loadFailed")}</span>
          <Button variant="outline" size="sm" onClick={() => setReload((value) => value + 1)}>{t("settings.jev.retry")}</Button>
        </div>
      ) : !settings ? <p className="text-sm text-muted-foreground">{t("common.loading")}</p> : (
        <form onSubmit={submit} className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <Label htmlFor="jev-enabled">{t("settings.jev.enabled")}</Label>
            <Switch id="jev-enabled" checked={enabled} onCheckedChange={setEnabled} disabled={saving} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="jev-api-key">{t("settings.jev.apiKey")}</Label>
            <Input id="jev-api-key" type="password" autoComplete="new-password" spellCheck={false}
              value={apiKey} onChange={(event) => setAPIKey(event.target.value)} maxLength={512} disabled={saving}
              placeholder={t(settings.api_key_configured ? "settings.jev.keyConfigured" : "settings.jev.keyPlaceholder")} />
            <p className="text-xs text-muted-foreground">{t("settings.jev.privacy")}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button type="submit" size="sm" disabled={saving || (enabled && !settings.api_key_configured && !apiKey.trim())}>
              {t(saving ? "settings.jev.saving" : "settings.jev.save")}
            </Button>
            {settings.api_key_configured && (
              <Button type="button" variant="outline" size="sm" disabled={saving} onClick={() => void save(true)}>
                {t("settings.jev.removeKey")}
              </Button>
            )}
          </div>
        </form>
      )}
    </section>
  )
}
