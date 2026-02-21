import { useTranslation } from "react-i18next"
import { AlertTriangle, Download } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { TabsContent } from "@/components/ui/tabs"

interface AdminBackupTabProps {
  includeAssetsInBackup: boolean
  onDownloadBackup: () => void | Promise<void>
  onIncludeAssetsInBackupChange: (value: boolean) => void
  onRestore: () => void | Promise<void>
  onRestoreConfirmOpenChange: (open: boolean) => void
  onRestoreFileChange: (file: File | null) => void
  restoreConfirmOpen: boolean
  restoreFile: File | null
}

export default function AdminBackupTab({
  includeAssetsInBackup,
  onDownloadBackup,
  onIncludeAssetsInBackupChange,
  onRestore,
  onRestoreConfirmOpenChange,
  onRestoreFileChange,
  restoreConfirmOpen,
  restoreFile,
}: AdminBackupTabProps) {
  const { t } = useTranslation()

  return (
    <TabsContent value="backup" className="space-y-4">
      <Card>
        <CardContent className="p-6">
          <h3 className="text-sm font-medium">{t("admin.backup.download")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("admin.backup.downloadDescription")}
          </p>
          <div className="mt-4 flex items-center justify-between gap-4 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label htmlFor="backup-include-assets">{t("admin.backup.includeAssets")}</Label>
              <p className="text-xs text-muted-foreground">
                {t("admin.backup.includeAssetsDescription")}
              </p>
            </div>
            <Switch
              id="backup-include-assets"
              checked={includeAssetsInBackup}
              onCheckedChange={onIncludeAssetsInBackupChange}
            />
          </div>
          <Button variant="outline" className="mt-4" onClick={() => void onDownloadBackup()}>
            <Download className="size-4" />
            {t("admin.backup.downloadButton")}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4 p-6">
          <div>
            <h3 className="text-sm font-medium">{t("admin.backup.restore")}</h3>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {t("admin.backup.restoreDescription")}
            </p>
          </div>
          <Input
            type="file"
            accept=".db,.zip"
            onChange={(event) => onRestoreFileChange(event.target.files?.[0] ?? null)}
          />
          <Button
            variant="destructive"
            disabled={!restoreFile}
            onClick={() => onRestoreConfirmOpenChange(true)}
          >
            {t("admin.backup.restoreButton")}
          </Button>

          {restoreConfirmOpen && (
            <div className="rounded-md border border-destructive bg-destructive/10 p-4">
              <div className="mb-2 flex items-center gap-2 font-medium text-destructive">
                <AlertTriangle className="size-4" />
                {t("admin.backup.restoreConfirm")}
              </div>
              <div className="mt-3 flex gap-2">
                <Button size="sm" variant="destructive" onClick={() => void onRestore()}>
                  {t("admin.backup.confirm")}
                </Button>
                <Button size="sm" variant="outline" onClick={() => onRestoreConfirmOpenChange(false)}>
                  {t("admin.backup.cancel")}
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  )
}
