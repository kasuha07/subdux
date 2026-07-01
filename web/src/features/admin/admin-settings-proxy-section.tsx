import { useState } from "react"
import { useTranslation } from "react-i18next"

import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"

import type { AdminSettingsProxySectionProps } from "./admin-settings-types"

export default function AdminSettingsProxySection({
  onSystemProxyEnabledChange,
  onSystemProxyTypeChange,
  onSystemProxyUrlChange,
  systemProxyEnabled,
  systemProxyType,
  systemProxyUrl,
  systemProxyUrlConfigured,
}: AdminSettingsProxySectionProps) {
  const { t } = useTranslation()
  const [editingSystemProxyUrl, setEditingSystemProxyUrl] = useState(false)
  const configuredMaskValue = "••••••••"
  const systemProxyUrlDisplayValue = editingSystemProxyUrl
    ? systemProxyUrl
    : systemProxyUrl || (systemProxyUrlConfigured ? configuredMaskValue : "")

  return (
    <>
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-0.5">
          <Label htmlFor="system-proxy-enabled">{t("admin.settings.systemProxyEnabled")}</Label>
          <p className="text-sm text-muted-foreground">
            {t("admin.settings.systemProxyEnabledDescription")}
          </p>
        </div>
        <Switch
          id="system-proxy-enabled"
          checked={systemProxyEnabled}
          onCheckedChange={onSystemProxyEnabledChange}
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-[180px_minmax(0,1fr)]">
        <div className="space-y-2">
          <Label htmlFor="system-proxy-type">{t("admin.settings.systemProxyType")}</Label>
          <Select value={systemProxyType} onValueChange={onSystemProxyTypeChange}>
            <SelectTrigger id="system-proxy-type" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="http">{t("admin.settings.systemProxyTypeHTTP")}</SelectItem>
              <SelectItem value="socks5">{t("admin.settings.systemProxyTypeSOCKS5")}</SelectItem>
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label htmlFor="system-proxy-url">{t("admin.settings.systemProxyUrl")}</Label>
          <Input
            id="system-proxy-url"
            value={systemProxyUrlDisplayValue}
            onFocus={() => setEditingSystemProxyUrl(true)}
            onBlur={() => setEditingSystemProxyUrl(false)}
            onChange={(event) => onSystemProxyUrlChange(event.target.value)}
            placeholder={t("admin.settings.secretNotConfigured")}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.settings.systemProxyUrlDescription")}
          </p>
        </div>
      </div>
    </>
  )
}
