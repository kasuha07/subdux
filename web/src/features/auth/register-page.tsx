import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { api, setAuth } from "@/lib/api"
import { toast } from "sonner"
import type { AuthResponse, RegistrationConfig } from "@/types"

export default function RegisterPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [verificationCode, setVerificationCode] = useState("")
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [emailVerificationEnabled, setEmailVerificationEnabled] = useState(false)
  const [configLoading, setConfigLoading] = useState(true)
  const [codeSending, setCodeSending] = useState(false)
  const [codeCooldown, setCodeCooldown] = useState(0)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api.get<RegistrationConfig>("/auth/register/config")
      .then((config) => {
        setRegistrationEnabled(config.registration_enabled)
        setEmailVerificationEnabled(config.email_verification_enabled)
      })
      .catch(() => {
        setRegistrationEnabled(true)
        setEmailVerificationEnabled(false)
      })
      .finally(() => setConfigLoading(false))
  }, [])

  useEffect(() => {
    if (codeCooldown <= 0) {
      return
    }

    const timer = window.setInterval(() => {
      setCodeCooldown((prev) => (prev > 0 ? prev - 1 : 0))
    }, 1000)

    return () => window.clearInterval(timer)
  }, [codeCooldown])

  async function handleSendCode() {
    setError("")

    if (!email.trim()) {
      setError(t("auth.register.emailRequired"))
      return
    }

    setCodeSending(true)
    try {
      await api.post<{ message: string }>("/auth/register/send-code", { email: email.trim() })
      toast.success(t("auth.register.verificationCodeSent"))
      setCodeCooldown(60)
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.register.error"))
    } finally {
      setCodeSending(false)
    }
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")

    if (!registrationEnabled) {
      setError(t("auth.register.registrationDisabled"))
      return
    }

    if (password !== confirmPassword) {
      setError(t("auth.register.passwordMismatch"))
      return
    }

    if (password.length < 6) {
      setError(t("auth.register.passwordTooShort"))
      return
    }

    if (emailVerificationEnabled && !verificationCode.trim()) {
      setError(t("auth.register.verificationCodeRequired"))
      return
    }

    setLoading(true)

    try {
      const data = await api.post<AuthResponse>("/auth/register", {
        username: username.trim(),
        email: email.trim(),
        password,
        verification_code: emailVerificationEnabled ? verificationCode.trim() : "",
      })
      setAuth(data.token, data.user)
      toast.success(t("auth.register.success"))
      navigate("/")
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.register.error"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">{t("auth.register.title")}</CardTitle>
          <CardDescription>{t("auth.register.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}

            {!registrationEnabled && !configLoading && (
              <div className="rounded-md bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-300">
                {t("auth.register.registrationDisabledDescription")}
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="username">{t("auth.register.usernameLabel")}</Label>
              <Input
                id="username"
                type="text"
                placeholder={t("auth.register.usernamePlaceholder")}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                disabled={configLoading || !registrationEnabled}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="email">{t("auth.register.emailLabel")}</Label>
              <Input
                id="email"
                type="email"
                placeholder={t("auth.register.emailPlaceholder")}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
                disabled={configLoading || !registrationEnabled}
              />
            </div>

            {emailVerificationEnabled && (
              <div className="space-y-2">
                <Label htmlFor="verification-code">{t("auth.register.verificationCodeLabel")}</Label>
                <div className="flex gap-2">
                  <Input
                    id="verification-code"
                    type="text"
                    inputMode="numeric"
                    maxLength={6}
                    placeholder={t("auth.register.verificationCodePlaceholder")}
                    value={verificationCode}
                    onChange={(e) => setVerificationCode(e.target.value)}
                    required
                    disabled={configLoading || !registrationEnabled}
                  />
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => void handleSendCode()}
                    disabled={
                      codeSending ||
                      codeCooldown > 0 ||
                      configLoading ||
                      !registrationEnabled ||
                      email.trim() === ""
                    }
                  >
                    {codeSending
                      ? t("auth.register.sendingVerificationCode")
                      : codeCooldown > 0
                        ? t("auth.register.resendIn", { seconds: codeCooldown })
                        : t("auth.register.sendVerificationCode")}
                  </Button>
                </div>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="password">{t("auth.register.passwordLabel")}</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={6}
                disabled={configLoading || !registrationEnabled}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="confirm-password">{t("auth.register.confirmPasswordLabel")}</Label>
              <Input
                id="confirm-password"
                type="password"
                placeholder="••••••••"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                minLength={6}
                disabled={configLoading || !registrationEnabled}
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading || configLoading || !registrationEnabled}>
              {loading
                ? t("auth.register.submitting")
                : configLoading
                  ? t("common.loading")
                  : t("auth.register.submit")}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-muted-foreground">
            {t("auth.register.hasAccount")}{" "}
            <Link to="/login" className="text-foreground underline underline-offset-4 hover:text-primary">
              {t("auth.register.signIn")}
            </Link>
          </p>
          <button
            type="button"
            className="mt-2 block mx-auto text-xs text-muted-foreground hover:text-foreground"
            onClick={() => {
              const langs = ["en", "zh-CN", "ja"] as const
              const idx = langs.indexOf(i18n.language as (typeof langs)[number])
              i18n.changeLanguage(langs[(idx + 1) % langs.length])
            }}
          >
            {{ en: "中文", "zh-CN": "日本語", ja: "EN" }[i18n.language] ?? "中文"}
          </button>
        </CardContent>
      </Card>
    </div>
  )
}
