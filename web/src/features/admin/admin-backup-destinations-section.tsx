import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Clock, Pencil, Plus, RefreshCw, Trash2, Wifi } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import type { BackupDestination } from "@/types"

import AdminBackupDestinationFormFields, {
  type DestinationFormValues,
} from "./admin-backup-destination-form-fields"
import { formatDateTime } from "./admin-backup-format"
import {
  buildLocalConfig,
  buildS3Config,
  buildWebDAVConfig,
  DESTINATION_SECRET_MASK,
  destStatusVariant,
  parseLocalConfig,
  parseS3Config,
  parseWebDAVConfig,
  resolveS3SecretUpdate,
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
  onRefreshDestinations: () => void | Promise<void>
  onTestDestination: (id: number) => Promise<void>
  // Create/update/delete are reauth-gated, so the section only reports the
  // intent plus its payload; the parent owns the step-up prompt and the ticket.
  onRequestCreate: (body: DestinationCreateBody) => void
  onRequestUpdate: (id: number, revision: number, body: DestinationUpdateBody) => void
  onRequestDelete: (destination: BackupDestination) => void
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
}

export default function AdminBackupDestinationsSection({
  destinations,
  destinationsRefreshing,
  onRefreshDestinations,
  onTestDestination,
  onRequestCreate,
  onRequestUpdate,
  onRequestDelete,
}: AdminBackupDestinationsSectionProps) {
  const { t, i18n } = useTranslation()

  // Destination form state
  const [destFormOpen, setDestFormOpen] = useState(false)
  const [destEditTarget, setDestEditTarget] = useState<BackupDestination | null>(null)
  const [destForm, setDestForm] = useState<DestinationFormValues>(EMPTY_DESTINATION_FORM)
  const [editingS3Secret, setEditingS3Secret] = useState(false)
  const [editingWebdavSecret, setEditingWebdavSecret] = useState(false)
  // in-flight test destination id
  const [testingDestinationId, setTestingDestinationId] = useState<number | null>(null)

  function updateDestForm(patch: Partial<DestinationFormValues>) {
    setDestForm((prev) => ({ ...prev, ...patch }))
  }

  function openCreateForm() {
    setDestEditTarget(null)
    setDestForm({ ...EMPTY_DESTINATION_FORM })
    setEditingS3Secret(false)
    setEditingWebdavSecret(false)
    setDestFormOpen(true)
  }

  // openEditForm always rebuilds the whole form from the destination, so fields
  // belonging to the other destination types return to their defaults.
  function openEditForm(dest: BackupDestination) {
    setDestEditTarget(dest)
    setEditingS3Secret(false)
    setEditingWebdavSecret(false)
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
        // secret_access_key is blanked by the server; use mask if configured
        secretAccessKey: dest.configured_secret_fields.includes("secret_access_key")
          ? DESTINATION_SECRET_MASK
          : "",
        useSsl: parsed.use_ssl ?? true,
        usePathStyle: parsed.use_path_style ?? false,
        retention: parsed.retention_count ?? 7,
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
        // password is blanked by the server; use mask if configured
        webdavPassword: dest.configured_secret_fields.includes("password")
          ? DESTINATION_SECRET_MASK
          : "",
        retention: parsed.retention_count ?? 7,
      })
    } else {
      const parsed = parseLocalConfig(dest)
      setDestForm({
        ...EMPTY_DESTINATION_FORM,
        type: "local",
        enabled: dest.enabled,
        dir: parsed.dir ?? "",
        retention: parsed.retention_count ?? 7,
      })
    }
    setDestFormOpen(true)
  }

  function handleDestFormSave() {
    let config: string
    let clearedSecretFields: string[] | undefined

    if (destForm.type === "s3") {
      let secretAccessKey = destForm.secretAccessKey
      if (destEditTarget) {
        const secretIsConfigured = destEditTarget.configured_secret_fields.includes("secret_access_key")
        const resolution = resolveS3SecretUpdate(destForm.secretAccessKey, editingS3Secret, secretIsConfigured)
        secretAccessKey = resolution.secret_access_key
        clearedSecretFields = resolution.cleared_secret_fields
      }
      config = buildS3Config({
        endpoint: destForm.endpoint,
        region: destForm.region,
        bucket: destForm.bucket,
        prefix: destForm.prefix,
        accessKeyId: destForm.accessKeyId,
        secretAccessKey,
        useSsl: destForm.useSsl,
        usePathStyle: destForm.usePathStyle,
        retentionCount: destForm.retention,
      })
    } else if (destForm.type === "webdav") {
      let password = destForm.webdavPassword
      if (destEditTarget) {
        const passwordIsConfigured = destEditTarget.configured_secret_fields.includes("password")
        const resolution = resolveSecretUpdate("password", destForm.webdavPassword, editingWebdavSecret, passwordIsConfigured)
        password = resolution.value
        clearedSecretFields = resolution.cleared_secret_fields
      }
      config = buildWebDAVConfig({
        url: destForm.webdavUrl,
        path: destForm.webdavPath,
        username: destForm.webdavUsername,
        password,
        retentionCount: destForm.retention,
      })
    } else {
      config = buildLocalConfig(destForm.dir, destForm.retention)
    }

    if (destEditTarget) {
      const body: DestinationUpdateBody = {
        revision: destEditTarget.revision,
        enabled: destForm.enabled,
        config,
        sort_order: destEditTarget.sort_order,
      }
      if (clearedSecretFields) body.cleared_secret_fields = clearedSecretFields
      setDestFormOpen(false)
      onRequestUpdate(destEditTarget.id, destEditTarget.revision, body)
    } else {
      setDestFormOpen(false)
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

      {destFormOpen && (
        <div className="rounded-md border p-4 space-y-4">
          <h4 className="text-sm font-medium">
            {destEditTarget
              ? t("admin.backup.destinations.editTitle")
              : t("admin.backup.destinations.addTitle")}
          </h4>

          <AdminBackupDestinationFormFields
            values={destForm}
            onValuesChange={updateDestForm}
            editTarget={destEditTarget}
            editingS3Secret={editingS3Secret}
            onEditingS3SecretChange={setEditingS3Secret}
            editingWebdavSecret={editingWebdavSecret}
            onEditingWebdavSecretChange={setEditingWebdavSecret}
          />

          <div className="flex gap-2">
            <Button size="sm" onClick={handleDestFormSave}>
              {t("admin.backup.destinations.save")}
            </Button>
            <Button size="sm" variant="outline" onClick={() => setDestFormOpen(false)}>
              {t("admin.backup.cancel")}
            </Button>
          </div>
        </div>
      )}

      {destinations.length === 0 && !destFormOpen ? (
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
                    {dest.last_run_at && (
                      <p className="flex items-center gap-1">
                        <Clock className="size-3" />
                        {formatDateTime(dest.last_run_at, i18n.language)}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex shrink-0 gap-1">
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
