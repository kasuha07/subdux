import { useState } from "react"
import { useTranslation } from "react-i18next"
import {
  AlertTriangle,
  Download,
  Lock,
  RefreshCw,
} from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { TabsContent } from "@/components/ui/tabs"
import type { BackupDestination, BackupRunRecord, DestinationBackup } from "@/types"
import { formatBytes, formatDateTime } from "./admin-backup-format"
import AdminBackupDestinationsSection, {
  type DestinationCreateBody,
  type DestinationUpdateBody,
} from "./admin-backup-destinations-section"
import { destStatusVariant, type DestinationProbeRequest } from "./hooks/backup-destinations"
import ReauthDialog, { type ReauthScope } from "./reauth-dialog"

interface AdminBackupTabProps {
  backupRuns: BackupRunRecord[]
  backupRunsRefreshing: boolean
  destinations: BackupDestination[]
  destinationsRefreshing: boolean
  downloadPassword: string
  includeAssetsInBackup: boolean
  runningDestinationId: number | null
  onCreateDestination: (body: DestinationCreateBody, reauthTicket: string) => Promise<boolean>
  onUpdateDestination: (id: number, body: DestinationUpdateBody, reauthTicket: string) => Promise<boolean>
  onDeleteDestination: (id: number, revision: number, reauthTicket: string) => Promise<boolean>
  onRunDestination: (id: number, reauthTicket: string) => void | Promise<void>
  onTestDestination: (id: number) => Promise<void>
  onLoadDestinationBackups: (id: number) => Promise<DestinationBackup[]>
  onRestoreFromDestination: (
    id: number,
    archiveName: string,
    password: string,
    reauthTicket: string
  ) => Promise<boolean>
  onTestDestinationConfig: (body: DestinationProbeRequest) => Promise<void>
  onDownloadBackup: (reauthTicket: string) => Promise<boolean>
  onDownloadPasswordChange: (value: string) => void
  onIncludeAssetsInBackupChange: (value: boolean) => void
  onRefreshBackupRuns: () => void | Promise<void>
  onRefreshDestinations: () => void | Promise<void>
  onRestore: (reauthTicket: string) => Promise<boolean>
  onValidateRestoreInputs: () => Promise<boolean>
  onRestoreConfirmOpenChange: (open: boolean) => void
  onRestoreFileChange: (file: File | null) => void
  onRestorePasswordChange: (value: string) => void
  restoreConfirmOpen: boolean
  restoreEncrypted: boolean
  restoreFile: File | null
  restorePassword: string
}

