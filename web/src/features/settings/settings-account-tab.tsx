import { type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
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

  return (
    <TabsContent value="account">
      <div className="space-y-4">
        <div>
          <h2 className="text-sm font-medium">{t("settings.account.title")}</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {t("settings.account.description")}
          </p>
        </div>

        <div>
          <Label className="text-xs text-muted-foreground">
            {t("settings.account.email")}
          </Label>
          <p className="mt-0.5 text-sm">{user?.email ?? "—"}</p>
        </div>

        <Separator />

        <div>
          <h3 className="text-sm font-medium">{t("settings.account.changeEmail")}</h3>
          <form onSubmit={(event) => void onSendEmailChangeCode(event)} className="mt-3 grid max-w-sm gap-3">
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
            <div>
              <Button size="sm" type="submit" disabled={emailCodeLoading}>
                {emailCodeLoading
                  ? t("settings.account.sendingEmailCode")
                  : t("settings.account.sendEmailCode")}
              </Button>
            </div>
          </form>

          <form onSubmit={(event) => void onConfirmEmailChange(event)} className="mt-3 grid max-w-sm gap-3">
            <div className="space-y-2">
              <Label htmlFor="email-verification-code">{t("settings.account.emailVerificationCode")}</Label>
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
            </div>
            <div>
              <Button size="sm" type="submit" disabled={emailChangeLoading}>
                {emailChangeLoading
                  ? t("settings.account.confirmingEmailChange")
                  : t("settings.account.confirmEmailChange")}
              </Button>
            </div>
          </form>
        </div>

        <Separator />

        <TotpSection user={user} onUserChange={onUserChange} />

        <Separator />

        <PasskeySection />

        <Separator />

        <OIDCSection />

        <Separator />

        <div>
          <h3 className="text-sm font-medium">{t("settings.account.changePassword")}</h3>
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
