import { useState } from "react"
import { useTranslation } from "react-i18next"
import {
  AlertTriangle,
  Clock,
  Download,
  Lock,
  PlayCircle,
  RefreshCw,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { TabsContent } from "@/components/ui/tabs"
import type { LocalBackupInfo } from "@/types"

interface AdminBackupTabProps {
  backupEncryptEnabled: boolean
  backupEncryptionPassword: string
  backupEncryptionPasswordConfigured: boolean
  backupIncludeAssets: boolean
  backupLocalDir: string
  backupRetentionCount: number
  backupScheduleEnabled: boolean
  backupTimeOfDay: string
  downloadPassword: string
  includeAssetsInBackup: boolean
  lastRunAt: string
  lastError: string
  lastStatus: string
  localBackupDir: string
  localBackups: LocalBackupInfo[]
  localBackupsRefreshing: boolean
  onBackupEncryptEnabledChange: (value: boolean) => void
  onBackupEncryptionPasswordChange: (value: string) => void
  onBackupIncludeAssetsChange: (value: boolean) => void
  onBackupLocalDirChange: (value: string) => void
  onBackupRetentionCountChange: (value: number) => void
  onBackupScheduleEnabledChange: (value: boolean) => void
  onBackupTimeOfDayChange: (value: string) => void
  onDownloadBackup: () => void | Promise<void>
  onDownloadPasswordChange: (value: string) => void
  onIncludeAssetsInBackupChange: (value: boolean) => void
  onRefreshLocalBackups: () => void | Promise<void>
  onRestore: () => void | Promise<void>
  onRestoreConfirmOpenChange: (open: boolean) => void
  onRestoreFileChange: (file: File | null) => void
  onRestorePasswordChange: (value: string) => void
  onRunBackupNow: () => void | Promise<void>
  onSaveSettings: () => void | Promise<void>
  restoreConfirmOpen: boolean
  restoreEncrypted: boolean
  restoreFile: File | null
  restorePassword: string
  runningBackup: boolean
}

function formatBytes(bytes: number, locale: string): string {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return "0 B"
  }

  const units = ["B", "KB", "MB", "GB", "TB"]
  const exponent = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1
  )
  const value = bytes / Math.pow(1024, exponent)
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: exponent === 0 ? 0 : 1,
  }).format(value)

  return `${formatted} ${units[exponent]}`
}

function formatDateTime(value: string, locale: string): string {
  if (!value) {
    return "—"
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  })
}

