import type { BackupDestination } from "@/types"

// Shared pure helpers for the backup-destinations UI. These live outside the
// component so the form/list logic and its tests exercise the SAME code rather
// than a copy that can silently drift.

// DESTINATION_SECRET_MASK is the placeholder shown in a secret input when a
// secret is already stored server-side (the backend never returns the value).
export const DESTINATION_SECRET_MASK = "••••••••"

// ScheduleDestConfig is the per-destination backup plan shared by every
// destination type: each destination carries its own daily schedule rather than
// inheriting one global one.
interface ScheduleDestConfig {
  time_of_day?: string
  include_assets?: boolean
  encrypt_enabled?: boolean
  encryption_password?: string
}

export interface LocalDestConfig extends ScheduleDestConfig {
  dir?: string
  retention_count?: number
}

export interface S3DestConfig extends ScheduleDestConfig {
  endpoint?: string
  region?: string
  bucket?: string
  prefix?: string
  access_key_id?: string
  secret_access_key?: string
  use_ssl?: boolean
  use_path_style?: boolean
  retention_count?: number
}

export interface WebDAVDestConfig extends ScheduleDestConfig {
  url?: string
  path?: string
  username?: string
  password?: string
  retention_count?: number
}

// ScheduleConfigFields are the schedule form values every build*Config call
// carries. They are ALWAYS serialised (never omit-empty) so a cleared time or a
// toggled-off switch actually overwrites the stored value; encryptionPassword is
// the one exception and follows resolveSecretUpdate's preserve/clear/replace
// rules like the other destination secrets.
export interface ScheduleConfigFields {
  timeOfDay: string
  includeAssets: boolean
  encryptEnabled: boolean
  encryptionPassword: string
}

function scheduleConfigEntries(fields: ScheduleConfigFields): Record<string, unknown> {
  return {
    time_of_day: fields.timeOfDay,
    include_assets: fields.includeAssets,
    encrypt_enabled: fields.encryptEnabled,
    encryption_password: fields.encryptionPassword,
  }
}

export function parseLocalConfig(dest: BackupDestination): LocalDestConfig {
  if (dest.type !== "local") return {}
  try {
    return JSON.parse(dest.config) as LocalDestConfig
  } catch {
    return {}
  }
}

export interface LocalConfigFields extends ScheduleConfigFields {
  dir: string
  retentionCount: number
}

// buildLocalConfig serialises the local form fields into the config JSON the
// backend expects. It takes a fields object rather than positional arguments so
// it matches buildS3Config/buildWebDAVConfig now that all three carry the
// shared schedule block.
export function buildLocalConfig(fields: LocalConfigFields): string {
  return JSON.stringify({
    dir: fields.dir,
    retention_count: fields.retentionCount,
    ...scheduleConfigEntries(fields),
  })
}

export function parseS3Config(dest: BackupDestination): S3DestConfig {
  if (dest.type !== "s3") return {}
  try {
    return JSON.parse(dest.config) as S3DestConfig
  } catch {
    return {}
  }
}

export interface S3ConfigFields extends ScheduleConfigFields {
  endpoint: string
  region: string
  bucket: string
  prefix: string
  accessKeyId: string
  secretAccessKey: string
  useSsl: boolean
  usePathStyle: boolean
  retentionCount: number
}

// buildS3Config serialises the s3 form fields into the config JSON the backend
// expects. Empty region/prefix are omitted so the stored config stays clean.
export function buildS3Config(fields: S3ConfigFields): string {
  const config: Record<string, unknown> = {
    endpoint: fields.endpoint,
    bucket: fields.bucket,
    access_key_id: fields.accessKeyId,
    secret_access_key: fields.secretAccessKey,
    use_ssl: fields.useSsl,
    use_path_style: fields.usePathStyle,
    retention_count: fields.retentionCount,
    ...scheduleConfigEntries(fields),
  }
  if (fields.region) config.region = fields.region
  if (fields.prefix) config.prefix = fields.prefix
  return JSON.stringify(config)
}

