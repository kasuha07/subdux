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
import type { BackupDestination, LocalBackupInfo } from "@/types"
import { formatBytes, formatDateTime } from "./admin-backup-format"
import AdminBackupDestinationsSection, {
  type DestinationCreateBody,
  type DestinationUpdateBody,
} from "./admin-backup-destinations-section"
import { DESTINATION_SECRET_MASK } from "./hooks/backup-destinations"
import ReauthDialog, { type ReauthScope } from "./reauth-dialog"

interface AdminBackupTabProps {
  backupEncryptEnabled: boolean
  backupEncryptionPassword: string
  backupEncryptionPasswordConfigured: boolean
  backupIncludeAssets: boolean
  backupScheduleEnabled: boolean
  backupTimeOfDay: string
  destinations: BackupDestination[]
  destinationsRefreshing: boolean
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
  onBackupScheduleEnabledChange: (value: boolean) => void
  onBackupTimeOfDayChange: (value: string) => void
  onCreateDestination: (body: DestinationCreateBody, reauthTicket: string) => Promise<boolean>
  onUpdateDestination: (id: number, body: DestinationUpdateBody, reauthTicket: string) => Promise<boolean>
  onDeleteDestination: (id: number, revision: number, reauthTicket: string) => Promise<boolean>
  onTestDestination: (id: number) => Promise<void>
  onDownloadBackup: (reauthTicket: string) => Promise<boolean>
  onDownloadPasswordChange: (value: string) => void
  onIncludeAssetsInBackupChange: (value: boolean) => void
  onRefreshDestinations: () => void | Promise<void>
  onRefreshLocalBackups: () => void | Promise<void>
  onRestore: (reauthTicket: string) => Promise<boolean>
  onValidateRestoreInputs: () => Promise<boolean>
  onRestoreConfirmOpenChange: (open: boolean) => void
  onRestoreFileChange: (file: File | null) => void
  onRestorePasswordChange: (value: string) => void
  onRunBackupNow: (reauthTicket: string) => void | Promise<void>
  onSaveSettings: (reauthTicket: string) => void | Promise<void>
  restoreConfirmOpen: boolean
  restoreEncrypted: boolean
  restoreFile: File | null
  restorePassword: string
  runningBackup: boolean
}

