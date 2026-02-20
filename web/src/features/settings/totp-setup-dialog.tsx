import { useState, useEffect, useCallback } from "react"
import { useTranslation } from "react-i18next"
import QRCode from "react-qr-code"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { TotpConfirmResponse, TotpSetupResponse } from "@/types"

type Step = "qr" | "verify" | "backup"

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
  onEnabled: (backupCodes: string[]) => void
}

export default function TotpSetupDialog({ open, onOpenChange, onEnabled }: Props) {
  const { t } = useTranslation()
  const [step, setStep] = useState<Step>("qr")
  const [setup, setSetup] = useState<TotpSetupResponse | null>(null)
  const [code, setCode] = useState("")
  const [verifying, setVerifying] = useState(false)
  const [verifyError, setVerifyError] = useState("")
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  const [copied, setCopied] = useState(false)

  const handleClose = useCallback((val: boolean) => onOpenChange(val), [onOpenChange])

  useEffect(() => {
    if (!open) return
    setStep("qr")
    setCode("")
    setVerifyError("")
    setBackupCodes([])
    setCopied(false)
    setSetup(null)

    api.get<TotpSetupResponse>("/auth/totp/setup").then(setSetup).catch(() => {
      toast.error(t("common.requestFailed"))
      handleClose(false)
    })
  }, [open, handleClose, t])

  async function handleVerify() {
    if (!code.trim()) return
    setVerifyError("")
    setVerifying(true)
    try {
      const resp = await api.post<TotpConfirmResponse>("/auth/totp/confirm", { code: code.trim() })
      setBackupCodes(resp.backup_codes)
      setStep("backup")
    } catch (err) {
      setVerifyError(err instanceof Error ? err.message : t("settings.twoFactor.verifyError"))
    } finally {
      setVerifying(false)
    }
  }

  function handleCopyAll() {
    void navigator.clipboard.writeText(backupCodes.join("\n")).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  function handleDone() {
    onEnabled(backupCodes)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-md">
        {step === "qr" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("settings.twoFactor.setupTitle")}</DialogTitle>
              <DialogDescription>{t("settings.twoFactor.setupStep1Description")}</DialogDescription>
            </DialogHeader>
            <div className="flex flex-col items-center gap-4 py-2">
              {setup ? (
                <div className="rounded-lg border bg-white p-3">
                  <QRCode value={setup.otpauth_uri} size={180} />
                </div>
              ) : (
                <div className="h-[204px] w-[204px] rounded-lg border bg-muted animate-pulse" />
              )}
              {setup && (
                <div className="w-full space-y-1.5">
                  <p className="text-xs text-muted-foreground text-center">
                    {t("settings.twoFactor.orEnterManually")}
                  </p>
                  <div className="rounded-md bg-muted px-3 py-2 text-center font-mono text-sm tracking-widest select-all break-all">
                    {setup.secret}
                  </div>
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => handleClose(false)}>
                {t("subscription.form.cancel")}
              </Button>
              <Button onClick={() => setStep("verify")} disabled={!setup}>
                {t("settings.twoFactor.setupStep2Title")}
              </Button>
            </div>
          </>
        )}

        {step === "verify" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("settings.twoFactor.setupStep2Title")}</DialogTitle>
              <DialogDescription>{t("settings.twoFactor.setupStep2Description")}</DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              {verifyError && (
                <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {verifyError}
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="totp-confirm-code">{t("auth.login.twoFactor.codePlaceholder")}</Label>
                <Input
                  id="totp-confirm-code"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9]*"
                  maxLength={6}
                  placeholder={t("settings.twoFactor.codePlaceholder")}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  onKeyDown={(e) => { if (e.key === "Enter") void handleVerify() }}
                  autoFocus
                />
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setStep("qr")}>
                {t("auth.login.twoFactor.back")}
              </Button>
              <Button onClick={() => void handleVerify()} disabled={verifying || code.length < 6}>
                {verifying ? t("settings.twoFactor.verifying") : t("settings.twoFactor.verify")}
              </Button>
            </div>
          </>
        )}

        {step === "backup" && (
          <>
            <DialogHeader>
              <DialogTitle>{t("settings.twoFactor.backupCodesTitle")}</DialogTitle>
              <DialogDescription>{t("settings.twoFactor.backupCodesDescription")}</DialogDescription>
            </DialogHeader>
            <div className="space-y-3 py-2">
              <p className="text-xs font-medium text-destructive">
                {t("settings.twoFactor.backupCodesWarning")}
              </p>
              <div className="grid grid-cols-2 gap-2 rounded-md border bg-muted p-3">
                {backupCodes.map((bc) => (
                  <span key={bc} className="font-mono text-sm text-center tracking-wider py-0.5">
                    {bc}
                  </span>
                ))}
              </div>
              <Button variant="outline" size="sm" className="w-full" onClick={handleCopyAll}>
                {copied ? t("settings.twoFactor.copied") : t("settings.twoFactor.copyAll")}
              </Button>
            </div>
            <div className="flex justify-end">
              <Button onClick={handleDone}>
                {t("settings.twoFactor.done")}
              </Button>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
