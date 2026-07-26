import { useTranslation } from "react-i18next"

import { SecretInput } from "@/components/secret-input"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import type { BackupDestination } from "@/types"

export type DestinationType = "local" | "s3" | "webdav"

// DestinationFormValues is the whole editable surface of one destination. It is
// held as a single object so opening the create/edit form resets every field of
// the other destination types in one assignment.
export interface DestinationFormValues {
  type: DestinationType
  enabled: boolean
  // local
  dir: string
  // shared by all types
  retention: number
  // s3
  endpoint: string
  region: string
  bucket: string
  prefix: string
  accessKeyId: string
  secretAccessKey: string
  useSsl: boolean
  usePathStyle: boolean
  // webdav
  webdavUrl: string
  webdavPath: string
  webdavUsername: string
  webdavPassword: string
  // shared by the two network types (s3, webdav); local writes to disk and has
  // no TLS to relax
  skipTlsVerify: boolean
  // schedule — every destination is its own backup plan, so all types carry these
  timeOfDay: string
  includeAssets: boolean
  encryptEnabled: boolean
  encryptionPassword: string
}

interface AdminBackupDestinationFormFieldsProps {
  values: DestinationFormValues
  onValuesChange: (patch: Partial<DestinationFormValues>) => void
  // The destination being edited, or null when creating. Editing locks the type
  // and switches configured secrets into the masked SecretInput presentation:
  // the server never returns the stored value, so the form only ever holds what
  // the admin typed. The cleared flag marks the stored webdav password for
  // removal on save; the s3 secret is replace-only, and the encryption password
  // is cleared by saving with the encrypt toggle off.
  editTarget: BackupDestination | null
  clearedWebdavSecret: boolean
  onClearWebdavSecret: () => void
}

