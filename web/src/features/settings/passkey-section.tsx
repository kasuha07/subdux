import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { api } from "@/lib/api"
import { createPasskeyCredential, isPasskeySupported, type CredentialCreationJSON } from "@/lib/passkey"
import { getPasskeyErrorMessage } from "@/lib/passkey-error"
import { toast } from "sonner"
import type { PasskeyBeginResult, PasskeyCredential } from "@/types"
import ReauthDialog from "@/features/admin/reauth-dialog"

export default function PasskeySection() {
  const { t } = useTranslation()
  const [passkeys, setPasskeys] = useState<PasskeyCredential[]>([])
  const [loading, setLoading] = useState(true)
  const [registering, setRegistering] = useState(false)
  const [deletingID, setDeletingID] = useState<number | null>(null)
  const [name, setName] = useState("")
  const [error, setError] = useState("")
  const [reauthOpen, setReauthOpen] = useState(false)
  const [deleteReauthOpen, setDeleteReauthOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<PasskeyCredential | null>(null)

  const passkeySupported = isPasskeySupported()
  const dateFormatter = useMemo(
    () => new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }),
    []
  )

  useEffect(() => {
    api.get<PasskeyCredential[]>("/auth/passkeys")
      .then((items) => setPasskeys(items ?? []))
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  // Adding a passkey is gated behind step-up re-authentication: the form submit
  // only opens the reauth dialog, and the actual WebAuthn registration runs once
  // a factor is verified and a single-use ticket is minted.
  function handleRegister(e: FormEvent) {
    e.preventDefault()
    setError("")
    if (!passkeySupported) {
      setError(t("settings.passkeys.unsupported"))
      return
    }
    setReauthOpen(true)
  }

  async function registerWithTicket(reauthTicket: string) {
    setError("")
    setRegistering(true)
    try {
      const begin = await api.post<PasskeyBeginResult<CredentialCreationJSON>>("/auth/passkeys/register/start", {
        name: name.trim(),
      })
      const credential = await createPasskeyCredential(begin.options)
      const created = await api.post<PasskeyCredential>("/auth/passkeys/register/finish", {
        session_id: begin.session_id,
        credential,
        reauth_ticket: reauthTicket,
      })
      setPasskeys((prev) => [created, ...prev])
      setName("")
      toast.success(t("settings.passkeys.registerSuccess"))
    } catch (err) {
      setError(getPasskeyErrorMessage(err, t, "settings.passkeys.registerError"))
    } finally {
      setRegistering(false)
    }
  }

  async function handleDelete(passkey: PasskeyCredential) {
    if (!window.confirm(t("settings.passkeys.deleteConfirm"))) {
      return
    }

    setError("")
    setDeleteTarget(passkey)
    setDeleteReauthOpen(true)
  }

  async function deleteWithTicket(reauthTicket: string) {
    if (!deleteTarget) {
      return
    }

    const target = deleteTarget
    setDeletingID(target.id)
    try {
      await api.delete(`/auth/passkeys/${target.id}`, {
        headers: { "X-Reauth-Ticket": reauthTicket },
      })
      setPasskeys((prev) => prev.filter((item) => item.id !== target.id))
      toast.success(t("settings.passkeys.deleteSuccess"))
    } catch (err) {
      setError(getPasskeyErrorMessage(err, t, "settings.passkeys.deleteError"))
    } finally {
      setDeletingID(null)
      setDeleteTarget(null)
    }
  }

  function formatDate(value: string | null): string {
    if (!value) {
      return t("settings.passkeys.neverUsed")
    }
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) {
      return value
    }
    return dateFormatter.format(date)
  }

  return (
    <div className="space-y-3">
      <div>
        <div className="flex items-center gap-2">
          <h3 className="text-base font-semibold tracking-tight select-none">{t("settings.passkeys.title")}</h3>
          <Badge variant="secondary" className="text-xs">{passkeys.length}</Badge>
        </div>
        <p className="text-sm text-muted-foreground mt-0.5">{t("settings.passkeys.description")}</p>
      </div>

      {!passkeySupported && (
        <div className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground">
          {t("settings.passkeys.unsupported")}
        </div>
      )}

      {error && (
        <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
          {error}
        </div>
      )}

      <form onSubmit={(e) => void handleRegister(e)} className="space-y-1 max-w-sm">
        <Label htmlFor="passkey-name">{t("settings.passkeys.nameLabel")}</Label>
        <div className="flex items-center gap-2">
          <Input
            id="passkey-name"
            placeholder={t("settings.passkeys.namePlaceholder")}
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={255}
            className="max-w-56"
          />
          <Button type="submit" size="sm" variant="outline" disabled={registering || !passkeySupported}>
            {registering ? t("settings.passkeys.registering") : t("settings.passkeys.register")}
          </Button>
        </div>
      </form>

      <ReauthDialog
        operation="add_passkey"
        open={reauthOpen}
        onOpenChange={setReauthOpen}
        onVerified={async (ticket) => {
          await registerWithTicket(ticket)
        }}
        title={t("settings.passkeys.reauth.title")}
        description={t("settings.passkeys.reauth.description")}
      />
      <ReauthDialog
        operation="delete_passkey"
        open={deleteReauthOpen}
        onOpenChange={setDeleteReauthOpen}
        onVerified={async (ticket) => {
          await deleteWithTicket(ticket)
        }}
        title={t("settings.passkeys.deleteReauth.title")}
        description={t("settings.passkeys.deleteReauth.description")}
        confirmVariant="destructive"
      />

      <div className="space-y-2">
        {loading && (
          <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
        )}
        {!loading && passkeys.length === 0 && (
          <p className="text-sm text-muted-foreground">{t("settings.passkeys.empty")}</p>
        )}
        {passkeys.map((passkey) => (
          <div key={passkey.id} className="flex items-start justify-between gap-3 rounded-md border bg-card px-3 py-2">
            <div className="min-w-0">
              <p className="text-sm font-medium">{passkey.name || t("settings.passkeys.defaultName")}</p>
              <p className="text-xs text-muted-foreground">
                {t("settings.passkeys.createdAt")}: {formatDate(passkey.created_at)}
              </p>
              <p className="text-xs text-muted-foreground">
                {t("settings.passkeys.lastUsedAt")}: {formatDate(passkey.last_used_at)}
              </p>
            </div>
            <Button
              size="sm"
              variant="ghost"
              className="text-destructive hover:text-destructive"
              disabled={deletingID === passkey.id || deleteReauthOpen}
              onClick={() => void handleDelete(passkey)}
            >
              {deletingID === passkey.id ? t("settings.passkeys.deleting") : t("settings.passkeys.delete")}
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}
