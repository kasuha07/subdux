import { useState, type FormEvent } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { ResetPasswordInput } from "@/types"

export default function ResetPasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [email, setEmail] = useState(searchParams.get("email") ?? "")
  const [verificationCode, setVerificationCode] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")

    if (newPassword !== confirmPassword) {
      setError(t("auth.resetPassword.passwordMismatch"))
      return
    }
    if (newPassword.length < 6) {
      setError(t("auth.resetPassword.passwordTooShort"))
      return
    }

    setLoading(true)
    try {
      const payload: ResetPasswordInput = {
        email: email.trim(),
        verification_code: verificationCode.trim(),
        new_password: newPassword,
      }
      await api.post<{ message: string }>("/auth/password/reset", payload)
      toast.success(t("auth.resetPassword.success"))
      navigate("/login")
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.resetPassword.error"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">
            {t("auth.resetPassword.title")}
          </CardTitle>
          <CardDescription>{t("auth.resetPassword.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={(event) => void handleSubmit(event)} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="reset-email">{t("auth.resetPassword.emailLabel")}</Label>
              <Input
                id="reset-email"
                type="email"
                placeholder={t("auth.resetPassword.emailPlaceholder")}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="reset-code">{t("auth.resetPassword.codeLabel")}</Label>
              <Input
                id="reset-code"
                type="text"
                inputMode="numeric"
                placeholder={t("auth.resetPassword.codePlaceholder")}
                value={verificationCode}
                onChange={(event) => setVerificationCode(event.target.value)}
                required
                maxLength={6}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="reset-password">{t("auth.resetPassword.newPasswordLabel")}</Label>
              <Input
                id="reset-password"
                type="password"
                placeholder="••••••••"
                value={newPassword}
                onChange={(event) => setNewPassword(event.target.value)}
                required
                minLength={6}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="reset-confirm-password">{t("auth.resetPassword.confirmPasswordLabel")}</Label>
              <Input
                id="reset-confirm-password"
                type="password"
                placeholder="••••••••"
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                required
                minLength={6}
              />
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t("auth.resetPassword.submitting") : t("auth.resetPassword.submit")}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            <Link to="/login" className="text-foreground underline underline-offset-4 hover:text-primary">
              {t("auth.resetPassword.backToLogin")}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
