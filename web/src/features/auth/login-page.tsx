import { useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { api, setToken } from "@/lib/api"
import type { AuthResponse } from "@/types"

export default function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const data = await api.post<AuthResponse>("/auth/login", { email, password })
      setToken(data.token)
      navigate("/")
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.error"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">{t("auth.login.title")}</CardTitle>
          <CardDescription>{t("auth.login.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="email">{t("auth.login.emailLabel")}</Label>
              <Input
                id="email"
                type="email"
                placeholder={t("auth.login.emailPlaceholder")}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">{t("auth.login.passwordLabel")}</Label>
              <Input
                id="password"
                type="password"
                placeholder="••••••••"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t("auth.login.submitting") : t("auth.login.submit")}
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-muted-foreground">
            {t("auth.login.noAccount")}{" "}
            <Link to="/register" className="text-foreground underline underline-offset-4 hover:text-primary">
              {t("auth.login.signUp")}
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