export default function AdminBackupTab({
  backupRuns,
  backupRunsRefreshing,
  destinations,
  destinationsRefreshing,
  downloadPassword,
  includeAssetsInBackup,
  runningDestinationId,
  onCreateDestination,
  onUpdateDestination,
  onDeleteDestination,
  onRunDestination,
  onTestDestination,
  onLoadDestinationBackups,
  onRestoreFromDestination,
  onTestDestinationConfig,
  onDownloadBackup,
  onDownloadPasswordChange,
  onIncludeAssetsInBackupChange,
  onRefreshBackupRuns,
  onRefreshDestinations,
  onRestore,
  onValidateRestoreInputs,
  onRestoreConfirmOpenChange,
  onRestoreFileChange,
  onRestorePasswordChange,
  restoreConfirmOpen,
  restoreEncrypted,
  restoreFile,
  restorePassword,
}: AdminBackupTabProps) {
  const { t, i18n } = useTranslation()
  // Which flow, if any, is awaiting step-up re-authentication.
  const [reauthPrompt, setReauthPrompt] = useState<"download" | "restore" | "dest-create" | "dest-update" | "dest-delete" | "dest-run" | "dest-restore" | null>(null)

  function destinationTypeLabel(type: string) {
    if (type === "s3") return t("admin.backup.destinations.typeS3")
    if (type === "webdav") return t("admin.backup.destinations.typeWebDAV")
    if (type === "local") return t("admin.backup.destinations.typeLocal")
    return type
  }

  function runSourceLabel(source: string) {
    if (source === "scheduled") return t("admin.backup.runSourceScheduled")
    if (source === "download") return t("admin.backup.runSourceDownload")
    return t("admin.backup.runSourceManual")
  }

  function runStatusLabel(status: string) {
    if (status === "success") return t("admin.backup.lastRunSuccess")
    if (status === "failed") return t("admin.backup.lastRunFailed")
    if (status === "partial") return t("admin.backup.lastRunPartial")
    if (status === "superseded") return t("admin.backup.runStatusSuperseded")
    return t("admin.backup.runStatusPending")
  }

  // Pending destination mutation params waiting for a reauth ticket. The
  // destination form itself lives in AdminBackupDestinationsSection; only the
  // payloads that drive `reauthPrompt` stay here alongside the dialog.
  const [pendingDestCreate, setPendingDestCreate] = useState<DestinationCreateBody | null>(null)
  const [pendingDestUpdate, setPendingDestUpdate] = useState<{ id: number; revision: number; body: DestinationUpdateBody } | null>(null)
  const [destDeleteTarget, setDestDeleteTarget] = useState<BackupDestination | null>(null)
  const [destRunTarget, setDestRunTarget] = useState<BackupDestination | null>(null)
  const [pendingDestRestore, setPendingDestRestore] = useState<{ id: number; archiveName: string; password: string } | null>(null)

  const reauthOperation =
    reauthPrompt === "dest-create"
      ? "backup_destination_create"
      : reauthPrompt === "dest-update"
        ? "backup_destination_update"
        : reauthPrompt === "dest-delete"
          ? "backup_destination_delete"
          : reauthPrompt === "restore" || reauthPrompt === "dest-restore"
            ? "restore"
            : reauthPrompt === "dest-run"
              ? "backup_run"
              : "backup"

  // Only update/delete may carry a resource binding: the backend's
  // ValidateTicketBinding rejects a bound create ticket, so create sends no
  // scope. The per-destination run and the per-destination restore are also
  // unbound for the same reason: neither changes destination configuration,
  // so ValidateTicketBinding rejects a bound ticket for them too.
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

      <AdminBackupDestinationsSection
        destinations={destinations}
        destinationsRefreshing={destinationsRefreshing}
        runningDestinationId={runningDestinationId}
        onRefreshDestinations={onRefreshDestinations}
        onTestDestination={onTestDestination}
        onTestDestinationConfig={onTestDestinationConfig}
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
        onRequestRun={(destination) => {
          setDestRunTarget(destination)
          setReauthPrompt("dest-run")
        }}
        onLoadDestinationBackups={onLoadDestinationBackups}
        onRequestRestore={(destination, archiveName, password) => {
          setPendingDestRestore({ id: destination.id, archiveName, password })
          setReauthPrompt("dest-restore")
        }}
      />

      <Separator />

      <div className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h3 className="text-sm font-medium">{t("admin.backup.recentBackups")}</h3>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {t("admin.backup.recentBackupsDescription")}
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            disabled={backupRunsRefreshing}
            onClick={() => void onRefreshBackupRuns()}
          >
            <RefreshCw className={`size-4 ${backupRunsRefreshing ? "animate-spin" : ""}`} />
            {t("admin.backup.refreshBackups")}
          </Button>
        </div>

        {backupRuns.length === 0 ? (
          <p className="rounded-md border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            {t("admin.backup.recentBackupsEmpty")}
          </p>
        ) : (
          <ul className="space-y-2">
            {backupRuns.map((run) => (
              <li key={run.id} className="space-y-1.5 rounded-md border p-3">
                <div className="flex items-center justify-between gap-4">
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="truncate font-mono text-sm">{run.archive_name}</span>
                    {run.encrypted && (
                      <Badge variant="secondary" className="gap-1">
                        <Lock className="size-3" />
                        {t("admin.backup.backupEncryptedBadge")}
                      </Badge>
                    )}
                  </div>
                  <div className="flex shrink-0 items-center gap-2">
                    {run.size_bytes > 0 && (
                      <span className="text-xs text-muted-foreground">
                        {formatBytes(run.size_bytes, i18n.language)}
                      </span>
                    )}
                    <Badge variant={destStatusVariant(run.status)}>
                      {runStatusLabel(run.status)}
                    </Badge>
                  </div>
                </div>
                <div className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted-foreground">
                  <span>{formatDateTime(run.finished_at ?? run.started_at, i18n.language)}</span>
                  <span>·</span>
                  <span>{runSourceLabel(run.source)}</span>
                  {run.destinations.map((destination) => (
                    <Badge
                      key={destination.destination_id}
                      variant="outline"
                      className={destination.delivery_status === "failed" ? "text-destructive" : ""}
                    >
                      {destinationTypeLabel(destination.type)} #{destination.destination_id}
                    </Badge>
                  ))}
                </div>
                {run.error && (
                  <p className="text-xs break-words text-destructive">{run.error}</p>
                )}
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
            setPendingDestRestore(null)
          }
        }}
        onVerified={async (ticket) => {
          let closePrompt = true
          if (reauthPrompt === "download") {
            await onDownloadBackup(ticket)
          } else if (reauthPrompt === "restore") {
            await onRestore(ticket)
          } else if (reauthPrompt === "dest-run" && destRunTarget) {
            await onRunDestination(destRunTarget.id, ticket)
            setDestRunTarget(null)
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
          } else if (reauthPrompt === "dest-restore" && pendingDestRestore) {
            closePrompt = await onRestoreFromDestination(
              pendingDestRestore.id,
              pendingDestRestore.archiveName,
              pendingDestRestore.password,
              ticket
            )
            if (closePrompt) {
              setPendingDestRestore(null)
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
            : reauthPrompt === "dest-run"
              ? t("admin.backup.reauth.runNowDescription")
              : reauthPrompt === "dest-create"
                ? t("admin.backup.reauth.destinationCreateDescription")
                : reauthPrompt === "dest-update"
                  ? t("admin.backup.reauth.destinationUpdateDescription")
                  : reauthPrompt === "dest-delete"
                    ? t("admin.backup.reauth.destinationDeleteDescription")
                    : reauthPrompt === "dest-restore"
                      ? t("admin.backup.reauth.destinationRestoreDescription")
                      : t("admin.backup.reauth.downloadDescription")
        }
        confirmVariant={
          reauthPrompt === "restore" || reauthPrompt === "dest-delete" || reauthPrompt === "dest-restore"
            ? "destructive"
            : "default"
        }
      />
    </TabsContent>
  )
}
