import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"
import { CheckCircle2, XCircle } from "lucide-react"

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
import { useSettingsAccountTransfer } from "@/features/settings/hooks/use-settings-account-transfer"
import { cn } from "@/lib/utils"
import type { User } from "@/types"

import OIDCSection from "./oidc-section"
import PasskeySection from "./passkey-section"
import {
  ImportPreviewDialog,
  SettingsAccountTransferSection,
  SubduxImportPreviewDialog,
} from "./settings-account-transfer-section"
import TotpSection from "./totp-section"

type PasswordStrengthLevel = "weak" | "fair" | "good" | "strong"

interface PasswordStrength {
  level: PasswordStrengthLevel
  score: number
}

const passwordStrengthStyles: Record<PasswordStrengthLevel, string> = {
  weak: "bg-destructive",
  fair: "bg-amber-500",
  good: "bg-sky-500",
  strong: "bg-emerald-500",
}

function getPasswordStrength(password: string): PasswordStrength {
  const hasLower = /[a-z]/.test(password)
  const hasUpper = /[A-Z]/.test(password)
  const hasNumber = /\d/.test(password)
  const hasSymbol = /[^A-Za-z0-9]/.test(password)
  const lengthScore = password.length >= 12 ? 2 : password.length >= 8 ? 1 : 0
  const varietyScore = [
    hasLower && hasUpper,
    hasNumber,
    hasSymbol,
  ].filter(Boolean).length
  const score = Math.min(4, lengthScore + varietyScore)

  if (score >= 4) {
    return { level: "strong", score: 4 }
  }
  if (score === 3) {
    return { level: "good", score: 3 }
  }
  if (score === 2) {
    return { level: "fair", score: 2 }
  }
  return { level: "weak", score: 1 }
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
  onLogout: () => void | Promise<void>
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
  const transfer = useSettingsAccountTransfer()

  return (
    <TabsContent value="account">
      <div className="space-y-4">
        <div>
          <h2 className="text-base font-semibold tracking-tight select-none">{t("settings.account.title")}</h2>
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
            <EmailChangeDialog
              confirmLoading={emailChangeLoading}
              emailChangeError={emailChangeError}
              emailChangePassword={emailChangePassword}
              emailCodeLoading={emailCodeLoading}
              emailCodeSent={emailCodeSent}
              emailVerificationCode={emailVerificationCode}
              newEmail={newEmail}
              onConfirmEmailChange={onConfirmEmailChange}
              onEmailChangePasswordChange={onEmailChangePasswordChange}
              onEmailVerificationCodeChange={onEmailVerificationCodeChange}
              onNewEmailChange={onNewEmailChange}
              onOpenChange={setEmailDialogOpen}
              onSendEmailChangeCode={onSendEmailChangeCode}
              open={emailDialogOpen}
            />
          </div>
        </div>

        <Separator />

        <TotpSection user={user} onUserChange={onUserChange} />

        <Separator />

        <PasskeySection />

        <Separator />

        <OIDCSection />

        <Separator />

        <SettingsAccountTransferSection
          exportLoading={transfer.exportLoading}
          exportSecretsConfirmOpen={transfer.exportSecretsConfirmOpen}
          importFileRef={transfer.importFileRef}
          importLoading={transfer.importLoading}
          onExport={transfer.handleExport}
          onExportSecretsConfirmOpenChange={transfer.setExportSecretsConfirmOpen}
          onImportSubdux={transfer.handleImportSubdux}
          onImportWallos={transfer.handleImportWallos}
          subduxImportFileRef={transfer.subduxImportFileRef}
          subduxImportLoading={transfer.subduxImportLoading}
        />

        <Separator />

        <PasswordSection
          confirmPassword={confirmPassword}
          currentPassword={currentPassword}
          newPassword={newPassword}
          onChangePassword={onChangePassword}
          onConfirmPasswordChange={onConfirmPasswordChange}
          onCurrentPasswordChange={onCurrentPasswordChange}
          onNewPasswordChange={onNewPasswordChange}
          passwordError={passwordError}
          passwordLoading={passwordLoading}
          passwordSuccess={passwordSuccess}
        />

        <Separator />

        <div>
          <p className="text-sm text-muted-foreground">{t("settings.account.logoutDescription")}</p>
          <Button variant="outline" size="sm" className="mt-2" onClick={() => void onLogout()}>
            {t("settings.account.logout")}
          </Button>
        </div>
      </div>

      <SubduxImportPreviewDialog
        loading={transfer.subduxImportLoading}
        onConfirm={transfer.handleConfirmSubduxImport}
        onOpenChange={transfer.setSubduxImportPreviewOpen}
        onReset={transfer.resetSubduxImportPreview}
        open={transfer.subduxImportPreviewOpen}
        preview={transfer.subduxImportPreview}
      />

      <ImportPreviewDialog
        loading={transfer.importLoading}
        onConfirm={transfer.handleConfirmImport}
        onOpenChange={transfer.setImportPreviewOpen}
        onReset={transfer.resetImportPreview}
        open={transfer.importPreviewOpen}
        preview={transfer.importPreview}
      />
    </TabsContent>
  )
}

