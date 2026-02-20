import { useEffect, useState, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { api, setAuth } from "@/lib/api"
import { getPasskeyCredential, isPasskeySupported, type CredentialAssertionJSON } from "@/lib/passkey"
import { getPasskeyErrorMessage } from "@/lib/passkey-error"
import { toast } from "sonner"
import type {
  AuthResponse,
  LoginResponse,
  OIDCConfig,
  OIDCSessionResult,
  OIDCStartResponse,
  PasskeyBeginResult,
} from "@/types"

type LoginStep = "credentials" | "totp"

export default function LoginPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const [identifier, setIdentifier] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)
  const [passkeyLoading, setPasskeyLoading] = useState(false)
  const [oidcLoading, setOidcLoading] = useState(false)
  const [oidcSubmitting, setOidcSubmitting] = useState(false)
  const [createDevAccountLoading, setCreateDevAccountLoading] = useState(false)
  const [oidcConfig, setOidcConfig] = useState<OIDCConfig | null>(null)

  const [step, setStep] = useState<LoginStep>("credentials")
  const [totpToken, setTotpToken] = useState("")
  const [totpCode, setTotpCode] = useState("")
  const passkeySupported = isPasskeySupported()
  const isDevMode = import.meta.env.DEV

  useEffect(() => {
    api.get<OIDCConfig>("/auth/oidc/config")
      .then(setOidcConfig)
      .catch(() => void 0)
  }, [])

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const action = params.get("oidc_action")
    const sessionID = params.get("oidc_session")
    if (action !== "login" || !sessionID) {
      return
    }

    setOidcLoading(true)
    api.get<OIDCSessionResult>(`/auth/oidc/session/${encodeURIComponent(sessionID)}`)
      .then((result) => {
        if (result.error) {
          setError(result.error)
          return
        }

        if (result.token && result.user) {
          setAuth(result.token, result.user)
          toast.success(t("auth.login.success"))
          navigate("/", { replace: true })
          return
        }

        setError(t("auth.login.oidcError"))
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : t("auth.login.oidcError"))
      })
      .finally(() => {
        const nextURL = new URL(window.location.href)
        nextURL.searchParams.delete("oidc_action")
        nextURL.searchParams.delete("oidc_session")
        const query = nextURL.searchParams.toString()
        const search = query ? `?${query}` : ""
        window.history.replaceState({}, "", `${nextURL.pathname}${search}${nextURL.hash}`)
        setOidcLoading(false)
      })
  }, [navigate, t])

  async function handleCredentialsSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const data = await api.post<LoginResponse>("/auth/login", { identifier, password })
      if ("requires_totp" in data && data.requires_totp) {
        setTotpToken(data.totp_token)
        setStep("totp")
      } else {
        const authData = data as AuthResponse
        setAuth(authData.token, authData.user)
        toast.success(t("auth.login.success"))
        navigate("/")
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.error"))
    } finally {
      setLoading(false)
    }
  }

  async function handleTotpSubmit(e: FormEvent) {
    e.preventDefault()
    setError("")
    setLoading(true)

    try {
      const data = await api.post<AuthResponse>("/auth/totp/verify-login", {
        totp_token: totpToken,
        code: totpCode.trim(),
      })
      setAuth(data.token, data.user)
      toast.success(t("auth.login.success"))
      navigate("/")
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.twoFactor.error"))
    } finally {
      setLoading(false)
    }
  }

  function handleBack() {
    setStep("credentials")
    setTotpToken("")
    setTotpCode("")
    setError("")
  }

  async function handlePasskeyLogin() {
    setError("")
    if (!passkeySupported) {
      setError(t("auth.login.passkeyUnsupported"))
      return
    }

    setPasskeyLoading(true)
    try {
      const begin = await api.post<PasskeyBeginResult<CredentialAssertionJSON>>("/auth/passkeys/login/start", {})
      const credential = await getPasskeyCredential(begin.options)
      const authData = await api.post<AuthResponse>("/auth/passkeys/login/finish", {
        session_id: begin.session_id,
        credential,
      })
      setAuth(authData.token, authData.user)
      toast.success(t("auth.login.success"))
      navigate("/")
    } catch (err) {
      setError(getPasskeyErrorMessage(err, t, "auth.login.passkeyError"))
    } finally {
      setPasskeyLoading(false)
    }
  }

  async function handleOIDCLogin() {
    setError("")
    setOidcSubmitting(true)
    try {
      const result = await api.post<OIDCStartResponse>("/auth/oidc/login/start", {})
      window.location.href = result.authorization_url
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.oidcError"))
      setOidcSubmitting(false)
    }
  }

  async function handleCreateDevAccount() {
    setError("")
    setCreateDevAccountLoading(true)
    try {
      await api.post<AuthResponse>("/auth/register", {
        username: "admin",
        email: "admin@dev.local",
        password: "123456",
      })
      setIdentifier("admin")
      setPassword("123456")
      toast.success(t("auth.login.devAccountSuccess"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("auth.login.devAccountError"))
    } finally {
      setCreateDevAccountLoading(false)
    }
  }

  if (step === "totp") {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <Card className="w-full max-w-sm">
          <CardHeader className="text-center">
            <CardTitle className="text-2xl font-bold tracking-tight">
              {t("auth.login.twoFactor.title")}
            </CardTitle>
            <CardDescription>{t("auth.login.twoFactor.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={(e) => void handleTotpSubmit(e)} className="space-y-4">
              {error && (
                <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="totp-code">{t("auth.login.twoFactor.description")}</Label>
                <Input
                  id="totp-code"
                  type="text"
                  inputMode="numeric"
                  pattern="[0-9a-fA-F]*"
                  maxLength={8}
                  placeholder={t("auth.login.twoFactor.codePlaceholder")}
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value)}
                  autoFocus
                  required
                />
              </div>
              <Button type="submit" className="w-full" disabled={loading || totpCode.length < 6}>
                {loading ? t("auth.login.twoFactor.submitting") : t("auth.login.twoFactor.submit")}
              </Button>
            </form>
            <button
              type="button"
              className="mt-3 block mx-auto text-sm text-muted-foreground hover:text-foreground underline underline-offset-4"
              onClick={handleBack}
            >
              {t("auth.login.twoFactor.back")}
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl font-bold tracking-tight">{t("auth.login.title")}</CardTitle>
          <CardDescription>{t("auth.login.description")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={(e) => void handleCredentialsSubmit(e)} className="space-y-4">
            {error && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {error}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="identifier">{t("auth.login.identifierLabel")}</Label>
              <Input
                id="identifier"
                type="text"
                placeholder={t("auth.login.identifierPlaceholder")}
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
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
            {oidcConfig?.enabled && (
              <Button
                type="button"
                variant="outline"
                className="w-full"
                disabled={loading || passkeyLoading || oidcLoading || oidcSubmitting}
                onClick={() => void handleOIDCLogin()}
              >
                {oidcSubmitting
                  ? t("auth.login.oidcSubmitting")
                  : t("auth.login.oidcSubmit", { provider: oidcConfig.provider_name })}
              </Button>
            )}
            <Button
              type="button"
              variant="outline"
              className="w-full"
              disabled={loading || passkeyLoading || oidcLoading || oidcSubmitting || !passkeySupported}
              onClick={() => void handlePasskeyLogin()}
            >
              {passkeyLoading ? t("auth.login.passkeySubmitting") : t("auth.login.passkeySubmit")}
            </Button>
            {oidcLoading && (
              <p className="text-xs text-muted-foreground">
                {t("auth.login.oidcProcessing")}
              </p>
            )}
            {!passkeySupported && (
              <p className="text-xs text-muted-foreground">
                {t("auth.login.passkeyUnsupported")}
              </p>
            )}
            {isDevMode && (
              <Button
                type="button"
                variant="secondary"
                className="w-full"
                disabled={loading || passkeyLoading || oidcLoading || oidcSubmitting || createDevAccountLoading}
                onClick={() => void handleCreateDevAccount()}
              >
                {createDevAccountLoading ? t("auth.login.devAccountSubmitting") : t("auth.login.devAccountCreate")}
              </Button>
            )}
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
