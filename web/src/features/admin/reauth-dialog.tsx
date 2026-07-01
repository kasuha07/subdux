import { useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { Fingerprint, KeyRound } from "lucide-react"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api } from "@/lib/api"
import {
  getPasskeyCredential,
  isPasskeySupported,
  type CredentialAssertionJSON,
} from "@/lib/passkey"
import { getPasskeyErrorMessage } from "@/lib/passkey-error"
import type { OIDCConfig, OIDCStartResponse, PasskeyBeginResult, ReauthMethods } from "@/types"

type ReauthOperation = "backup" | "restore"

interface ReauthDialogProps {
  operation: ReauthOperation
  open: boolean
  onOpenChange: (open: boolean) => void
  // Called once a factor is verified, receiving the single-use ticket to pass
  // to the sensitive endpoint. The dialog closes itself after invoking this.
  onVerified: (ticket: string) => void | Promise<void>
  title: string
  description: string
  confirmVariant?: "default" | "destructive"
}

export default function ReauthDialog({
  operation,
  open,
  onOpenChange,
  onVerified,
  title,
  description,
  confirmVariant = "default",
}: ReauthDialogProps) {
  const { t } = useTranslation()
  const [methods, setMethods] = useState<ReauthMethods>({ password: true, passkey: false, oidc: false })
  const [oidcConfig, setOidcConfig] = useState<OIDCConfig | null>(null)
  const [password, setPassword] = useState("")
  const [busy, setBusy] = useState(false)
  // Tracks a live OIDC popup so we don't open a second one and can react to the
  // user closing it manually.
  const oidcPopupRef = useRef<Window | null>(null)

  const passkeySupported = isPasskeySupported()

  useEffect(() => {
    if (!open) {
      return
    }
    // Discover which factors this user can present. Transient state (password,
    // busy) starts fresh because the parent remounts this dialog per open via a
    // `key`, so no in-effect reset is needed.
    let cancelled = false
    api
      .get<ReauthMethods>(`/admin/reauth/methods?operation=${operation}`)
      .then((result) => {
        if (!cancelled && result) {
          setMethods(result)
        }
      })
      .catch(() => {
        // Password is always a valid attempt; fall back to it if discovery fails.
        if (!cancelled) {
          setMethods({ password: true, passkey: false, oidc: false })
        }
      })
    // The provider name is only needed to label the OIDC button; failure is
    // non-fatal (the button falls back to a generic label).
    api
      .get<OIDCConfig>("/auth/oidc/config")
      .then((config) => {
        if (!cancelled && config) {
          setOidcConfig(config)
        }
      })
      .catch(() => void 0)
    return () => {
      cancelled = true
    }
  }, [open, operation])

  // If the dialog unmounts (parent closes it) while an OIDC popup is still open,
  // close the orphaned popup so it can't post back to a gone listener.
  useEffect(() => {
    return () => {
      oidcPopupRef.current?.close()
      oidcPopupRef.current = null
    }
  }, [])

  async function verified(ticket: string) {
    onOpenChange(false)
    await onVerified(ticket)
  }

  async function handlePasswordConfirm() {
    if (busy || password.trim() === "") {
      return
    }
    setBusy(true)
    try {
      const { ticket } = await api.post<{ ticket: string }>("/admin/reauth/password", {
        operation,
        password,
      })
      await verified(ticket)
    } catch {
      // api.post surfaces the backend message via toast; keep the dialog open
      // so the admin can correct the password.
      setBusy(false)
    }
  }

  async function handlePasskeyConfirm() {
    if (busy) {
      return
    }
    if (!passkeySupported) {
      toast.error(t("admin.backup.reauth.passkeyUnsupported"))
      return
    }
    setBusy(true)
    try {
      const begin = await api.post<PasskeyBeginResult<CredentialAssertionJSON>>(
        "/admin/reauth/passkey/start",
        { operation }
      )
      const credential = await getPasskeyCredential(begin.options)
      const { ticket } = await api.post<{ ticket: string }>("/admin/reauth/passkey/finish", {
        operation,
        session_id: begin.session_id,
        credential,
      })
      await verified(ticket)
    } catch (err) {
      // Backend failures already toast via api.post. Surface browser/WebAuthn
      // errors (user cancelled, no authenticator, etc.); the api.post path
      // rethrows a plain Error whose message was already shown, so only toast
      // for non-api errors to avoid a duplicate.
      if (err instanceof DOMException) {
        toast.error(getPasskeyErrorMessage(err, t, "admin.backup.reauth.passkeyError"))
      } else if (!(err instanceof Error)) {
        toast.error(t("admin.backup.reauth.passkeyError"))
      }
      setBusy(false)
    }
  }

  async function handleOIDCConfirm() {
    if (busy) {
      return
    }

    // Open the popup synchronously inside the click handler; browsers block
    // window.open() issued from an async continuation. It briefly shows about:blank
    // while the authorization URL is fetched.
    const popup = window.open("", "oidc-reauth", "width=520,height=640")
    if (!popup) {
      toast.error(t("admin.backup.reauth.oidcPopupBlocked"))
      return
    }
    const popupWin: Window = popup
    oidcPopupRef.current = popupWin
    setBusy(true)

    try {
      const { authorization_url } = await api.post<OIDCStartResponse>("/admin/reauth/oidc/start", {
        operation,
      })
      popupWin.location.href = authorization_url

      // Wait for the callback popup (same origin, on /admin) to post its outcome
      // back, or for the user to close the popup without finishing.
      await new Promise<void>((resolve, reject) => {
        function cleanup() {
          window.removeEventListener("message", onMessage)
          window.clearInterval(pollClosed)
          oidcPopupRef.current = null
        }
        function onMessage(event: MessageEvent) {
          if (event.origin !== window.location.origin) {
            return
          }
          if (event.data?.type !== "oidc-reauth") {
            return
          }
          cleanup()
          popupWin.close()
          resolve()
        }
        const pollClosed = window.setInterval(() => {
          if (popupWin.closed) {
            cleanup()
            reject(new DOMException("popup closed", "AbortError"))
          }
        }, 500)
        window.addEventListener("message", onMessage)
      })

      const { ticket } = await api.post<{ ticket: string }>("/admin/reauth/oidc/finish", {
        operation,
      })
      await verified(ticket)
    } catch (err) {
      // Backend failures already toast via api.post. A user-closed popup throws
      // an AbortError with no toast, so surface a generic message only for that.
      if (err instanceof DOMException) {
        toast.error(t("admin.backup.reauth.oidcCancelled"))
      }
      oidcPopupRef.current?.close()
      oidcPopupRef.current = null
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          onOpenChange(false)
        }
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="reauth-password">{t("admin.backup.reauth.passwordLabel")}</Label>
          <Input
            id="reauth-password"
            type="password"
            autoComplete="current-password"
            autoFocus
            value={password}
            disabled={busy}
            onChange={(event) => setPassword(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                event.preventDefault()
                void handlePasswordConfirm()
              }
            }}
          />
        </div>

        {methods.passkey && passkeySupported && (
          <div className="space-y-2">
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background px-2 text-muted-foreground">
                  {t("admin.backup.reauth.or")}
                </span>
              </div>
            </div>
            <Button
              type="button"
              variant="outline"
              className="w-full"
              disabled={busy}
              onClick={() => void handlePasskeyConfirm()}
            >
              <Fingerprint className="size-4" />
              {t("admin.backup.reauth.usePasskey")}
            </Button>
          </div>
        )}

        {methods.oidc && (
          <div className="space-y-2">
            {!(methods.passkey && passkeySupported) && (
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-background px-2 text-muted-foreground">
                    {t("admin.backup.reauth.or")}
                  </span>
                </div>
              </div>
            )}
            <Button
              type="button"
              variant="outline"
              className="w-full"
              disabled={busy}
              onClick={() => void handleOIDCConfirm()}
            >
              <KeyRound className="size-4" />
              {oidcConfig?.provider_name
                ? t("admin.backup.reauth.useOIDCNamed", { provider: oidcConfig.provider_name })
                : t("admin.backup.reauth.useOIDC")}
            </Button>
            {!methods.passkey && (
              <p className="text-xs text-muted-foreground">
                {t("admin.backup.reauth.passkeyHint")}{" "}
                <Link
                  to="/settings"
                  className="underline underline-offset-4 hover:text-foreground"
                >
                  {t("admin.backup.reauth.passkeyHintLink")}
                </Link>
              </p>
            )}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={busy}>
            {t("admin.backup.cancel")}
          </Button>
          <Button
            variant={confirmVariant}
            onClick={() => void handlePasswordConfirm()}
            disabled={busy || password.trim() === ""}
          >
            {t("admin.backup.reauth.confirm")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
