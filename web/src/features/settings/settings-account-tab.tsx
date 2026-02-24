import { useState, useRef, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { TabsContent } from "@/components/ui/tabs"
import type { User } from "@/types"

import OIDCSection from "./oidc-section"
import PasskeySection from "./passkey-section"
import TotpSection from "./totp-section"

interface SettingsAccountTabProps {
  confirmPassword: string
  currentPassword: string
  emailChangeError: string
  emailChangeLoading: boolean
  emailChangePassword: string
  emailCodeLoading: boolean
  emailCodeSent: boolean
  emailVerificationCode: string
  newEmail: string
  newPassword: string
  onConfirmEmailChange: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onChangePassword: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onEmailChangePasswordChange: (value: string) => void
  onEmailVerificationCodeChange: (value: string) => void
  onConfirmPasswordChange: (value: string) => void
  onCurrentPasswordChange: (value: string) => void
  onLogout: () => void
  onNewEmailChange: (value: string) => void
  onNewPasswordChange: (value: string) => void
  onSendEmailChangeCode: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onUserChange: (user: User) => void
  passwordError: string
  passwordLoading: boolean
  passwordSuccess: string
  user: User | null
}

export default function SettingsAccountTab({
  confirmPassword,
  currentPassword,
  emailChangeError,
  emailChangeLoading,
  emailChangePassword,
  emailCodeLoading,
  emailCodeSent,
  emailVerificationCode,
  newEmail,
  newPassword,
  onConfirmEmailChange,
  onChangePassword,
  onEmailChangePasswordChange,
  onEmailVerificationCodeChange,
  onConfirmPasswordChange,
  onCurrentPasswordChange,
  onLogout,
  onNewEmailChange,
  onNewPasswordChange,
  onSendEmailChangeCode,
  onUserChange,
  passwordError,
  passwordLoading,
  passwordSuccess,
  user,
}: SettingsAccountTabProps) {
  const { t } = useTranslation()
  const [emailDialogOpen, setEmailDialogOpen] = useState(false)
  const [exportLoading, setExportLoading] = useState(false)
  const [importLoading, setImportLoading] = useState(false)
  const importFileRef = useRef<HTMLInputElement>(null)

  async function handleExport() {
    setExportLoading(true)
    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/export", {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      if (!res.ok) throw new Error("Export failed")
      const blob = await res.blob()
      const disposition = res.headers.get("Content-Disposition")
      let filename = "subdux-export.json"
      if (disposition) {
        const match = disposition.match(/filename="?([^"]+)"?/)
        if (match) filename = match[1]
      }
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      URL.revokeObjectURL(url)
      a.remove()
    } catch {
      // error toast is handled by the fetch failure
    } finally {
      setExportLoading(false)
    }
  }

  async function handleImportWallos(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) return

    setImportLoading(true)
    try {
      const text = await file.text()
      const data = JSON.parse(text)

      if (!Array.isArray(data)) {
        toast.error(t("settings.account.importInvalidFormat"))
        return
      }

      const token = localStorage.getItem("token")
      const res = await fetch("/api/import/wallos", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify(data),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.importFailed"))
        return
      }

      const result = await res.json()
      toast.success(
        t("settings.account.importSuccess", {
          imported: result.imported,
          skipped: result.skipped,
        })
      )
    } catch {
      toast.error(t("settings.account.importFailed"))
    } finally {
      setImportLoading(false)
      if (importFileRef.current) {
        importFileRef.current.value = ""
      }
    }
  }

  return (
    <TabsContent value="account">
      <div className="space-y-4">
        <div>
          <h2 className="text-base font-semibold tracking-tight">{t("settings.account.title")}</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("settings.account.description")}
          </p>
        </div>

        <div>
          <Label className="text-xs text-muted-foreground">
            {t("settings.account.username")}
          </Label>
          <p className="mt-0.5 text-sm">{user?.username ?? "—"}</p>
        </div>

        <div>
          <Label className="text-xs text-muted-foreground">
            {t("settings.account.email")}
          </Label>
          <div className="mt-0.5 flex items-center gap-2">
            <p className="text-sm">{user?.email ?? "—"}</p>
            <Dialog open={emailDialogOpen} onOpenChange={setEmailDialogOpen}>
              <DialogTrigger asChild>
                <Button size="xs" variant="outline">
                  {t("settings.account.changeEmail")}
                </Button>
              </DialogTrigger>
              <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-md flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
                <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
                  <DialogTitle>{t("settings.account.changeEmail")}</DialogTitle>
                </DialogHeader>
                <div className="flex min-h-0 flex-1 flex-col">
                  <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4 sm:px-6">
                    <form
                      id="send-email-change-code-form"
                      onSubmit={(event) => void onSendEmailChangeCode(event)}
                      className="grid gap-3"
                    >
                      {emailChangeError && (
                        <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                          {emailChangeError}
                        </div>
                      )}
                      {emailCodeSent && (
                        <div className="rounded-md bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
                          {t("settings.account.emailCodeSent")}
                        </div>
                      )}
                      <div className="space-y-2">
                        <Label htmlFor="new-email">{t("settings.account.newEmail")}</Label>
                        <Input
                          id="new-email"
                          type="email"
                          placeholder={t("auth.register.emailPlaceholder")}
                          value={newEmail}
                          onChange={(event) => onNewEmailChange(event.target.value)}
                          required
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="email-change-password">{t("settings.account.currentPassword")}</Label>
                        <Input
                          id="email-change-password"
                          type="password"
                          placeholder="••••••••"
                          value={emailChangePassword}
                          onChange={(event) => onEmailChangePasswordChange(event.target.value)}
                          required
                        />
                      </div>
                    </form>

                    <form
                      id="confirm-email-change-form"
                      onSubmit={(event) => void onConfirmEmailChange(event)}
                      className="grid gap-3"
                    >
                      <div className="space-y-2">
                        <Label htmlFor="email-verification-code">{t("settings.account.emailVerificationCode")}</Label>
                        <div className="flex gap-2">
                          <Input
                            id="email-verification-code"
                            type="text"
                            inputMode="numeric"
                            maxLength={6}
                            placeholder={t("auth.register.verificationCodePlaceholder")}
                            value={emailVerificationCode}
                            onChange={(event) => onEmailVerificationCodeChange(event.target.value)}
                            required
                          />
                          <Button
                            size="sm"
                            type="submit"
                            form="send-email-change-code-form"
                            variant="outline"
                            className="shrink-0"
                            disabled={emailCodeLoading}
                          >
                            {emailCodeLoading
                              ? t("settings.account.sendingEmailCode")
                              : t("settings.account.sendEmailCode")}
                          </Button>
                        </div>
                      </div>
                    </form>
                  </div>

                  <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
                    <Button
                      size="sm"
                      type="submit"
                      form="confirm-email-change-form"
                      disabled={emailChangeLoading}
                    >
                      {emailChangeLoading
                        ? t("settings.account.confirmingEmailChange")
                        : t("settings.account.confirmEmailChange")}
                    </Button>
                  </div>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        <Separator />

        <TotpSection user={user} onUserChange={onUserChange} />

        <Separator />

        <PasskeySection />

        <Separator />

        <OIDCSection />

        <Separator />

        <div>
          <h3 className="text-base font-semibold tracking-tight">{t("settings.account.exportTitle")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("settings.account.exportDescription")}
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            disabled={exportLoading}
            onClick={handleExport}
          >
            {exportLoading
              ? t("settings.account.exporting")
              : t("settings.account.exportButton")}
          </Button>
        </div>

        <div className="mt-3">
          <h3 className="text-base font-semibold tracking-tight">{t("settings.account.importTitle")}</h3>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("settings.account.importDescription")}
          </p>
          <input
            ref={importFileRef}
            type="file"
            accept=".json"
            className="hidden"
            onChange={handleImportWallos}
          />
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            disabled={importLoading}
            onClick={() => importFileRef.current?.click()}
          >
            {importLoading
              ? t("settings.account.importing")
              : t("settings.account.importButton")}
          </Button>
        </div>

        <Separator />

        <div>
          <h3 className="text-base font-semibold tracking-tight">{t("settings.account.changePassword")}</h3>
          <form onSubmit={(event) => void onChangePassword(event)} className="mt-3 grid max-w-sm gap-3">
            {passwordError && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {passwordError}
              </div>
            )}
            {passwordSuccess && (
              <div className="rounded-md bg-emerald-500/10 px-3 py-2 text-sm text-emerald-700">
                {passwordSuccess}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="current-password">{t("settings.account.currentPassword")}</Label>
              <Input
                id="current-password"
                type="password"
                placeholder="••••••••"
                value={currentPassword}
                onChange={(event) => onCurrentPasswordChange(event.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-password">{t("settings.account.newPassword")}</Label>
              <Input
                id="new-password"
                type="password"
                placeholder="••••••••"
                value={newPassword}
                onChange={(event) => onNewPasswordChange(event.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-password">{t("settings.account.confirmPassword")}</Label>
              <Input
                id="confirm-password"
                type="password"
                placeholder="••••••••"
                value={confirmPassword}
                onChange={(event) => onConfirmPasswordChange(event.target.value)}
                required
              />
            </div>
            <div>
              <Button size="sm" type="submit" disabled={passwordLoading}>
                {passwordLoading
                  ? t("settings.account.updating")
                  : t("settings.account.update")}
              </Button>
            </div>
          </form>
        </div>

        <Separator />

        <div>
          <p className="text-sm text-muted-foreground">{t("settings.account.logoutDescription")}</p>
          <Button variant="outline" size="sm" className="mt-2" onClick={onLogout}>
            {t("settings.account.logout")}
          </Button>
        </div>
      </div>
    </TabsContent>
  )
}
