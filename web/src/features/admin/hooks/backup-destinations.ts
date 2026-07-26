import type { BackupDestination } from "@/types"

// Shared pure helpers for the backup-destinations UI. These live outside the
// component so the form/list logic and its tests exercise the SAME code rather
// than a copy that can silently drift.

// Secret inputs render through the shared SecretInput component (the SMTP
// password treatment): the form state only ever holds what the admin actually
// typed, and the "configured" mask is purely presentational. The stored secret
// therefore never round-trips through the form, not even as a sentinel value.

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
  skip_tls_verify?: boolean
  retention_count?: number
}

export interface WebDAVDestConfig extends ScheduleDestConfig {
  url?: string
  path?: string
  username?: string
  password?: string
  skip_tls_verify?: boolean
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

export interface LocalTransportFields {
  dir: string
  retentionCount: number
}

export interface LocalConfigFields extends LocalTransportFields, ScheduleConfigFields {}

function localTransportEntries(fields: LocalTransportFields): Record<string, unknown> {
  return {
    dir: fields.dir,
    retention_count: fields.retentionCount,
  }
}

// buildLocalConfig serialises the local form fields into the config JSON the
// backend expects. It takes a fields object rather than positional arguments so
// it matches buildS3Config/buildWebDAVConfig now that all three carry the
// shared schedule block.
export function buildLocalConfig(fields: LocalConfigFields): string {
  return JSON.stringify({
    ...localTransportEntries(fields),
    ...scheduleConfigEntries(fields),
  })
}

// buildLocalProbeConfig is the connectivity-test half of buildLocalConfig: same
// transport fields, no schedule block. See buildDestinationProbeRequest for why
// the two are split.
export function buildLocalProbeConfig(fields: LocalTransportFields): string {
  return JSON.stringify(localTransportEntries(fields))
}

export function parseS3Config(dest: BackupDestination): S3DestConfig {
  if (dest.type !== "s3") return {}
  try {
    return JSON.parse(dest.config) as S3DestConfig
  } catch {
    return {}
  }
}

export interface S3TransportFields {
  endpoint: string
  region: string
  bucket: string
  prefix: string
  accessKeyId: string
  secretAccessKey: string
  useSsl: boolean
  usePathStyle: boolean
  skipTlsVerify: boolean
  retentionCount: number
}

export interface S3ConfigFields extends S3TransportFields, ScheduleConfigFields {}

function s3TransportEntries(fields: S3TransportFields): Record<string, unknown> {
  const config: Record<string, unknown> = {
    endpoint: fields.endpoint,
    bucket: fields.bucket,
    access_key_id: fields.accessKeyId,
    secret_access_key: fields.secretAccessKey,
    use_ssl: fields.useSsl,
    use_path_style: fields.usePathStyle,
    skip_tls_verify: fields.skipTlsVerify,
    retention_count: fields.retentionCount,
  }
  if (fields.region) config.region = fields.region
  if (fields.prefix) config.prefix = fields.prefix
  return config
}

// buildS3Config serialises the s3 form fields into the config JSON the backend
// expects. Empty region/prefix are omitted so the stored config stays clean.
// skip_tls_verify is always written, like the other switches: omitting it when
// off would leave a previously enabled destination skipping verification after
// the admin turned the toggle back off.
export function buildS3Config(fields: S3ConfigFields): string {
  return JSON.stringify({
    ...s3TransportEntries(fields),
    ...scheduleConfigEntries(fields),
  })
}

export function buildS3ProbeConfig(fields: S3TransportFields): string {
  return JSON.stringify(s3TransportEntries(fields))
}

export function parseWebDAVConfig(dest: BackupDestination): WebDAVDestConfig {
  if (dest.type !== "webdav") return {}
  try {
    return JSON.parse(dest.config) as WebDAVDestConfig
  } catch {
    return {}
  }
}

export interface WebDAVTransportFields {
  url: string
  path: string
  username: string
  password: string
  skipTlsVerify: boolean
  retentionCount: number
}

export interface WebDAVConfigFields extends WebDAVTransportFields, ScheduleConfigFields {}

function webdavTransportEntries(fields: WebDAVTransportFields): Record<string, unknown> {
  const config: Record<string, unknown> = {
    url: fields.url,
    password: fields.password,
    skip_tls_verify: fields.skipTlsVerify,
    retention_count: fields.retentionCount,
  }
  if (fields.path) config.path = fields.path
  if (fields.username) config.username = fields.username
  return config
}

// buildWebDAVConfig serialises the webdav form fields into the config JSON the
// backend expects. Empty path/username are omitted so the stored config stays
// clean (mirrors s3's omit-empty treatment for region/prefix); skip_tls_verify
// is always written for the same reason it is in buildS3Config.
export function buildWebDAVConfig(fields: WebDAVConfigFields): string {
  return JSON.stringify({
    ...webdavTransportEntries(fields),
    ...scheduleConfigEntries(fields),
  })
}

export function buildWebDAVProbeConfig(fields: WebDAVTransportFields): string {
  return JSON.stringify(webdavTransportEntries(fields))
}

// DestinationProbeFormValues is the slice of the destination form a connectivity
// test reads. It is declared structurally rather than imported from the form
// component so this module keeps depending on nothing but the DTO, and so the
// list doubles as a statement of what a probe does NOT look at: `enabled` and
// the whole schedule block, none of which affect whether the target answers.
//
// DestinationFormValues satisfies it, so renaming or removing a form field fails
// to compile at the call site. Adding one does not, and does not need to: the
// guard against a new transport field reaching only the save path is that both
// builders take the same *TransportFields types, so it breaks both or neither.
export interface DestinationProbeFormValues {
  type: string
  dir: string
  retention: number
  endpoint: string
  region: string
  bucket: string
  prefix: string
  accessKeyId: string
  secretAccessKey: string
  useSsl: boolean
  usePathStyle: boolean
  webdavUrl: string
  webdavPath: string
  webdavUsername: string
  webdavPassword: string
  skipTlsVerify: boolean
}

export interface SecretResolution {
  value: string
  cleared_secret_fields?: string[]
}

// resolveSecretUpdate encodes the three secret paths on save for any named
// secret field:
//   - a new value typed                      -> replace: send the value
//   - explicitly cleared (configured, admin
//     pressed the clear affordance)          -> wipe: send "" + cleared field
//   - left untouched/empty                   -> preserve: send ""
// `fieldName` is the backend field identifier (e.g. "password"). `isConfigured`
// is whether the server currently holds a secret for this field; `cleared` is
// whether the admin explicitly asked for the stored secret to be dropped.
// A typed value always wins over a stale cleared flag: replacing after clearing
// is a change of mind, not a request to do both.
//
// The S3 secret_access_key deliberately never goes through the cleared path
// (its field has no clear affordance): target_s3.go's parseS3Config rejects a
// config with a blank secret (ErrS3CredentialsRequired), so "S3 with no
// secret" is never a valid saved state. The WebDAV password is the one secret
// with a clear affordance — anonymous WebDAV is legitimate. The encryption
// password goes through resolveEncryptionSecretUpdate below, which derives
// clearing from the encrypt toggle instead.
export function resolveSecretUpdate(
  fieldName: string,
  secretFieldValue: string,
  cleared: boolean,
  isConfigured: boolean
): SecretResolution {
  if (secretFieldValue.trim() !== "") {
    return { value: secretFieldValue }
  }
  if (isConfigured && cleared) {
    return { value: "", cleared_secret_fields: [fieldName] }
  }
  return { value: "" }
}

// resolveEncryptionSecretUpdate ties the encryption password's lifecycle to the
// encrypt toggle instead of a clear affordance: an enabled toggle follows the
// ordinary replace-or-preserve rules, while saving with encryption off drops
// any stored password. "Encryption on without a password" is rejected by the
// backend (ErrBackupEncryptionPasswordRequired), so a clear button could only
// ever produce a failed save; turning encryption off is the one real way to
// stop needing the secret, and keeping it stored past that point would
// contradict the field's own hygiene story.
export function resolveEncryptionSecretUpdate(
  encryptEnabled: boolean,
  secretFieldValue: string,
  isConfigured: boolean
): SecretResolution {
  if (encryptEnabled) {
    return resolveSecretUpdate("encryption_password", secretFieldValue, false, isConfigured)
  }
  if (isConfigured) {
    return { value: "", cleared_secret_fields: ["encryption_password"] }
  }
  return { value: "" }
}

// DestinationProbeRequest is the body of the unsaved-config connectivity test,
// which is how the add/edit dialog checks a destination before committing to it.
export interface DestinationProbeRequest {
  type: string
  config: string
  destination_id?: number
  cleared_secret_fields?: string[]
}

// buildDestinationProbeRequest assembles a connectivity test for whatever is in
// the add/edit form right now, saved or not.
//
// The config it sends carries transport fields only. The schedule block — when
// the plan fires, whether assets are included, and the archive encryption
// password — never reaches the storage target, so a reachability check neither
// needs those fields nor should make the admin complete them first.
//
// Secrets follow the same three paths as a save (see resolveSecretUpdate): a
// typed value is sent, an explicitly cleared one is reported as cleared, and a
// blank one is left for the server to fill in from the saved destination named
// by destination_id. The server only performs that inheritance while the
// credential still points at the address it already reaches, so an endpoint
// edited in the same breath as a blank secret comes back asking for the secret
// rather than quietly shipping it somewhere new.
export function buildDestinationProbeRequest(
  values: DestinationProbeFormValues,
  editTarget: BackupDestination | null,
  clearedWebdavSecret: boolean
): DestinationProbeRequest {
  const secretIsConfigured = (field: string) =>
    editTarget?.configured_secret_fields.includes(field) ?? false

  let config: string
  let clearedSecretFields: string[] | undefined

  if (values.type === "s3") {
    // Replace-only, exactly like the save path: the s3 secret has no clear
    // affordance, so cleared is always false.
    const secret = resolveSecretUpdate(
      "secret_access_key",
      values.secretAccessKey,
      false,
      secretIsConfigured("secret_access_key")
    )
    config = buildS3ProbeConfig({
      endpoint: values.endpoint,
      region: values.region,
      bucket: values.bucket,
      prefix: values.prefix,
      accessKeyId: values.accessKeyId,
      secretAccessKey: secret.value,
      useSsl: values.useSsl,
      usePathStyle: values.usePathStyle,
      skipTlsVerify: values.skipTlsVerify,
      retentionCount: values.retention,
    })
  } else if (values.type === "webdav") {
    const password = resolveSecretUpdate(
      "password",
      values.webdavPassword,
      clearedWebdavSecret,
      secretIsConfigured("password")
    )
    clearedSecretFields = password.cleared_secret_fields
    config = buildWebDAVProbeConfig({
      url: values.webdavUrl,
      path: values.webdavPath,
      username: values.webdavUsername,
      password: password.value,
      skipTlsVerify: values.skipTlsVerify,
      retentionCount: values.retention,
    })
  } else {
    config = buildLocalProbeConfig({
      dir: values.dir,
      retentionCount: values.retention,
    })
  }

  const request: DestinationProbeRequest = { type: values.type, config }
  if (editTarget) request.destination_id = editTarget.id
  if (clearedSecretFields) request.cleared_secret_fields = clearedSecretFields
  return request
}

export function mutationSucceeded<T>(value: T | undefined): value is T {
  return value !== undefined && value !== null
}

export function destStatusVariant(status: string): "secondary" | "destructive" | "outline" {
  if (status === "success") return "secondary"
  if (status === "failed") return "destructive"
  return "outline"
}
