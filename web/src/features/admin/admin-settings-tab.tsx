import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { TabsContent } from "@/components/ui/tabs"

interface AdminSettingsTabProps {
  maxIconFileSize: number
  onMaxIconFileSizeChange: (value: number) => void
  onRegistrationEnabledChange: (enabled: boolean) => void
  onSave: () => void | Promise<void>
  onSiteNameChange: (value: string) => void
  onSiteUrlChange: (value: string) => void
  registrationEnabled: boolean
  siteName: string
  siteUrl: string
}

export default function AdminSettingsTab({
  maxIconFileSize,
  onMaxIconFileSizeChange,
  onRegistrationEnabledChange,
  onSave,
  onSiteNameChange,
  onSiteUrlChange,
  registrationEnabled,
  siteName,
  siteUrl,
}: AdminSettingsTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="settings">
      <Card>
        <CardContent className="space-y-6 p-6">
          <div className="space-y-2">
            <Label htmlFor="site-name">{t("admin.settings.siteName")}</Label>
            <Input
              id="site-name"
              value={siteName}
              onChange={(event) => onSiteNameChange(event.target.value)}
              placeholder="Subdux"
            />
            <p className="text-xs text-muted-foreground">{t("admin.settings.siteNameDescription")}</p>
          </div>

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="site-url">{t("admin.settings.siteUrl")}</Label>
            <Input
              id="site-url"
              type="url"
              value={siteUrl}
              onChange={(event) => onSiteUrlChange(event.target.value)}
              placeholder="https://example.com"
            />
            <p className="text-xs text-muted-foreground">{t("admin.settings.siteUrlDescription")}</p>
          </div>

          <Separator />

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label htmlFor="registration">{t("admin.settings.registrationEnabled")}</Label>
              <p className="text-sm text-muted-foreground">
                {t("admin.settings.registrationDescription")}
              </p>
            </div>
            <Switch
              id="registration"
              checked={registrationEnabled}
              onCheckedChange={onRegistrationEnabledChange}
            />
          </div>

          <Separator />

          <div className="space-y-2">
            <Label htmlFor="max-icon-size">{t("admin.settings.maxIconFileSize")}</Label>
            <div className="flex items-center gap-2">
              <Input
                id="max-icon-size"
                type="number"
                min={1}
                max={10240}
                step={1}
                className="w-32"
                value={maxIconFileSize}
                onChange={(event) => {
                  const next = parseInt(event.target.value, 10)
                  if (!Number.isNaN(next) && next >= 1) {
                    onMaxIconFileSizeChange(next)
                  }
                }}
              />
              <span className="text-sm text-muted-foreground">
                {t("admin.settings.maxIconFileSizeUnit")}
              </span>
            </div>
            <p className="text-xs text-muted-foreground">
              {t("admin.settings.maxIconFileSizeDescription")}
            </p>
          </div>

          <Separator />

          <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
        </CardContent>
      </Card>
    </TabsContent>
  )
}
