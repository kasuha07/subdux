import { useState, useEffect, type FormEvent } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ArrowLeft, Sun, Moon, Monitor } from "lucide-react"
import { api, clearToken } from "@/lib/api"
import { getTheme, applyTheme, type Theme } from "@/lib/theme"
import { toast } from "sonner"
import type { User } from "@/types"

const languages = [
  { value: "en", label: "English" },
  { value: "zh-CN", label: "中文" },
  { value: "ja", label: "日本語" },
]

const currencies = ["USD", "EUR", "GBP", "JPY", "CNY", "CAD", "AUD"]

export default function SettingsPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()

  // Appearance
  const [theme, setTheme] = useState<Theme>(getTheme())

  // Currency
  const [currency, setCurrency] = useState(
    localStorage.getItem("defaultCurrency") || "USD"
  )

  // Account
  const [user, setUser] = useState<User | null>(null)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [passwordError, setPasswordError] = useState("")
  const [passwordSuccess, setPasswordSuccess] = useState("")

  useEffect(() => {
    api.get<User>("/auth/me").then(setUser).catch(() => void 0)
  }, [])

  function handleTheme(next: Theme) {
    setTheme(next)
    applyTheme(next)
  }

  function handleCurrency(value: string) {
    setCurrency(value)
    localStorage.setItem("defaultCurrency", value)
  }

  async function handleChangePassword(e: FormEvent) {
    e.preventDefault()
    setPasswordError("")
    setPasswordSuccess("")

    if (newPassword !== confirmPassword) {
      setPasswordError(t("settings.account.passwordMismatch"))
      return
    }
    if (newPassword.length < 6) {
      setPasswordError(t("settings.account.passwordTooShort"))
      return
    }

    setPasswordLoading(true)
    try {
      await api.put("/auth/password", {
        current_password: currentPassword,
        new_password: newPassword,
      })
      toast.success(t("settings.account.passwordChanged"))
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err) {
      setPasswordError(
        err instanceof Error ? err.message : t("settings.account.passwordError")
      )
    } finally {
      setPasswordLoading(false)
    }
  }

  function handleLogout() {
    clearToken()
    toast.success(t("settings.account.logoutSuccess"))
    navigate("/login")
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-4xl items-center px-4 gap-3">
          <Button variant="ghost" size="icon-sm" asChild>
            <Link to="/">
              <ArrowLeft className="size-4" />
            </Link>
          </Button>
          <h1 className="text-lg font-bold tracking-tight">
            {t("settings.title")}
          </h1>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-6 space-y-8">
        {/* Appearance */}
        <div>
          <h2 className="text-sm font-medium">
            {t("settings.appearance.title")}
          </h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.appearance.description")}
          </p>
          <div className="mt-3 flex gap-2">
            <Button
              size="sm"
              variant={theme === "light" ? "default" : "outline"}
              onClick={() => handleTheme("light")}
            >
              <Sun className="size-4" />
              {t("settings.appearance.light")}
            </Button>
            <Button
              size="sm"
              variant={theme === "dark" ? "default" : "outline"}
              onClick={() => handleTheme("dark")}
            >
              <Moon className="size-4" />
              {t("settings.appearance.dark")}
            </Button>
            <Button
              size="sm"
              variant={theme === "system" ? "default" : "outline"}
              onClick={() => handleTheme("system")}
            >
              <Monitor className="size-4" />
              {t("settings.appearance.system")}
            </Button>
          </div>
        </div>

        <Separator />

        {/* Language */}
        <div>
          <h2 className="text-sm font-medium">
            {t("settings.language.title")}
          </h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.language.description")}
          </p>
          <div className="mt-3">
            <Select
              value={i18n.language}
              onValueChange={(v) => i18n.changeLanguage(v)}
            >
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {languages.map((lang) => (
                  <SelectItem key={lang.value} value={lang.value}>
                    {lang.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <Separator />

        {/* Currency */}
        <div>
          <h2 className="text-sm font-medium">
            {t("settings.currency.title")}
          </h2>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.currency.description")}
          </p>
          <div className="mt-3">
            <Select value={currency} onValueChange={handleCurrency}>
              <SelectTrigger className="w-48">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {currencies.map((c) => (
                  <SelectItem key={c} value={c}>
                    {c}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <Separator />

        {/* Account */}
        <div className="space-y-4">
          <div>
            <h2 className="text-sm font-medium">
              {t("settings.account.title")}
            </h2>
            <p className="text-sm text-muted-foreground mt-0.5">
              {t("settings.account.description")}
            </p>
          </div>

          <div>
            <Label className="text-muted-foreground text-xs">
              {t("settings.account.email")}
            </Label>
            <p className="text-sm mt-0.5">{user?.email ?? "—"}</p>
          </div>

          <Separator />

          <div>
            <h3 className="text-sm font-medium">
              {t("settings.account.changePassword")}
            </h3>
            <form onSubmit={handleChangePassword} className="mt-3 grid gap-3 max-w-sm">
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
                <Label htmlFor="current-password">
                  {t("settings.account.currentPassword")}
                </Label>
                <Input
                  id="current-password"
                  type="password"
                  placeholder="••••••••"
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="new-password">
                  {t("settings.account.newPassword")}
                </Label>
                <Input
                  id="new-password"
                  type="password"
                  placeholder="••••••••"
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">
                  {t("settings.account.confirmPassword")}
                </Label>
                <Input
                  id="confirm-password"
                  type="password"
                  placeholder="••••••••"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
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
            <p className="text-sm text-muted-foreground">
              {t("settings.account.logoutDescription")}
            </p>
            <Button
              variant="outline"
              size="sm"
              className="mt-2"
              onClick={handleLogout}
            >
              {t("settings.account.logout")}
            </Button>
          </div>
        </div>
      </main>
    </div>
  )
}