export default function AdminBackupTab({
  backupEncryptEnabled,
  backupEncryptionPassword,
  backupEncryptionPasswordConfigured,
  backupIncludeAssets,
  backupLocalDir,
  backupRetentionCount,
  backupScheduleEnabled,
  backupTimeOfDay,
  downloadPassword,
  includeAssetsInBackup,
  lastRunAt,
  lastError,
  lastStatus,
  localBackupDir,
  localBackups,
  localBackupsRefreshing,
  onBackupEncryptEnabledChange,
  onBackupEncryptionPasswordChange,
  onBackupIncludeAssetsChange,
  onBackupLocalDirChange,
  onBackupRetentionCountChange,
  onBackupScheduleEnabledChange,
  onBackupTimeOfDayChange,
  onDownloadBackup,
  onDownloadPasswordChange,
  onIncludeAssetsInBackupChange,
  onRefreshLocalBackups,
  onRestore,
  onRestoreConfirmOpenChange,
  onRestoreFileChange,
  onRestorePasswordChange,
  onRunBackupNow,
  onSaveSettings,
  restoreConfirmOpen,
  restoreEncrypted,
  restoreFile,
  restorePassword,
  runningBackup,
}: AdminBackupTabProps) {
  const { t, i18n } = useTranslation()
  const [editingEncryptionPassword, setEditingEncryptionPassword] = useState(false)
  const configuredMaskValue = "••••••••"
  const encryptionPasswordDisplayValue = editingEncryptionPassword
    ? backupEncryptionPassword
    : backupEncryptionPassword ||
      (backupEncryptionPasswordConfigured ? configuredMaskValue : "")

  const lastRunLabel =
    lastStatus === "success"
      ? t("admin.backup.lastRunSuccess")
      : lastStatus === "failed"
        ? t("admin.backup.lastRunFailed")
        : t("admin.backup.lastRunNever")

  return (
    <TabsContent value="backup" className="space-y-6">
      <div className="space-y-4">
        <div>
          <h3 className="text-sm font-medium">{t("admin.backup.download")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("admin.backup.downloadDescription")}
          </p>
        </div>
        <div className="flex items-center justify-between gap-4 rounded-md border p-3">
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
        <div className="space-y-2">
          <Label htmlFor="backup-download-password">{t("admin.backup.downloadPassword")}</Label>
          <Input
            id="backup-download-password"
            type="password"
            autoComplete="new-password"
            value={downloadPassword}
            onChange={(event) => onDownloadPasswordChange(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.backup.downloadPasswordDescription")}
          </p>
        </div>
        <Button variant="outline" onClick={() => void onDownloadBackup()}>
          <Download className="size-4" />
          {t("admin.backup.downloadButton")}
        </Button>
      </div>

      <Separator />

      <div className="space-y-4">
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
        {restoreEncrypted && (
          <div className="space-y-2">
            <Label htmlFor="backup-restore-password">{t("admin.backup.restorePassword")}</Label>
            <Input
              id="backup-restore-password"
              type="password"
              autoComplete="new-password"
              value={restorePassword}
              onChange={(event) => onRestorePasswordChange(event.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.restorePasswordDescription")}
            </p>
          </div>
        )}
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
      </div>

      <Separator />

      <div className="space-y-4">
        <div>
          <h3 className="text-sm font-medium">{t("admin.backup.scheduleTitle")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("admin.backup.scheduleDescription")}
          </p>
        </div>

        <div className="flex items-center justify-between gap-4 rounded-md border p-3">
          <div className="space-y-0.5">
            <Label htmlFor="backup-schedule-enabled">{t("admin.backup.scheduleEnabled")}</Label>
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.scheduleEnabledDescription")}
            </p>
          </div>
          <Switch
            id="backup-schedule-enabled"
            checked={backupScheduleEnabled}
            onCheckedChange={onBackupScheduleEnabledChange}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="backup-time-of-day">{t("admin.backup.timeOfDay")}</Label>
          <Input
            id="backup-time-of-day"
            type="time"
            className="w-40"
            value={backupTimeOfDay}
            onChange={(event) => onBackupTimeOfDayChange(event.target.value)}
          />
          <p className="text-xs text-muted-foreground">{t("admin.backup.timeOfDayDescription")}</p>
        </div>

        <div className="flex items-center justify-between gap-4 rounded-md border p-3">
          <div className="space-y-0.5">
            <Label htmlFor="backup-schedule-include-assets">
              {t("admin.backup.scheduleIncludeAssets")}
            </Label>
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.includeAssetsDescription")}
            </p>
          </div>
          <Switch
            id="backup-schedule-include-assets"
            checked={backupIncludeAssets}
            onCheckedChange={onBackupIncludeAssetsChange}
          />
        </div>

        <div className="flex items-center justify-between gap-4 rounded-md border p-3">
          <div className="space-y-0.5">
            <Label htmlFor="backup-encrypt-enabled">{t("admin.backup.encrypt")}</Label>
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.encryptDescription")}
            </p>
          </div>
          <Switch
            id="backup-encrypt-enabled"
            checked={backupEncryptEnabled}
            onCheckedChange={onBackupEncryptEnabledChange}
          />
        </div>

        {backupEncryptEnabled && (
          <div className="space-y-2">
            <Label htmlFor="backup-encryption-password">
              {t("admin.backup.encryptionPassword")}
            </Label>
            <Input
              id="backup-encryption-password"
              type="password"
              value={encryptionPasswordDisplayValue}
              onFocus={() => setEditingEncryptionPassword(true)}
              onBlur={() => setEditingEncryptionPassword(false)}
              onChange={(event) => onBackupEncryptionPasswordChange(event.target.value)}
              placeholder={t("admin.backup.encryptionPasswordConfigured")}
            />
          </div>
        )}

        <div className="space-y-2">
          <Label htmlFor="backup-local-dir">{t("admin.backup.localDir")}</Label>
          <Input
            id="backup-local-dir"
            value={backupLocalDir}
            onChange={(event) => onBackupLocalDirChange(event.target.value)}
            placeholder={t("admin.backup.localDirPlaceholder")}
          />
          <p className="text-xs text-muted-foreground">{t("admin.backup.localDirDescription")}</p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="backup-retention-count">{t("admin.backup.retentionCount")}</Label>
          <Input
            id="backup-retention-count"
            type="number"
            min={1}
            step={1}
            className="w-32"
            value={backupRetentionCount}
            onChange={(event) => {
              const next = parseInt(event.target.value, 10)
              if (!Number.isNaN(next) && next >= 1) {
                onBackupRetentionCountChange(next)
              }
            }}
          />
          <p className="text-xs text-muted-foreground">
            {t("admin.backup.retentionCountDescription")}
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <Button onClick={() => void onSaveSettings()}>{t("admin.backup.saveSchedule")}</Button>
          <Button variant="outline" disabled={runningBackup} onClick={() => void onRunBackupNow()}>
            <PlayCircle className="size-4" />
            {t("admin.backup.runNow")}
          </Button>
        </div>

        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Clock className="size-3.5" />
          <span>
            {t("admin.backup.lastRun")}: {lastRunLabel}
            {lastRunAt ? ` (${formatDateTime(lastRunAt, i18n.language)})` : ""}
          </span>
        </div>
        {lastStatus === "failed" && lastError && (
          <p className="text-xs text-destructive">{lastError}</p>
        )}
      </div>

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h3 className="text-sm font-medium">{t("admin.backup.recentBackups")}</h3>
            {localBackupDir && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {t("admin.backup.directoryLabel")}: <span className="font-mono">{localBackupDir}</span>
              </p>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            disabled={localBackupsRefreshing}
            onClick={() => void onRefreshLocalBackups()}
          >
            <RefreshCw className={`size-4 ${localBackupsRefreshing ? "animate-spin" : ""}`} />
            {t("admin.backup.refreshBackups")}
          </Button>
        </div>

        {localBackups.length === 0 ? (
          <p className="rounded-md border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            {t("admin.backup.recentBackupsEmpty")}
          </p>
        ) : (
          <ul className="space-y-2">
            {localBackups.map((backup) => (
              <li
                key={backup.name}
                className="flex items-center justify-between gap-4 rounded-md border p-3"
              >
                <div className="min-w-0 space-y-0.5">
                  <div className="flex items-center gap-2">
                    <span className="truncate font-mono text-sm">{backup.name}</span>
                    {backup.encrypted && (
                      <Badge variant="secondary" className="gap-1">
                        <Lock className="size-3" />
                        {t("admin.backup.backupEncryptedBadge")}
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {formatDateTime(backup.modified_at, i18n.language)}
                  </p>
                </div>
                <span className="shrink-0 text-xs text-muted-foreground">
                  {formatBytes(backup.size, i18n.language)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </TabsContent>
  )
}