export default function AdminBackupDestinationFormFields({
  values,
  onValuesChange,
  editTarget,
  clearedWebdavSecret,
  onClearWebdavSecret,
}: AdminBackupDestinationFormFieldsProps) {
  const { t } = useTranslation()

  function isConfigured(field: string) {
    return editTarget?.configured_secret_fields.includes(field) ?? false
  }

  // Both network types carry the same switch against the same config field, so
  // it is written once here rather than duplicated into each type's branch.
  const skipTlsVerifyToggle = (
    <div className="flex items-center justify-between gap-4 rounded-md border p-3">
      <div className="space-y-0.5">
        <Label htmlFor="dest-skip-tls-verify">
          {t("admin.backup.destinations.skipTlsVerify")}
        </Label>
        <p className="text-xs text-muted-foreground">
          {t("admin.backup.destinations.skipTlsVerifyDescription")}
        </p>
      </div>
      <Switch
        id="dest-skip-tls-verify"
        checked={values.skipTlsVerify}
        onCheckedChange={(checked) => onValuesChange({ skipTlsVerify: checked })}
      />
    </div>
  )

  return (
    <>
      <div className="space-y-2">
        <Label htmlFor="dest-type">{t("admin.backup.destinations.type")}</Label>
        {editTarget ? (
          <p className="text-sm text-muted-foreground">
            {values.type === "s3"
              ? t("admin.backup.destinations.typeS3")
              : values.type === "webdav"
                ? t("admin.backup.destinations.typeWebDAV")
                : t("admin.backup.destinations.typeLocal")}
          </p>
        ) : (
          <Select value={values.type} onValueChange={(v) => onValuesChange({ type: v as DestinationType })}>
            <SelectTrigger id="dest-type" className="w-40">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="local">{t("admin.backup.destinations.typeLocal")}</SelectItem>
              <SelectItem value="s3">{t("admin.backup.destinations.typeS3")}</SelectItem>
              <SelectItem value="webdav">{t("admin.backup.destinations.typeWebDAV")}</SelectItem>
            </SelectContent>
          </Select>
        )}
      </div>

      <div className="flex items-center justify-between gap-4 rounded-md border p-3">
        <div className="space-y-0.5">
          <Label htmlFor="dest-enabled">{t("admin.backup.destinations.enabled")}</Label>
        </div>
        <Switch
          id="dest-enabled"
          checked={values.enabled}
          onCheckedChange={(checked) => onValuesChange({ enabled: checked })}
        />
      </div>

      {values.type === "local" && (
        <>
          <div className="space-y-2">
            <Label htmlFor="dest-dir">{t("admin.backup.localDir")}</Label>
            <Input
              id="dest-dir"
              value={values.dir}
              onChange={(event) => onValuesChange({ dir: event.target.value })}
              placeholder={t("admin.backup.localDirPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">{t("admin.backup.localDirDescription")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-retention">{t("admin.backup.retentionCount")}</Label>
            <Input
              id="dest-retention"
              type="number"
              min={1}
              step={1}
              className="w-32"
              value={values.retention}
              onChange={(event) => {
                const next = parseInt(event.target.value, 10)
                if (!Number.isNaN(next) && next >= 1) {
                  onValuesChange({ retention: next })
                }
              }}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.retentionCountDescription")}
            </p>
          </div>
        </>
      )}

      {values.type === "s3" && (
        <>
          <div className="space-y-2">
            <Label htmlFor="dest-s3-endpoint">{t("admin.backup.destinations.s3Endpoint")}</Label>
            <Input
              id="dest-s3-endpoint"
              value={values.endpoint}
              onChange={(event) => onValuesChange({ endpoint: event.target.value })}
              placeholder={t("admin.backup.destinations.s3EndpointPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">{t("admin.backup.destinations.s3EndpointDescription")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-s3-region">{t("admin.backup.destinations.s3Region")}</Label>
            <Input
              id="dest-s3-region"
              value={values.region}
              onChange={(event) => onValuesChange({ region: event.target.value })}
              placeholder={t("admin.backup.destinations.s3RegionPlaceholder")}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-s3-bucket">{t("admin.backup.destinations.s3Bucket")}</Label>
            <Input
              id="dest-s3-bucket"
              value={values.bucket}
              onChange={(event) => onValuesChange({ bucket: event.target.value })}
              placeholder={t("admin.backup.destinations.s3BucketPlaceholder")}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-s3-prefix">{t("admin.backup.destinations.s3Prefix")}</Label>
            <Input
              id="dest-s3-prefix"
              value={values.prefix}
              onChange={(event) => onValuesChange({ prefix: event.target.value })}
              placeholder={t("admin.backup.destinations.s3PrefixPlaceholder")}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-s3-access-key-id">{t("admin.backup.destinations.s3AccessKeyId")}</Label>
            <Input
              id="dest-s3-access-key-id"
              value={values.accessKeyId}
              onChange={(event) => onValuesChange({ accessKeyId: event.target.value })}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-s3-secret">{t("admin.backup.destinations.s3SecretAccessKey")}</Label>
            <SecretInput
              id="dest-s3-secret"
              type="password"
              autoComplete="new-password"
              value={values.secretAccessKey}
              configured={isConfigured("secret_access_key")}
              onValueChange={(value) => onValuesChange({ secretAccessKey: value })}
            />
            {isConfigured("secret_access_key") && (
              <p className="text-xs text-muted-foreground">
                {t("admin.backup.destinations.s3SecretConfigured")}
              </p>
            )}
          </div>

          <div className="flex items-center justify-between gap-4 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label htmlFor="dest-s3-use-ssl">{t("admin.backup.destinations.s3UseSsl")}</Label>
              <p className="text-xs text-muted-foreground">{t("admin.backup.destinations.s3UseSslDescription")}</p>
            </div>
            <Switch
              id="dest-s3-use-ssl"
              checked={values.useSsl}
              onCheckedChange={(checked) => onValuesChange({ useSsl: checked })}
            />
          </div>

          <div className="flex items-center justify-between gap-4 rounded-md border p-3">
            <div className="space-y-0.5">
              <Label htmlFor="dest-s3-use-path-style">{t("admin.backup.destinations.s3UsePathStyle")}</Label>
              <p className="text-xs text-muted-foreground">{t("admin.backup.destinations.s3UsePathStyleDescription")}</p>
            </div>
            <Switch
              id="dest-s3-use-path-style"
              checked={values.usePathStyle}
              onCheckedChange={(checked) => onValuesChange({ usePathStyle: checked })}
            />
          </div>

          {skipTlsVerifyToggle}

          <div className="space-y-2">
            <Label htmlFor="dest-s3-retention">{t("admin.backup.retentionCount")}</Label>
            <Input
              id="dest-s3-retention"
              type="number"
              min={1}
              step={1}
              className="w-32"
              value={values.retention}
              onChange={(event) => {
                const next = parseInt(event.target.value, 10)
                if (!Number.isNaN(next) && next >= 1) {
                  onValuesChange({ retention: next })
                }
              }}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.retentionCountDescription")}
            </p>
          </div>
        </>
      )}

      {values.type === "webdav" && (
        <>
          <div className="space-y-2">
            <Label htmlFor="dest-webdav-url">{t("admin.backup.destinations.webdavUrl")}</Label>
            <Input
              id="dest-webdav-url"
              value={values.webdavUrl}
              onChange={(event) => onValuesChange({ webdavUrl: event.target.value })}
              placeholder={t("admin.backup.destinations.webdavUrlPlaceholder")}
            />
            <p className="text-xs text-muted-foreground">{t("admin.backup.destinations.webdavUrlDescription")}</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-webdav-path">{t("admin.backup.destinations.webdavPath")}</Label>
            <Input
              id="dest-webdav-path"
              value={values.webdavPath}
              onChange={(event) => onValuesChange({ webdavPath: event.target.value })}
              placeholder={t("admin.backup.destinations.webdavPathPlaceholder")}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-webdav-username">{t("admin.backup.destinations.webdavUsername")}</Label>
            <Input
              id="dest-webdav-username"
              value={values.webdavUsername}
              onChange={(event) => onValuesChange({ webdavUsername: event.target.value })}
              autoComplete="username"
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="dest-webdav-password">{t("admin.backup.destinations.webdavPassword")}</Label>
            <div className="flex items-center gap-2">
              <SecretInput
                id="dest-webdav-password"
                type="password"
                autoComplete="new-password"
                className="flex-1"
                value={values.webdavPassword}
                configured={isConfigured("password") && !clearedWebdavSecret}
                onValueChange={(value) => onValuesChange({ webdavPassword: value })}
              />
              {isConfigured("password") && !clearedWebdavSecret && (
                <Button type="button" size="sm" variant="outline" onClick={onClearWebdavSecret}>
                  {t("admin.backup.destinations.webdavPasswordClear")}
                </Button>
              )}
            </div>
            {isConfigured("password") &&
              (clearedWebdavSecret && values.webdavPassword.trim() === "" ? (
                <p className="text-xs text-muted-foreground">
                  {t("admin.backup.destinations.secretClearPending")}
                </p>
              ) : !clearedWebdavSecret ? (
                <p className="text-xs text-muted-foreground">
                  {t("admin.backup.destinations.webdavPasswordConfigured")}
                </p>
              ) : null)}
          </div>

          {skipTlsVerifyToggle}

          <div className="space-y-2">
            <Label htmlFor="dest-webdav-retention">{t("admin.backup.retentionCount")}</Label>
            <Input
              id="dest-webdav-retention"
              type="number"
              min={1}
              step={1}
              className="w-32"
              value={values.retention}
              onChange={(event) => {
                const next = parseInt(event.target.value, 10)
                if (!Number.isNaN(next) && next >= 1) {
                  onValuesChange({ retention: next })
                }
              }}
            />
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.retentionCountDescription")}
            </p>
          </div>
        </>
      )}

      {/* Schedule block. Every destination is a self-contained backup plan, so
          these fields are shared by all types instead of being repeated inside
          each type's branch above. */}
      <div className="space-y-2">
        <Label htmlFor="dest-time-of-day">
          {t("admin.backup.destinations.scheduleTimeOfDay")}
        </Label>
        <Input
          id="dest-time-of-day"
          type="time"
          className="w-40"
          value={values.timeOfDay}
          onChange={(event) => onValuesChange({ timeOfDay: event.target.value })}
        />
        <p className="text-xs text-muted-foreground">
          {t("admin.backup.destinations.scheduleTimeOfDayDescription")}
        </p>
      </div>

      <div className="flex items-center justify-between gap-4 rounded-md border p-3">
        <div className="space-y-0.5">
          <Label htmlFor="dest-include-assets">
            {t("admin.backup.destinations.scheduleIncludeAssets")}
          </Label>
          <p className="text-xs text-muted-foreground">
            {t("admin.backup.destinations.scheduleIncludeAssetsDescription")}
          </p>
        </div>
        <Switch
          id="dest-include-assets"
          checked={values.includeAssets}
          onCheckedChange={(checked) => onValuesChange({ includeAssets: checked })}
        />
      </div>

      <div className="flex items-center justify-between gap-4 rounded-md border p-3">
        <div className="space-y-0.5">
          <Label htmlFor="dest-encrypt-enabled">
            {t("admin.backup.destinations.scheduleEncrypt")}
          </Label>
          <p className="text-xs text-muted-foreground">
            {t("admin.backup.destinations.scheduleEncryptDescription")}
          </p>
        </div>
        <Switch
          id="dest-encrypt-enabled"
          checked={values.encryptEnabled}
          onCheckedChange={(checked) => onValuesChange({ encryptEnabled: checked })}
        />
      </div>

      {values.encryptEnabled && (
        <div className="space-y-2">
          <Label htmlFor="dest-encryption-password">
            {t("admin.backup.destinations.encryptionPassword")}
          </Label>
          {/* Same masked SecretInput presentation as the webdav password: the
              server blanks configured secrets, so the stored value can only be
              kept or replaced — never read back. There is no clear button; the
              backend requires a password while encryption is on, so the stored
              password is dropped by saving with the encrypt toggle off. */}
          <SecretInput
            id="dest-encryption-password"
            type="password"
            autoComplete="new-password"
            value={values.encryptionPassword}
            configured={isConfigured("encryption_password")}
            onValueChange={(value) => onValuesChange({ encryptionPassword: value })}
          />
          {isConfigured("encryption_password") && (
            <p className="text-xs text-muted-foreground">
              {t("admin.backup.destinations.encryptionPasswordConfigured")}
            </p>
          )}
        </div>
      )}
    </>
  )
}
