import { useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { api, setAuth } from "@/lib/api"
import { toast } from "sonner"
import type { AuthResponse } from "@/types"

export default function RegisterPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")

    if (password !== confirmPassword) {
      setError(t("auth.register.passwordMismatch"))
      return
    }

    if (password.length < 6) {
      setError(t("auth.register.passwordTooShort"))
      return
    }

    setLoading(true)

    try {
      const data = await api.post<AuthResponse>("/auth/register", { username, email, password })
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
            <div className="space-y-2">
              <Label htmlFor="username">{t("auth.register.usernameLabel")}</Label>
              <Input
                id="username"
                type="text"
                placeholder={t("auth.register.usernamePlaceholder")}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
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
              />
            </div>
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
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t("auth.register.submitting") : t("auth.register.submit")}
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
              const idx = langs.indexOf(i18n.language as typeof langs[number])
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
