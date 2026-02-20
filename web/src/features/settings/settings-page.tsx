import { useState, useEffect, useMemo, useRef, type DragEvent, type FormEvent } from "react"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ArrowLeft, GripVertical, Monitor, Moon, Sun, Trash2 } from "lucide-react"
import { api, clearToken } from "@/lib/api"
import {
  DEFAULT_CURRENCY_FALLBACK,
  getPresetCurrencies,
  getPresetCurrencyMeta,
} from "@/lib/currencies"
import { getTheme, applyTheme, type Theme } from "@/lib/theme"
import { toast } from "sonner"
import type {
  CreateCurrencyInput,
  ReorderCurrencyItem,
  UpdateCurrencyInput,
  User,
  UserCurrency,
  UserPreference,
} from "@/types"
import TotpSection from "./totp-section"
import PasskeySection from "./passkey-section"
import CategoryManagement from "./category-management"

const languages = [
  { value: "en", label: "English" },
  { value: "zh-CN", label: "中文" },
  { value: "ja", label: "日本語" },
]

const customCodeOption = "__custom__"

export default function SettingsPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()

  const [theme, setTheme] = useState<Theme>(getTheme())
  const [currency, setCurrency] = useState(
    localStorage.getItem("defaultCurrency") || "USD"
  )
  const [user, setUser] = useState<User | null>(null)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [passwordError, setPasswordError] = useState("")
  const [passwordSuccess, setPasswordSuccess] = useState("")

  const [userCurrencies, setUserCurrencies] = useState<UserCurrency[]>([])
  const [addCode, setAddCode] = useState("")
  const [customCode, setCustomCode] = useState("")
  const [addSymbol, setAddSymbol] = useState("")
  const [addAlias, setAddAlias] = useState("")
  const [addLoading, setAddLoading] = useState(false)
  const [orderChanged, setOrderChanged] = useState(false)
  const [orderSaving, setOrderSaving] = useState(false)

  const dragFrom = useRef<number | null>(null)
  const dragTo = useRef<number | null>(null)

  useEffect(() => {
    api.get<User>("/auth/me").then(setUser).catch(() => void 0)
    api.get<UserPreference>("/preferences/currency").then((pref) => {
      if (pref?.preferred_currency) {
        setCurrency(pref.preferred_currency)
        localStorage.setItem("defaultCurrency", pref.preferred_currency)
      }
    }).catch(() => void 0)
    api.get<UserCurrency[]>("/currencies").then((list) => {
      setUserCurrencies(list ?? [])
    }).catch(() => void 0)
  }, [])

  useEffect(() => {
    if (userCurrencies.length === 0) {
      return
    }

    if (!userCurrencies.some((item) => item.code === currency)) {
      const nextCurrency = userCurrencies[0].code
      setCurrency(nextCurrency)
      localStorage.setItem("defaultCurrency", nextCurrency)
      api.put("/preferences/currency", { preferred_currency: nextCurrency }).catch(() => void 0)
    }
  }, [currency, userCurrencies])

  const addableCurrencyCodes = useMemo(
    () =>
      getPresetCurrencies(i18n.language)
        .map((item) => item.code)
        .filter((code) => !userCurrencies.some((item) => item.code === code)),
    [i18n.language, userCurrencies]
  )

  useEffect(() => {
    if (addCode === customCodeOption) {
      setAddSymbol("")
      setAddAlias("")
      return
    }

    if (addableCurrencyCodes.length === 0) {
      setAddCode(customCodeOption)
      return
    }

    if (!addCode || !addableCurrencyCodes.includes(addCode)) {
      setAddCode(addableCurrencyCodes[0])
    }
  }, [addCode, addableCurrencyCodes])

  useEffect(() => {
    if (addCode === customCodeOption) {
      return
    }

    const preset = getPresetCurrencyMeta(addCode, i18n.language)
    if (!preset) {
      setAddSymbol("")
      setAddAlias("")
      return
    }

    setAddSymbol(preset.symbol)
    setAddAlias(preset.alias)
  }, [addCode, i18n.language])

  function handleTheme(next: Theme) {
    setTheme(next)
    applyTheme(next)
  }

  async function handleCurrency(value: string) {
    setCurrency(value)
    localStorage.setItem("defaultCurrency", value)
    try {
      await api.put("/preferences/currency", { preferred_currency: value })
    } catch {
      void 0
    }
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

  async function handleAddCurrency(e: FormEvent) {
    e.preventDefault()

    const code = (addCode === customCodeOption ? customCode : addCode).trim().toUpperCase()
    const preset = getPresetCurrencyMeta(code, i18n.language)
    if (!code) {
      toast.error(t("settings.currencyManagement.invalidCode"))
      return
    }

    setAddLoading(true)
    try {
      const input: CreateCurrencyInput = {
        code,
        symbol: addSymbol.trim() || preset?.symbol || "",
        alias: addAlias.trim() || preset?.alias || "",
        sort_order: userCurrencies.length,
      }
      const created = await api.post<UserCurrency>("/currencies", input)
      setUserCurrencies((prev) => [...prev, created])
      if (addCode === customCodeOption) {
        setCustomCode("")
        setAddSymbol("")
        setAddAlias("")
      }
      toast.success(t("settings.currencyManagement.addSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.currencyManagement.invalidCode"))
    } finally {
      setAddLoading(false)
    }
  }

  async function handleDeleteCurrency(id: number) {
    if (!window.confirm(t("settings.currencyManagement.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/currencies/${id}`)
      setUserCurrencies((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("settings.currencyManagement.deleteSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("settings.currencyManagement.cannotDeletePreferred"))
    }
  }

  async function handleUpdateCurrency(id: number, input: UpdateCurrencyInput) {
    try {
      const updated = await api.put<UserCurrency>(`/currencies/${id}`, input)
      setUserCurrencies((prev) => prev.map((item) => (item.id === id ? updated : item)))
      toast.success(t("settings.currencyManagement.updateSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    }
  }

  function handleDragStart(index: number) {
    dragFrom.current = index
  }

  function handleDragOver(e: DragEvent<HTMLDivElement>, index: number) {
    e.preventDefault()
    dragTo.current = index
  }

  function handleDrop() {
    if (dragFrom.current === null || dragTo.current === null || dragFrom.current === dragTo.current) {
      return
    }

    const reordered = [...userCurrencies]
    const [moved] = reordered.splice(dragFrom.current, 1)
    reordered.splice(dragTo.current, 0, moved)
    setUserCurrencies(reordered)
    setOrderChanged(true)
    dragFrom.current = null
    dragTo.current = null
  }

  async function handleSaveOrder() {
    setOrderSaving(true)
    try {
      const payload: ReorderCurrencyItem[] = userCurrencies.map((item, index) => ({
        id: item.id,
        sort_order: index,
      }))
      await api.put("/currencies/reorder", payload)
      setUserCurrencies((prev) =>
        prev.map((item, index) => ({
          ...item,
          sort_order: index,
        }))
      )
      setOrderChanged(false)
      toast.success(t("settings.currencyManagement.orderSaved"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    } finally {
      setOrderSaving(false)
    }
  }

  const preferredCurrencyCodes = userCurrencies.length > 0
    ? userCurrencies.map((item) => item.code)
    : DEFAULT_CURRENCY_FALLBACK

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

      <main className="mx-auto max-w-4xl px-4 py-6">
        <Tabs defaultValue="general" className="space-y-6">
          <TabsList>
            <TabsTrigger value="general">{t("settings.general.title")}</TabsTrigger>
            <TabsTrigger value="payment">{t("settings.payment.title")}</TabsTrigger>
            <TabsTrigger value="account">{t("settings.account.title")}</TabsTrigger>
          </TabsList>

          <TabsContent value="general" className="space-y-6">
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
          </TabsContent>

          <TabsContent value="payment" className="space-y-6">
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
                    {preferredCurrencyCodes.map((item) => (
                      <SelectItem key={item} value={item}>
                        {item}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <Separator />

            <div>
              <h2 className="text-sm font-medium">
                {t("settings.currencyManagement.title")}
              </h2>
              <p className="text-sm text-muted-foreground mt-0.5">
                {t("settings.currencyManagement.description")}
              </p>

              <div className="mt-3 space-y-1">
                {userCurrencies.length === 0 && (
                  <p className="text-sm text-muted-foreground py-2">
                    {t("settings.currencyManagement.empty")}
                  </p>
                )}

                {userCurrencies.map((item, index) => (
                  <div
                    key={item.id}
                    draggable
                    onDragStart={() => handleDragStart(index)}
                    onDragOver={(e) => handleDragOver(e, index)}
                    onDrop={handleDrop}
                    className="grid grid-cols-[1rem_3rem_5rem_minmax(0,1fr)_1.75rem] items-center gap-2 rounded-md border bg-card px-2 py-1.5"
                  >
                    <GripVertical className="size-4 text-muted-foreground shrink-0 cursor-grab" />
                    <span className="inline-flex h-7 items-center text-sm font-mono font-medium">{item.code}</span>

                    <Input
                      className="h-7 w-full px-2 text-sm"
                      placeholder={t("settings.currencyManagement.symbolPlaceholder")}
                      defaultValue={item.symbol}
                      maxLength={10}
                      onBlur={(e) => {
                        if (e.target.value !== item.symbol) {
                          void handleUpdateCurrency(item.id, { symbol: e.target.value })
                        }
                      }}
                    />

                    <Input
                      className="h-7 w-full px-2 text-sm"
                      placeholder={t("settings.currencyManagement.aliasPlaceholder")}
                      defaultValue={item.alias}
                      maxLength={100}
                      onBlur={(e) => {
                        if (e.target.value !== item.alias) {
                          void handleUpdateCurrency(item.id, { alias: e.target.value })
                        }
                      }}
                    />

                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="size-7 text-muted-foreground hover:text-destructive"
                      onClick={() => void handleDeleteCurrency(item.id)}
                    >
                      <Trash2 className="size-3.5" />
                    </Button>
                  </div>
                ))}
              </div>

              {orderChanged && (
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-2"
                  disabled={orderSaving}
                  onClick={() => void handleSaveOrder()}
                >
                  {orderSaving
                    ? t("settings.currencyManagement.savingOrder")
                    : t("settings.currencyManagement.saveOrder")}
                </Button>
              )}

              <form onSubmit={(e) => void handleAddCurrency(e)} className="mt-3 space-y-2">
                <div className="grid gap-1 sm:grid-cols-[6rem_5rem_minmax(0,1fr)_auto]">
                  <Label className="text-xs text-muted-foreground">
                    {t("settings.currencyManagement.codeLabel")}
                  </Label>
                  <Label className="text-xs text-muted-foreground">
                    {t("settings.currencyManagement.symbolLabel")}
                  </Label>
                  <Label className="text-xs text-muted-foreground">
                    {t("settings.currencyManagement.aliasLabel")}
                  </Label>
                  <Label className="text-xs text-transparent">
                    {t("settings.currencyManagement.addButton")}
                  </Label>
                </div>

                <div className="grid items-center gap-2 sm:grid-cols-[6rem_5rem_minmax(0,1fr)_auto]">
                  <Select value={addCode} onValueChange={setAddCode}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder={t("settings.currencyManagement.codePlaceholder")} />
                    </SelectTrigger>
                    <SelectContent>
                      {addableCurrencyCodes.map((code) => (
                        <SelectItem key={code} value={code}>
                          {code}
                        </SelectItem>
                      ))}
                      <SelectItem value={customCodeOption}>
                        {t("settings.currencyManagement.customCode")}
                      </SelectItem>
                    </SelectContent>
                  </Select>

                  <Input
                    className="w-full text-sm"
                    placeholder={t("settings.currencyManagement.symbolPlaceholder")}
                    value={addSymbol}
                    onChange={(e) => setAddSymbol(e.target.value)}
                    maxLength={10}
                  />

                  <Input
                    className="w-full text-sm"
                    placeholder={t("settings.currencyManagement.aliasPlaceholder")}
                    value={addAlias}
                    onChange={(e) => setAddAlias(e.target.value)}
                    maxLength={100}
                  />

                  <Button
                    type="submit"
                    className="sm:min-w-20"
                    disabled={
                      addLoading ||
                      (addCode === customCodeOption ? customCode.trim() === "" : addCode.trim() === "")
                    }
                  >
                    {addLoading
                      ? t("settings.currencyManagement.adding")
                      : t("settings.currencyManagement.addButton")}
                  </Button>
                </div>

                {addCode === customCodeOption && (
                  <div className="grid gap-2 sm:max-w-72">
                    <div className="space-y-1">
                      <Label className="text-xs text-muted-foreground">
                        {t("settings.currencyManagement.customCode")}
                      </Label>
                      <Input
                        className="w-full text-sm uppercase"
                        placeholder={t("settings.currencyManagement.codePlaceholder")}
                        value={customCode}
                        onChange={(e) => setCustomCode(e.target.value.toUpperCase())}
                        maxLength={10}
                      />
                    </div>
                  </div>
                )}
              </form>
            </div>

            <Separator />

            <CategoryManagement />
          </TabsContent>

          <TabsContent value="account">
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

              <TotpSection user={user} onUserChange={setUser} />

              <Separator />

              <PasskeySection />

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
          </TabsContent>
        </Tabs>
      </main>
    </div>
  )
}
