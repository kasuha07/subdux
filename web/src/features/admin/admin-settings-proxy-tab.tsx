import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"

import AdminSettingsProxySection from "./admin-settings-proxy-section"
import type { AdminSettingsProxyTabProps } from "./admin-settings-types"

export default function AdminSettingsProxyTab({
  onSave,
  onSystemProxyEnabledChange,
  onSystemProxyTypeChange,
  onSystemProxyUrlChange,
  systemProxyEnabled,
  systemProxyType,
  systemProxyUrl,
  systemProxyUrlConfigured,
}: AdminSettingsProxyTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="proxy" className="space-y-6">
      <AdminSettingsProxySection
        onSystemProxyEnabledChange={onSystemProxyEnabledChange}
        onSystemProxyTypeChange={onSystemProxyTypeChange}
        onSystemProxyUrlChange={onSystemProxyUrlChange}
        systemProxyEnabled={systemProxyEnabled}
        systemProxyType={systemProxyType}
        systemProxyUrl={systemProxyUrl}
        systemProxyUrlConfigured={systemProxyUrlConfigured}
      />

      <Separator />

      <Button onClick={() => void onSave()}>{t("admin.settings.save")}</Button>
    </TabsContent>
  )
}