function EmailChangeDialog({
  confirmLoading,
  emailChangeError,
  emailChangePassword,
  emailCodeLoading,
  emailCodeSent,
  emailVerificationCode,
  newEmail,
  onConfirmEmailChange,
  onEmailChangePasswordChange,
  onEmailVerificationCodeChange,
  onNewEmailChange,
  onOpenChange,
  onSendEmailChangeCode,
  open,
}: {
  confirmLoading: boolean
  emailChangeError: string
  emailChangePassword: string
  emailCodeLoading: boolean
  emailCodeSent: boolean
  emailVerificationCode: string
  newEmail: string
  onConfirmEmailChange: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onEmailChangePasswordChange: (value: string) => void
  onEmailVerificationCodeChange: (value: string) => void
  onNewEmailChange: (value: string) => void
  onOpenChange: (open: boolean) => void
  onSendEmailChangeCode: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  open: boolean
}) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        <Button size="xs" variant="outline">
          {t("settings.account.changeEmail")}
        </Button>
      </DialogTrigger>
      <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-md flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>{t("settings.account.changeEmail")}</DialogTitle>
          <DialogDescription className="sr-only">
            {t("settings.account.changeEmailDescription")}
          </DialogDescription>
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
                  autoComplete="email"
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
                  autoComplete="current-password"
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
                    autoComplete="one-time-code"
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
              disabled={confirmLoading}
            >
              {confirmLoading
                ? t("settings.account.confirmingEmailChange")
                : t("settings.account.confirmEmailChange")}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function PasswordSection({
  confirmPassword,
  currentPassword,
  newPassword,
  onChangePassword,
  onConfirmPasswordChange,
  onCurrentPasswordChange,
  onNewPasswordChange,
  passwordError,
  passwordLoading,
  passwordSuccess,
}: {
  confirmPassword: string
  currentPassword: string
  newPassword: string
  onChangePassword: (event: FormEvent<HTMLFormElement>) => void | Promise<void>
  onConfirmPasswordChange: (value: string) => void
  onCurrentPasswordChange: (value: string) => void
  onNewPasswordChange: (value: string) => void
  passwordError: string
  passwordLoading: boolean
  passwordSuccess: string
}) {
  const { t } = useTranslation()

  return (
    <div>
      <h3 className="text-base font-semibold tracking-tight select-none">{t("settings.account.changePassword")}</h3>
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
            autoComplete="current-password"
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
            autoComplete="new-password"
            placeholder="••••••••"
            value={newPassword}
            onChange={(event) => onNewPasswordChange(event.target.value)}
            required
          />
          <PasswordStrengthIndicator password={newPassword} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="confirm-password">{t("settings.account.confirmPassword")}</Label>
          <Input
            id="confirm-password"
            type="password"
            autoComplete="new-password"
            placeholder="••••••••"
            value={confirmPassword}
            onChange={(event) => onConfirmPasswordChange(event.target.value)}
            required
          />
          <PasswordMatchIndicator
            confirmPassword={confirmPassword}
            newPassword={newPassword}
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
  )
}

function PasswordStrengthIndicator({ password }: { password: string }) {
  const { t } = useTranslation()

  if (!password) {
    return null
  }

  const strength = getPasswordStrength(password)

  return (
    <div className="space-y-1" aria-live="polite">
      <div className="flex items-center justify-between gap-3 text-xs">
        <span className="text-muted-foreground">
          {t("settings.account.passwordStrength")}
        </span>
        <span className={cn(
          "font-medium",
          strength.level === "weak" && "text-destructive",
          strength.level === "fair" && "text-amber-700 dark:text-amber-400",
          strength.level === "good" && "text-sky-700 dark:text-sky-400",
          strength.level === "strong" && "text-emerald-700 dark:text-emerald-400"
        )}>
          {t(`settings.account.passwordStrengthLevels.${strength.level}`)}
        </span>
      </div>
      <div
        className="grid h-1.5 grid-cols-4 gap-1"
        role="meter"
        aria-label={t("settings.account.passwordStrength")}
        aria-valuemin={0}
        aria-valuemax={4}
        aria-valuenow={strength.score}
        aria-valuetext={t(`settings.account.passwordStrengthLevels.${strength.level}`)}
      >
        {Array.from({ length: 4 }, (_, index) => (
          <span
            key={index}
            className={cn(
              "rounded-full bg-muted transition-colors",
              index < strength.score && passwordStrengthStyles[strength.level]
            )}
          />
        ))}
      </div>
    </div>
  )
}

function PasswordMatchIndicator({
  confirmPassword,
  newPassword,
}: {
  confirmPassword: string
  newPassword: string
}) {
  const { t } = useTranslation()

  if (!confirmPassword) {
    return null
  }

  const matches = newPassword === confirmPassword

  return (
    <div
      className={cn(
        "flex items-center gap-1.5 text-xs",
        matches
          ? "text-emerald-700 dark:text-emerald-400"
          : "text-destructive"
      )}
      aria-live="polite"
    >
      {matches ? (
        <CheckCircle2 className="size-3.5 shrink-0" aria-hidden="true" />
      ) : (
        <XCircle className="size-3.5 shrink-0" aria-hidden="true" />
      )}
      <span>
        {matches
          ? t("settings.account.passwordsMatch")
          : t("settings.account.passwordsDoNotMatch")}
      </span>
    </div>
  )
}