export default function AdminBackupTab({
  backupEncryptEnabled,
  backupEncryptionPassword,
  backupEncryptionPasswordConfigured,
  backupIncludeAssets,
  backupScheduleEnabled,
  backupTimeOfDay,
  destinations,
  destinationsRefreshing,
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
  onBackupScheduleEnabledChange,
  onBackupTimeOfDayChange,
  onCreateDestination,
  onUpdateDestination,
  onDeleteDestination,
  onTestDestination,
  onDownloadBackup,
  onDownloadPasswordChange,
  onIncludeAssetsInBackupChange,
  onRefreshDestinations,
  onRefreshLocalBackups,
  onRestore,
  onValidateRestoreInputs,
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
  // Which flow, if any, is awaiting step-up re-authentication.
  const [reauthPrompt, setReauthPrompt] = useState<"download" | "run" | "restore" | "schedule" | "dest-create" | "dest-update" | "dest-delete" | null>(null)

  // Pending destination mutation params waiting for a reauth ticket. The
  // destination form itself lives in AdminBackupDestinationsSection; only the
  // payloads that drive `reauthPrompt` stay here alongside the dialog.
  const [pendingDestCreate, setPendingDestCreate] = useState<DestinationCreateBody | null>(null)
  const [pendingDestUpdate, setPendingDestUpdate] = useState<{ id: number; revision: number; body: DestinationUpdateBody } | null>(null)
  const [destDeleteTarget, setDestDeleteTarget] = useState<BackupDestination | null>(null)

  const configuredMaskValue = DESTINATION_SECRET_MASK
  const encryptionPasswordDisplayValue = editingEncryptionPassword
    ? backupEncryptionPassword
    : backupEncryptionPassword ||
      (backupEncryptionPasswordConfigured ? configuredMaskValue : "")

  const lastRunLabel =
    lastStatus === "success"
      ? t("admin.backup.lastRunSuccess")
      : lastStatus === "failed"
        ? t("admin.backup.lastRunFailed")
        : lastStatus === "partial"
          ? t("admin.backup.lastRunPartial")
        : t("admin.backup.lastRunNever")

  const reauthOperation =
    reauthPrompt === "dest-create"
      ? "backup_destination_create"
      : reauthPrompt === "dest-update"
        ? "backup_destination_update"
        : reauthPrompt === "dest-delete"
          ? "backup_destination_delete"
          : reauthPrompt === "restore"
            ? "restore"
            : reauthPrompt === "run"
              ? "backup_run"
              : reauthPrompt === "schedule"
                ? "backup_schedule"
                : "backup"

  // Only update/delete may carry a resource binding: the backend's
  // ValidateTicketBinding rejects a bound create ticket, so create sends no scope.
  const reauthScope: ReauthScope | undefined =
    reauthPrompt === "dest-update" && pendingDestUpdate
      ? {
          destination_id: pendingDestUpdate.id,
          destination_revision: pendingDestUpdate.revision,
        }
      : reauthPrompt === "dest-delete" && destDeleteTarget
        ? {
            destination_id: destDeleteTarget.id,
            destination_revision: destDeleteTarget.revision,
          }
        : undefined

  return (
    <TabsContent value="backup" className="space-y-6 select-none">
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
        <Button variant="outline" onClick={() => setReauthPrompt("download")}>
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
              <Button
                size="sm"
                variant="destructive"
                onClick={async () => {
                  if (await onValidateRestoreInputs()) {
                    setReauthPrompt("restore")
                  }
                }}
              >
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

        <div className="flex flex-wrap gap-2">
          <Button onClick={() => setReauthPrompt("schedule")}>{t("admin.backup.saveSchedule")}</Button>
          <Button variant="outline" disabled={runningBackup} onClick={() => setReauthPrompt("run")}>
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
        {(lastStatus === "failed" || lastStatus === "partial") && lastError && (
          <p className="text-xs text-destructive">{lastError}</p>
        )}
      </div>

      <Separator />

      <AdminBackupDestinationsSection
        destinations={destinations}
        destinationsRefreshing={destinationsRefreshing}
        onRefreshDestinations={onRefreshDestinations}
        onTestDestination={onTestDestination}
        onRequestCreate={(body) => {
          setPendingDestCreate(body)
          setReauthPrompt("dest-create")
        }}
        onRequestUpdate={(id, revision, body) => {
          setPendingDestUpdate({ id, revision, body })
          setReauthPrompt("dest-update")
        }}
        onRequestDelete={(destination) => {
          setDestDeleteTarget(destination)
          setReauthPrompt("dest-delete")
        }}
      />

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

      <ReauthDialog
        operation={reauthOperation}
        scope={reauthScope}
        open={reauthPrompt !== null}
        onOpenChange={(open) => {
          if (!open) {
            setReauthPrompt(null)
          }
        }}
        onVerified={async (ticket) => {
          let closePrompt = true
          if (reauthPrompt === "download") {
            await onDownloadBackup(ticket)
          } else if (reauthPrompt === "run") {
            await onRunBackupNow(ticket)
          } else if (reauthPrompt === "restore") {
            await onRestore(ticket)
          } else if (reauthPrompt === "schedule") {
            await onSaveSettings(ticket)
          } else if (reauthPrompt === "dest-create" && pendingDestCreate) {
            closePrompt = await onCreateDestination(pendingDestCreate, ticket)
            if (closePrompt) {
              setPendingDestCreate(null)
            }
          } else if (reauthPrompt === "dest-update" && pendingDestUpdate) {
            closePrompt = await onUpdateDestination(pendingDestUpdate.id, pendingDestUpdate.body, ticket)
            if (closePrompt) {
              setPendingDestUpdate(null)
            }
          } else if (reauthPrompt === "dest-delete" && destDeleteTarget) {
            closePrompt = await onDeleteDestination(destDeleteTarget.id, destDeleteTarget.revision, ticket)
            if (closePrompt) {
              setDestDeleteTarget(null)
            }
          }
          if (closePrompt) {
            setReauthPrompt(null)
          }
        }}
        title={t("admin.backup.reauth.title")}
        description={
          reauthPrompt === "restore"
            ? t("admin.backup.reauth.restoreDescription")
            : reauthPrompt === "run"
              ? t("admin.backup.reauth.runNowDescription")
              : reauthPrompt === "schedule"
                ? t("admin.backup.reauth.scheduleDescription")
                : reauthPrompt === "dest-create"
                  ? t("admin.backup.reauth.destinationCreateDescription")
                  : reauthPrompt === "dest-update"
                    ? t("admin.backup.reauth.destinationUpdateDescription")
                    : reauthPrompt === "dest-delete"
                      ? t("admin.backup.reauth.destinationDeleteDescription")
                      : t("admin.backup.reauth.downloadDescription")
        }
        confirmVariant={reauthPrompt === "restore" || reauthPrompt === "dest-delete" ? "destructive" : "default"}
      />
    </TabsContent>
  )
}
