import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Clock, Pencil, PlayCircle, Plus, RefreshCw, Trash2, Wifi } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { BackupDestination } from "@/types"

import AdminBackupDestinationFormFields, {
  type DestinationFormValues,
} from "./admin-backup-destination-form-fields"
import { formatDateTime } from "./admin-backup-format"
import {
  buildLocalConfig,
  buildS3Config,
  buildWebDAVConfig,
  destStatusVariant,
  parseLocalConfig,
  parseS3Config,
  parseWebDAVConfig,
  resolveEncryptionSecretUpdate,
  resolveSecretUpdate,
} from "./hooks/backup-destinations"

// Request bodies handed back to the parent, which holds them until a reauth
// ticket is obtained and then passes them to the mutation hook.
export interface DestinationCreateBody {
  type: string
  enabled: boolean
  config: string
  sort_order: number
}

export interface DestinationUpdateBody {
  revision: number
  enabled?: boolean
  config?: string
  sort_order?: number
  cleared_secret_fields?: string[]
}

interface AdminBackupDestinationsSectionProps {
  destinations: BackupDestination[]
  destinationsRefreshing: boolean
  // Id of the destination whose run is currently in flight, or null.
  runningDestinationId: number | null
  onRefreshDestinations: () => void | Promise<void>
  onTestDestination: (id: number) => Promise<void>
  // Create/update/delete and run are reauth-gated, so the section only reports
  // the intent plus its payload; the parent owns the step-up prompt and ticket.
  onRequestCreate: (body: DestinationCreateBody) => void
  onRequestUpdate: (id: number, revision: number, body: DestinationUpdateBody) => void
  onRequestDelete: (destination: BackupDestination) => void
  onRequestRun: (destination: BackupDestination) => void
}

const EMPTY_DESTINATION_FORM: DestinationFormValues = {
  type: "local",
  enabled: true,
  dir: "",
  retention: 7,
  endpoint: "",
  region: "",
  bucket: "",
  prefix: "",
  accessKeyId: "",
  secretAccessKey: "",
  useSsl: true,
  usePathStyle: false,
  webdavUrl: "",
  webdavPath: "",
  webdavUsername: "",
  webdavPassword: "",
  timeOfDay: "03:00",
  includeAssets: false,
  encryptEnabled: false,
  encryptionPassword: "",
}

