import { useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { ForgotPasswordInput } from "@/types"

export default function ForgotPasswordPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [email, setEmail] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")
    setLoading(true)

    try {
      const payload: ForgotPasswordInput = { email: email.trim() }
      await api.post<{ message: string }>("/auth/password/forgot", payload)
      toast.success(t("auth.forgotPassword.success"))
      navigate(`/reset-password?email=${encodeURIComponent(email.trim())}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.forgotPassword.error"))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">
            {t("auth.forgotPassword.title")}
          </CardTitle>
          <CardDescription>{t("auth.forgotPassword.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={(event) => void handleSubmit(event)} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="forgot-email">{t("auth.forgotPassword.emailLabel")}</Label>
              <Input
                id="forgot-email"
                type="email"
                placeholder={t("auth.forgotPassword.emailPlaceholder")}
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                required
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? t("auth.forgotPassword.submitting") : t("auth.forgotPassword.submit")}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            <Link to="/login" className="text-foreground underline underline-offset-4 hover:text-primary">
              {t("auth.forgotPassword.backToLogin")}
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
