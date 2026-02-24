import { useState, useRef, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
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

interface PreviewCurrencyChange {
  code: string
  symbol: string
  is_new: boolean
}

interface PreviewPaymentMethodChange {
  name: string
  is_new: boolean
  matched?: string
}

interface PreviewCategoryChange {
  name: string
  is_new: boolean
}

interface PreviewSubscriptionChange {
  name: string
  amount: number
  currency: string
  billing_type: string
  category?: string
  skipped: boolean
  skip_reason?: string
}

interface ImportPreview {
  currencies: PreviewCurrencyChange[]
  payment_methods: PreviewPaymentMethodChange[]
  categories: PreviewCategoryChange[]
  subscriptions: PreviewSubscriptionChange[]
}

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
  const [importPreviewOpen, setImportPreviewOpen] = useState(false)
  const [importPreview, setImportPreview] = useState<ImportPreview | null>(null)
  const [importRawData, setImportRawData] = useState<unknown[] | null>(null)

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
        body: JSON.stringify({ data, confirm: false }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.importFailed"))
        return
      }

      const preview: ImportPreview = (await res.json()).preview
      setImportPreview(preview)
      setImportRawData(data)
      setImportPreviewOpen(true)
    } catch {
      toast.error(t("settings.account.importFailed"))
    } finally {
      setImportLoading(false)
      if (importFileRef.current) {
        importFileRef.current.value = ""
      }
    }
  }

  async function handleConfirmImport() {
    if (!importRawData) return

    setImportLoading(true)
    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/import/wallos", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ data: importRawData, confirm: true }),
      })

      if (!res.ok) {
        const err = await res.json()
        toast.error(err.error || t("settings.account.importFailed"))
        return
      }

      const { result } = await res.json()
      toast.success(
        t("settings.account.importSuccess", {
          imported: result.imported,
          skipped: result.skipped,
        })
      )
      setImportPreviewOpen(false)
      setImportPreview(null)
      setImportRawData(null)
    } catch {
      toast.error(t("settings.account.importFailed"))
    } finally {
      setImportLoading(false)
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
              ? t("settings.account.importAnalyzing")
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

      <Dialog open={importPreviewOpen} onOpenChange={(open) => {
        if (!open && !importLoading) {
          setImportPreviewOpen(false)
          setImportPreview(null)
          setImportRawData(null)
        }
      }}>
        <DialogContent
          className="flex max-h-[calc(100vh-1.5rem)] flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh] sm:max-w-2xl"
          onInteractOutside={(e) => e.preventDefault()}
          onEscapeKeyDown={(e) => { if (importLoading) e.preventDefault() }}
          showCloseButton={false}
        >
          <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
            <DialogTitle>{t("settings.account.importPreviewTitle")}</DialogTitle>
            <DialogDescription>{t("settings.account.importPreviewDescription")}</DialogDescription>
          </DialogHeader>

          <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4 sm:px-6">
            {importPreview && (
              <div className="space-y-5">
                {importPreview.currencies.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewCurrencies")}</h4>
                    <div className="space-y-1.5">
                      {importPreview.currencies.map((c) => (
                        <div key={c.code} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                          <span>{c.code}{c.symbol ? ` (${c.symbol})` : ""}</span>
                          <Badge variant={c.is_new ? "default" : "secondary"} className="text-xs">
                            {c.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                          </Badge>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {importPreview.payment_methods.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewPaymentMethods")}</h4>
                    <div className="space-y-1.5">
                      {importPreview.payment_methods.map((pm) => (
                        <div key={pm.name} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                          <div>
                            <span>{pm.name}</span>
                            {pm.matched && (
                              <span className="ml-2 text-xs text-muted-foreground">
                                {t("settings.account.importPreviewMatchedAs", { name: pm.matched })}
                              </span>
                            )}
                          </div>
                          <Badge variant={pm.is_new ? "default" : "secondary"} className="text-xs">
                            {pm.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                          </Badge>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {importPreview.categories.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewCategories")}</h4>
                    <div className="space-y-1.5">
                      {importPreview.categories.map((cat) => (
                        <div key={cat.name} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                          <span>{cat.name}</span>
                          <Badge variant={cat.is_new ? "default" : "secondary"} className="text-xs">
                            {cat.is_new ? t("settings.account.importPreviewNew") : t("settings.account.importPreviewExists")}
                          </Badge>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {importPreview.subscriptions.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium">{t("settings.account.importPreviewSubscriptions")}</h4>
                    <div className="space-y-1.5">
                      {importPreview.subscriptions.map((sub, i) => (
                        <div key={i} className="flex items-center justify-between rounded-md border px-3 py-2 text-sm">
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="truncate font-medium">{sub.name}</span>
                              <span className="shrink-0 text-xs text-muted-foreground">
                                {sub.amount} {sub.currency}
                              </span>
                            </div>
                            {sub.category && (
                              <span className="text-xs text-muted-foreground">{sub.category}</span>
                            )}
                          </div>
                          <Badge
                            variant={sub.skipped ? "outline" : "default"}
                            className="ml-2 shrink-0 text-xs"
                          >
                            {sub.skipped ? t("settings.account.importPreviewSkipped") : t("settings.account.importPreviewNew")}
                          </Badge>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {importPreview.currencies.length === 0 &&
                  importPreview.payment_methods.length === 0 &&
                  importPreview.categories.length === 0 &&
                  importPreview.subscriptions.length === 0 && (
                    <p className="py-8 text-center text-sm text-muted-foreground">
                      {t("settings.account.importPreviewEmpty")}
                    </p>
                  )}
              </div>
            )}
          </div>

          <div className="sticky bottom-0 z-10 flex justify-end gap-2 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
            <Button
              variant="outline"
              size="sm"
              disabled={importLoading}
              onClick={() => {
                setImportPreviewOpen(false)
                setImportPreview(null)
                setImportRawData(null)
              }}
            >
              {t("settings.account.importPreviewCancel")}
            </Button>
            <Button
              size="sm"
              disabled={importLoading || (importPreview?.subscriptions.every(s => s.skipped) ?? false)}
              onClick={() => void handleConfirmImport()}
            >
              {importLoading
                ? t("settings.account.importing")
                : t("settings.account.importPreviewConfirm")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </TabsContent>
  )
}