export default function AdminBackupDestinationsSection({
  destinations,
  destinationsRefreshing,
  runningDestinationId,
  onRefreshDestinations,
  onTestDestination,
  onRequestCreate,
  onRequestUpdate,
  onRequestDelete,
  onRequestRun,
}: AdminBackupDestinationsSectionProps) {
  const { t, i18n } = useTranslation()

  // Destination form state
  const [destFormOpen, setDestFormOpen] = useState(false)
  const [destEditTarget, setDestEditTarget] = useState<BackupDestination | null>(null)
  const [destForm, setDestForm] = useState<DestinationFormValues>(EMPTY_DESTINATION_FORM)
  const [clearedWebdavSecret, setClearedWebdavSecret] = useState(false)
  // in-flight test destination id
  const [testingDestinationId, setTestingDestinationId] = useState<number | null>(null)

  function updateDestForm(patch: Partial<DestinationFormValues>) {
    setDestForm((prev) => ({ ...prev, ...patch }))
  }

  // closeDestForm also drops the form values so a typed (plaintext) secret does
  // not linger in component state after the dialog is dismissed.
  function closeDestForm() {
    setDestFormOpen(false)
    setDestForm({ ...EMPTY_DESTINATION_FORM })
    setClearedWebdavSecret(false)
  }

  function openCreateForm() {
    setDestEditTarget(null)
    setDestForm({ ...EMPTY_DESTINATION_FORM })
    setClearedWebdavSecret(false)
    setDestFormOpen(true)
  }

  // scheduleFormValues maps the shared schedule block of a stored config onto
  // the form. encryption_password is blanked by the server and only ever holds
  // what the admin types; SecretInput renders the configured mask on its own.
  function scheduleFormValues(parsed: {
    time_of_day?: string
    include_assets?: boolean
    encrypt_enabled?: boolean
  }): Pick<DestinationFormValues, "timeOfDay" | "includeAssets" | "encryptEnabled" | "encryptionPassword"> {
    return {
      timeOfDay: parsed.time_of_day || EMPTY_DESTINATION_FORM.timeOfDay,
      includeAssets: parsed.include_assets ?? false,
      encryptEnabled: parsed.encrypt_enabled ?? false,
      encryptionPassword: "",
    }
  }

  // openEditForm always rebuilds the whole form from the destination, so fields
  // belonging to the other destination types return to their defaults.
  function openEditForm(dest: BackupDestination) {
    setDestEditTarget(dest)
    setClearedWebdavSecret(false)
    if (dest.type === "s3") {
      const parsed = parseS3Config(dest)
      setDestForm({
        ...EMPTY_DESTINATION_FORM,
        type: "s3",
        enabled: dest.enabled,
        endpoint: parsed.endpoint ?? "",
        region: parsed.region ?? "",
        bucket: parsed.bucket ?? "",
        prefix: parsed.prefix ?? "",
        accessKeyId: parsed.access_key_id ?? "",
        // secret_access_key is blanked by the server; the form starts empty and
        // SecretInput shows the configured mask without holding the value
        secretAccessKey: "",
        useSsl: parsed.use_ssl ?? true,
        usePathStyle: parsed.use_path_style ?? false,
        retention: parsed.retention_count ?? 7,
        ...scheduleFormValues(parsed),
      })
    } else if (dest.type === "webdav") {
      const parsed = parseWebDAVConfig(dest)
      setDestForm({
        ...EMPTY_DESTINATION_FORM,
        type: "webdav",
        enabled: dest.enabled,
        webdavUrl: parsed.url ?? "",
        webdavPath: parsed.path ?? "",
        webdavUsername: parsed.username ?? "",
        // password is blanked by the server; same empty-start treatment
        webdavPassword: "",
        retention: parsed.retention_count ?? 7,
        ...scheduleFormValues(parsed),
      })
    } else {
      const parsed = parseLocalConfig(dest)
      setDestForm({
        ...EMPTY_DESTINATION_FORM,
        type: "local",
        enabled: dest.enabled,
        dir: parsed.dir ?? "",
        retention: parsed.retention_count ?? 7,
        ...scheduleFormValues(parsed),
      })
    }
    setDestFormOpen(true)
  }

  function handleDestFormSave() {
    let config: string
    const clearedSecretFieldNames: string[] = []
    const secretIsConfigured = (field: string) =>
      destEditTarget?.configured_secret_fields.includes(field) ?? false

    // The encryption password is shared by every destination type, so resolve it
    // once here and feed the result into whichever build*Config runs below.
    // Saving with encryption off drops any stored password; on create the typed
    // value passes through.
    const encryptionResolution = resolveEncryptionSecretUpdate(
      destForm.encryptEnabled,
      destForm.encryptionPassword,
      secretIsConfigured("encryption_password")
    )
    if (encryptionResolution.cleared_secret_fields) {
      clearedSecretFieldNames.push(...encryptionResolution.cleared_secret_fields)
    }
    const schedule = {
      timeOfDay: destForm.timeOfDay,
      includeAssets: destForm.includeAssets,
      encryptEnabled: destForm.encryptEnabled,
      encryptionPassword: encryptionResolution.value,
    }

    if (destForm.type === "s3") {
      // The s3 secret is replace-only (no clear affordance, cleared is always
      // false): empty preserves the stored secret on update.
      const secretResolution = resolveSecretUpdate(
        "secret_access_key",
        destForm.secretAccessKey,
        false,
        secretIsConfigured("secret_access_key")
      )
      config = buildS3Config({
        endpoint: destForm.endpoint,
        region: destForm.region,
        bucket: destForm.bucket,
        prefix: destForm.prefix,
        accessKeyId: destForm.accessKeyId,
        secretAccessKey: secretResolution.value,
        useSsl: destForm.useSsl,
        usePathStyle: destForm.usePathStyle,
        retentionCount: destForm.retention,
        ...schedule,
      })
    } else if (destForm.type === "webdav") {
      const passwordResolution = resolveSecretUpdate(
        "password",
        destForm.webdavPassword,
        clearedWebdavSecret,
        secretIsConfigured("password")
      )
      if (passwordResolution.cleared_secret_fields) {
        clearedSecretFieldNames.push(...passwordResolution.cleared_secret_fields)
      }
      config = buildWebDAVConfig({
        url: destForm.webdavUrl,
        path: destForm.webdavPath,
        username: destForm.webdavUsername,
        password: passwordResolution.value,
        retentionCount: destForm.retention,
        ...schedule,
      })
    } else {
      config = buildLocalConfig({
        dir: destForm.dir,
        retentionCount: destForm.retention,
        ...schedule,
      })
    }

    const clearedSecretFields =
      clearedSecretFieldNames.length > 0 ? clearedSecretFieldNames : undefined

    if (destEditTarget) {
      const body: DestinationUpdateBody = {
        revision: destEditTarget.revision,
        enabled: destForm.enabled,
        config,
        sort_order: destEditTarget.sort_order,
      }
      if (clearedSecretFields) body.cleared_secret_fields = clearedSecretFields
      closeDestForm()
      onRequestUpdate(destEditTarget.id, destEditTarget.revision, body)
    } else {
      closeDestForm()
      onRequestCreate({
        type: destForm.type,
        enabled: destForm.enabled,
        config,
        sort_order: destinations.length,
      })
    }
  }

  function destStatusLabel(status: string) {
    if (status === "success") return t("admin.backup.lastRunSuccess")
    if (status === "failed") return t("admin.backup.lastRunFailed")
    if (status === "partial") return t("admin.backup.lastRunPartial")
    return t("admin.backup.lastRunNever")
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h3 className="text-sm font-medium">{t("admin.backup.destinations.title")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("admin.backup.destinations.description")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={destinationsRefreshing}
            onClick={() => void onRefreshDestinations()}
          >
            <RefreshCw className={`size-4 ${destinationsRefreshing ? "animate-spin" : ""}`} />
            {t("admin.backup.refreshBackups")}
          </Button>
          <Button size="sm" onClick={openCreateForm}>
            <Plus className="size-4" />
            {t("admin.backup.destinations.add")}
          </Button>
        </div>
      </div>

      <Dialog
        open={destFormOpen}
        onOpenChange={(open) => {
          if (!open) closeDestForm()
        }}
      >
        <DialogContent
          onOpenAutoFocus={(event) => event.preventDefault()}
          className="flex max-h-[calc(100vh-1.5rem)] flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]"
        >
          <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
            <DialogTitle>
              {destEditTarget
                ? t("admin.backup.destinations.editTitle")
                : t("admin.backup.destinations.addTitle")}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t("admin.backup.destinations.description")}
            </DialogDescription>
          </DialogHeader>

          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4 sm:px-6">
            <AdminBackupDestinationFormFields
              key={destEditTarget?.id ?? "create"}
              values={destForm}
              onValuesChange={updateDestForm}
              editTarget={destEditTarget}
              clearedWebdavSecret={clearedWebdavSecret}
              onClearWebdavSecret={() => {
                setClearedWebdavSecret(true)
                updateDestForm({ webdavPassword: "" })
              }}
            />
          </div>

          <div className="border-t px-5 py-4 sm:px-6">
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" onClick={closeDestForm}>
                {t("admin.backup.cancel")}
              </Button>
              <Button className="flex-1" onClick={handleDestFormSave}>
                {t("admin.backup.destinations.save")}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {destinations.length === 0 ? (
        <p className="rounded-md border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
          {t("admin.backup.destinations.empty")}
        </p>
      ) : (
        <ul className="space-y-2">
          {destinations.map((dest) => {
            const isS3 = dest.type === "s3"
            const isWebDAV = dest.type === "webdav"
            const localParsed = parseLocalConfig(dest)
            const s3Parsed = parseS3Config(dest)
            const webdavParsed = parseWebDAVConfig(dest)
            const dir = localParsed.dir
            const retention = localParsed.retention_count
            // The schedule block lives in every type's config, so read it back
            // from whichever parser matches this destination's type.
            const schedule = isS3 ? s3Parsed : isWebDAV ? webdavParsed : localParsed
            return (
              <li
                key={dest.id}
                className="flex items-start justify-between gap-4 rounded-md border p-3"
              >
                <div className="min-w-0 space-y-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium">
                      {isS3
                        ? t("admin.backup.destinations.typeS3")
                        : isWebDAV
                          ? t("admin.backup.destinations.typeWebDAV")
                          : t("admin.backup.destinations.typeLocal")}
                    </span>
                    {dest.enabled ? (
                      <Badge variant="secondary">{t("admin.backup.destinations.statusEnabled")}</Badge>
                    ) : (
                      <Badge variant="outline">{t("admin.backup.destinations.statusDisabled")}</Badge>
                    )}
                    {dest.last_status && (
                      <Badge variant={destStatusVariant(dest.last_status)}>
                        {destStatusLabel(dest.last_status)}
                      </Badge>
                    )}
                    {dest.last_retention_status && dest.last_retention_status !== "success" && (
                      <Badge variant={destStatusVariant(dest.last_retention_status)}>
                        {t("admin.backup.retentionCount")}: {destStatusLabel(dest.last_retention_status)}
                      </Badge>
                    )}
                    {dest.last_bookkeeping_status && dest.last_bookkeeping_status !== "success" && (
                      <Badge variant={destStatusVariant(dest.last_bookkeeping_status)}>
                        {t("admin.backup.lastRunFailed")}
                      </Badge>
                    )}
                    {dest.last_status === "failed" && dest.last_error && (
                      <span className="text-xs text-destructive">{dest.last_error}</span>
                    )}
                    {dest.last_retention_status === "failed" && dest.last_retention_error && (
                      <span className="text-xs text-destructive">{dest.last_retention_error}</span>
                    )}
                    {dest.last_bookkeeping_status === "failed" && dest.last_bookkeeping_error && (
                      <span className="text-xs text-destructive">{dest.last_bookkeeping_error}</span>
                    )}
                  </div>
                  <div className="text-xs text-muted-foreground space-y-0.5">
                    {isS3 ? (
                      <>
                        {s3Parsed.endpoint && (
                          <p>
                            {t("admin.backup.destinations.s3Endpoint")}:{" "}
                            <span className="font-mono">{s3Parsed.endpoint}</span>
                          </p>
                        )}
                        {s3Parsed.bucket && (
                          <p>
                            {t("admin.backup.destinations.s3Bucket")}:{" "}
                            <span className="font-mono">{s3Parsed.bucket}</span>
                          </p>
                        )}
                        {s3Parsed.prefix && (
                          <p>
                            {t("admin.backup.destinations.s3Prefix")}:{" "}
                            <span className="font-mono">{s3Parsed.prefix}</span>
                          </p>
                        )}
                      </>
                    ) : isWebDAV ? (
                      <>
                        {webdavParsed.url && (
                          <p>
                            {t("admin.backup.destinations.webdavUrl")}:{" "}
                            <span className="font-mono">{webdavParsed.url}</span>
                          </p>
                        )}
                        {webdavParsed.path && (
                          <p>
                            {t("admin.backup.destinations.webdavPath")}:{" "}
                            <span className="font-mono">{webdavParsed.path}</span>
                          </p>
                        )}
                      </>
                    ) : (
                      <>
                        {dir ? (
                          <p>
                            {t("admin.backup.directoryLabel")}:{" "}
                            <span className="font-mono">{dir}</span>
                          </p>
                        ) : (
                          <p>{t("admin.backup.destinations.defaultDir")}</p>
                        )}
                        {retention !== undefined && (
                          <p>
                            {t("admin.backup.retentionCount")}: {retention}
                          </p>
                        )}
                      </>
                    )}
                    {(schedule.time_of_day || schedule.include_assets || schedule.encrypt_enabled) && (
                      <div className="flex flex-wrap items-center gap-1.5">
                        {schedule.time_of_day && (
                          <span>
                            {t("admin.backup.destinations.scheduleTimeOfDay")}:{" "}
                            <span className="font-mono">{schedule.time_of_day}</span>
                          </span>
                        )}
                        {schedule.include_assets && (
                          <Badge variant="outline">
                            {t("admin.backup.destinations.scheduleIncludeAssets")}
                          </Badge>
                        )}
                        {schedule.encrypt_enabled && (
                          <Badge variant="outline">
                            {t("admin.backup.destinations.scheduleEncrypt")}
                          </Badge>
                        )}
                      </div>
                    )}
                    {dest.last_run_at && (
                      <p className="flex items-center gap-1">
                        <Clock className="size-3" />
                        {formatDateTime(dest.last_run_at, i18n.language)}
                      </p>
                    )}
                    {dest.last_scheduled_run_at && (
                      <p className="flex items-center gap-1">
                        <Clock className="size-3" />
                        {t("admin.backup.destinations.lastScheduledRun")}:{" "}
                        {formatDateTime(dest.last_scheduled_run_at, i18n.language)}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex shrink-0 gap-1">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={runningDestinationId === dest.id}
                    title={t("admin.backup.destinations.runNow")}
                    onClick={() => onRequestRun(dest)}
                  >
                    <PlayCircle
                      className={`size-3.5 ${runningDestinationId === dest.id ? "animate-pulse" : ""}`}
                    />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    disabled={testingDestinationId === dest.id}
                    title={t("admin.backup.destinations.testConnection")}
                    onClick={() => {
                      setTestingDestinationId(dest.id)
                      void onTestDestination(dest.id).finally(() => setTestingDestinationId(null))
                    }}
                  >
                    <Wifi className={`size-3.5 ${testingDestinationId === dest.id ? "animate-pulse" : ""}`} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => openEditForm(dest)}
                  >
                    <Pencil className="size-3.5" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => onRequestDelete(dest)}
                  >
                    <Trash2 className="size-3.5 text-destructive" />
                  </Button>
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