export function parseWebDAVConfig(dest: BackupDestination): WebDAVDestConfig {
  if (dest.type !== "webdav") return {}
  try {
    return JSON.parse(dest.config) as WebDAVDestConfig
  } catch {
    return {}
  }
}

export interface WebDAVConfigFields extends ScheduleConfigFields {
  url: string
  path: string
  username: string
  password: string
  retentionCount: number
}

// buildWebDAVConfig serialises the webdav form fields into the config JSON the
// backend expects. Empty path/username are omitted so the stored config stays
// clean (mirrors s3's omit-empty treatment for region/prefix).
export function buildWebDAVConfig(fields: WebDAVConfigFields): string {
  const config: Record<string, unknown> = {
    url: fields.url,
    password: fields.password,
    retention_count: fields.retentionCount,
    ...scheduleConfigEntries(fields),
  }
  if (fields.path) config.path = fields.path
  if (fields.username) config.username = fields.username
  return JSON.stringify(config)
}

export interface SecretResolution {
  value: string
  cleared_secret_fields?: string[]
}

// resolveSecretUpdate encodes the three secret paths on save for any named
// secret field:
//   - mask still shown (configured, untouched)  -> preserve: send ""
//   - explicitly cleared (configured, editing, emptied)
//                                                -> wipe: send "" + cleared field
//   - a new value typed (or on create)           -> replace: send the value
// `fieldName` is the backend field identifier (e.g. "secret_access_key",
// "password"). `isConfigured` is whether the server currently holds a secret
// for this field; `editing` is whether the admin switched the field into edit
// mode.
export function resolveSecretUpdate(
  fieldName: string,
  secretFieldValue: string,
  editing: boolean,
  isConfigured: boolean
): SecretResolution {
  if (isConfigured && secretFieldValue === DESTINATION_SECRET_MASK) {
    return { value: "" }
  }
  if (isConfigured && secretFieldValue === "" && editing) {
    return { value: "", cleared_secret_fields: [fieldName] }
  }
  return { value: secretFieldValue }
}

export interface S3SecretResolution {
  secret_access_key: string
  cleared_secret_fields?: string[]
}

// resolveS3SecretUpdate wraps resolveSecretUpdate to make the S3 secret
// REPLACEMENT-ONLY: emptying the field while editing preserves the stored
// secret instead of clearing it. The WebDAV password and the per-destination
// encryption_password, which go through resolveSecretUpdate directly, can be
// cleared — turning encryption off legitimately drops the password. That
// asymmetry is deliberate and follows from backend validation:
//   - internal/service/backup/target_s3.go -> parseS3Config rejects a config
//     whose secret_access_key is blank (ErrS3CredentialsRequired), so "S3 with
//     no secret" is never a valid saved state; offering a clear could only ever
//     produce a rejected save.
//   - internal/service/backup/target_webdav.go -> parseWebDAVConfig does not
//     require a password, and authorize() only sends basic auth when the
//     username or password is non-empty. Anonymous WebDAV is therefore
//     legitimate, so clearing the password is a real user intent.
export function resolveS3SecretUpdate(
  secretFieldValue: string,
  editing: boolean,
  isConfigured: boolean
): S3SecretResolution {
  const r =
    isConfigured && editing && secretFieldValue === ""
      ? { value: "" }
      : resolveSecretUpdate("secret_access_key", secretFieldValue, editing, isConfigured)
  return { secret_access_key: r.value, cleared_secret_fields: r.cleared_secret_fields }
}

export function mutationSucceeded<T>(value: T | undefined): value is T {
  return value !== undefined && value !== null
}

export function destStatusVariant(status: string): "secondary" | "destructive" | "outline" {
  if (status === "success") return "secondary"
  if (status === "failed") return "destructive"
  return "outline"
}
