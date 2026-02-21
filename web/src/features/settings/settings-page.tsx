import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type FormEvent,
} from "react"
import { Link, useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ArrowLeft } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api, clearToken, setAuth } from "@/lib/api"
import {
  DEFAULT_CURRENCY_FALLBACK,
  getPresetCurrencies,
  getPresetCurrencyMeta,
} from "@/lib/currencies"
import {
  applyThemeColorScheme,
  applyTheme,
  getCustomThemeColors,
  getDefaultCustomThemeColors,
  getTheme,
  getThemeColorScheme,
  type CustomThemeColors,
  type Theme,
  type ThemeColorScheme,
} from "@/lib/theme"
import { toast } from "sonner"
import type {
  AuthResponse,
  ConfirmEmailChangeInput,
  CreateCurrencyInput,
  ReorderCurrencyItem,
  SendEmailChangeCodeInput,
  UpdateCurrencyInput,
  User,
  UserCurrency,
  UserPreference,
} from "@/types"

import SettingsAccountTab from "./settings-account-tab"
import SettingsGeneralTab from "./settings-general-tab"
import SettingsPaymentTab from "./settings-payment-tab"

const customCodeOption = "__custom__"
const currencyPlaceholderFallbackCode = "USD"
type SettingsTab = "general" | "payment" | "account"

function isSettingsTab(value: string): value is SettingsTab {
  return value === "general" || value === "payment" || value === "account"
}

function getIntlCurrencySymbolPlaceholder(code: string, locale: string): string | null {
  try {
    const parts = new Intl.NumberFormat(locale, {
      style: "currency",
      currency: code,
      currencyDisplay: "symbol",
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).formatToParts(0)
    return parts.find((part) => part.type === "currency")?.value ?? null
  } catch {
    return null
  }
}

function getIntlCurrencyAliasPlaceholder(code: string, locale: string): string | null {
  try {
    if (typeof Intl.DisplayNames !== "function") {
      return null
    }
    const alias = new Intl.DisplayNames([locale], { type: "currency" }).of(code)
    return alias ?? null
  } catch {
    return null
  }
}

export default function SettingsPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()

  const [theme, setTheme] = useState<Theme>(getTheme())
  const [themeColorScheme, setThemeColorScheme] = useState<ThemeColorScheme>(getThemeColorScheme())
  const [customThemeColors, setCustomThemeColors] = useState<CustomThemeColors>(
    getCustomThemeColors()
  )
  const [activeTab, setActiveTab] = useState<SettingsTab>("general")
  const [currency, setCurrency] = useState(localStorage.getItem("defaultCurrency") || "USD")
  const [user, setUser] = useState<User | null>(null)
  const [newEmail, setNewEmail] = useState("")
  const [emailChangePassword, setEmailChangePassword] = useState("")
  const [emailVerificationCode, setEmailVerificationCode] = useState("")
  const [emailCodeSent, setEmailCodeSent] = useState(false)
  const [emailCodeLoading, setEmailCodeLoading] = useState(false)
  const [emailChangeLoading, setEmailChangeLoading] = useState(false)
  const [emailChangeError, setEmailChangeError] = useState("")
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
  const paymentLoaded = useRef(false)
  const accountLoaded = useRef(false)

  useEffect(() => {
    if (activeTab !== "account" || accountLoaded.current) {
      return
    }
    accountLoaded.current = true
    api.get<User>("/auth/me").then(setUser).catch(() => void 0)
  }, [activeTab])

  useEffect(() => {
    if (activeTab !== "payment" || paymentLoaded.current) {
      return
    }
    paymentLoaded.current = true

    api.get<UserPreference>("/preferences/currency")
      .then((pref) => {
        if (pref?.preferred_currency) {
          setCurrency(pref.preferred_currency)
          localStorage.setItem("defaultCurrency", pref.preferred_currency)
        }
      })
      .catch(() => void 0)

    api.get<UserCurrency[]>("/currencies")
      .then((list) => {
        setUserCurrencies(list ?? [])
      })
      .catch(() => void 0)
  }, [activeTab])

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

  const addPlaceholderCode = useMemo(() => {
    if (addCode === customCodeOption) {
      return customCode.trim().toUpperCase() || currencyPlaceholderFallbackCode
    }
    return addCode || currencyPlaceholderFallbackCode
  }, [addCode, customCode])

  const intlFallbackSymbolPlaceholder = useMemo(
    () =>
      getIntlCurrencySymbolPlaceholder(currencyPlaceholderFallbackCode, i18n.language) ??
      currencyPlaceholderFallbackCode,
    [i18n.language]
  )

  const intlFallbackAliasPlaceholder = useMemo(
    () =>
      getIntlCurrencyAliasPlaceholder(currencyPlaceholderFallbackCode, i18n.language) ??
      currencyPlaceholderFallbackCode,
    [i18n.language]
  )

  const addSymbolPlaceholder = useMemo(() => {
    const preset = getPresetCurrencyMeta(addPlaceholderCode, i18n.language)
    return (
      preset?.symbol ??
      getIntlCurrencySymbolPlaceholder(addPlaceholderCode, i18n.language) ??
      intlFallbackSymbolPlaceholder
    )
  }, [addPlaceholderCode, i18n.language, intlFallbackSymbolPlaceholder])

  const addAliasPlaceholder = useMemo(() => {
    const preset = getPresetCurrencyMeta(addPlaceholderCode, i18n.language)
    return (
      preset?.alias ??
      getIntlCurrencyAliasPlaceholder(addPlaceholderCode, i18n.language) ??
      intlFallbackAliasPlaceholder
    )
  }, [addPlaceholderCode, i18n.language, intlFallbackAliasPlaceholder])

  const getCurrencySymbolPlaceholder = useCallback(
    (code: string): string => {
      const preset = getPresetCurrencyMeta(code, i18n.language)
      return (
        preset?.symbol ??
        getIntlCurrencySymbolPlaceholder(code, i18n.language) ??
        intlFallbackSymbolPlaceholder
      )
    },
    [i18n.language, intlFallbackSymbolPlaceholder]
  )

  const getCurrencyAliasPlaceholder = useCallback(
    (code: string): string => {
      const preset = getPresetCurrencyMeta(code, i18n.language)
      return (
        preset?.alias ??
        getIntlCurrencyAliasPlaceholder(code, i18n.language) ??
        intlFallbackAliasPlaceholder
      )
    },
    [i18n.language, intlFallbackAliasPlaceholder]
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

  function handleThemeColorScheme(nextScheme: ThemeColorScheme) {
    setThemeColorScheme(nextScheme)
    applyThemeColorScheme(nextScheme, customThemeColors)
  }

  function handleCustomThemeColorChange(key: keyof CustomThemeColors, value: string) {
    const nextColors: CustomThemeColors = {
      ...customThemeColors,
      [key]: value,
    }
    setCustomThemeColors(nextColors)
    applyThemeColorScheme(themeColorScheme, nextColors)
  }

  function handleResetCustomThemeColors() {
    const defaultColors = getDefaultCustomThemeColors()
    setCustomThemeColors(defaultColors)
    applyThemeColorScheme(themeColorScheme, defaultColors)
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

  async function handleChangePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
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
      setPasswordError(err instanceof Error ? err.message : t("settings.account.passwordError"))
    } finally {
      setPasswordLoading(false)
    }
  }

  async function handleSendEmailChangeCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setEmailChangeError("")
    setEmailCodeSent(false)

    if (!newEmail.trim()) {
      setEmailChangeError(t("settings.account.newEmailRequired"))
      return
    }
    if (!emailChangePassword) {
      setEmailChangeError(t("settings.account.emailChangePasswordRequired"))
      return
    }
    if (user?.email && newEmail.trim().toLowerCase() === user.email.toLowerCase()) {
      setEmailChangeError(t("settings.account.newEmailMustBeDifferent"))
      return
    }

    setEmailCodeLoading(true)
    try {
      const payload: SendEmailChangeCodeInput = {
        new_email: newEmail.trim(),
        password: emailChangePassword,
      }
      await api.post<{ message: string }>("/auth/email/change/send-code", payload)
      setEmailCodeSent(true)
      toast.success(t("settings.account.emailCodeSent"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailCodeLoading(false)
    }
  }

  async function handleConfirmEmailChange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setEmailChangeError("")

    if (!newEmail.trim()) {
      setEmailChangeError(t("settings.account.newEmailRequired"))
      return
    }
    if (!emailVerificationCode.trim()) {
      setEmailChangeError(t("settings.account.emailVerificationCodeRequired"))
      return
    }

    setEmailChangeLoading(true)
    try {
      const payload: ConfirmEmailChangeInput = {
        new_email: newEmail.trim(),
        verification_code: emailVerificationCode.trim(),
      }
      const authData = await api.post<AuthResponse>("/auth/email/change/confirm", payload)
      setAuth(authData.token, authData.user)
      setUser(authData.user)
      setNewEmail("")
      setEmailChangePassword("")
      setEmailVerificationCode("")
      setEmailCodeSent(false)
      toast.success(t("settings.account.emailChangeSuccess"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailChangeLoading(false)
    }
  }

  function handleLogout() {
    clearToken()
    toast.success(t("settings.account.logoutSuccess"))
    navigate("/login")
  }

  async function handleAddCurrency(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

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
      toast.error(
        err instanceof Error
          ? err.message
          : t("settings.currencyManagement.cannotDeletePreferred")
      )
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

  function handleDragOver(event: DragEvent<HTMLDivElement>, index: number) {
    event.preventDefault()
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

  const preferredCurrencyCodes =
    userCurrencies.length > 0
      ? userCurrencies.map((item) => item.code)
      : DEFAULT_CURRENCY_FALLBACK

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-4xl items-center gap-3 px-4">
          <Button variant="ghost" size="icon-sm" asChild>
            <Link to="/">
              <ArrowLeft className="size-4" />
            </Link>
          </Button>
          <h1 className="text-lg font-bold tracking-tight">{t("settings.title")}</h1>
        </div>
      </header>

      <main className="mx-auto max-w-4xl px-4 py-6">
        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            if (isSettingsTab(value)) {
              setActiveTab(value)
            }
          }}
          className="space-y-6"
        >
          <TabsList>
            <TabsTrigger value="general">{t("settings.general.title")}</TabsTrigger>
            <TabsTrigger value="payment">{t("settings.payment.title")}</TabsTrigger>
            <TabsTrigger value="account">{t("settings.account.title")}</TabsTrigger>
          </TabsList>

          <SettingsGeneralTab
            theme={theme}
            onThemeChange={handleTheme}
            colorScheme={themeColorScheme}
            onColorSchemeChange={handleThemeColorScheme}
            customThemeColors={customThemeColors}
            onCustomThemeColorChange={handleCustomThemeColorChange}
            onResetCustomThemeColors={handleResetCustomThemeColors}
            language={i18n.language}
            onLanguageChange={(language) => {
              void i18n.changeLanguage(language)
            }}
          />

          <SettingsPaymentTab
            currency={currency}
            preferredCurrencyCodes={preferredCurrencyCodes}
            onCurrencyChange={handleCurrency}
            userCurrencies={userCurrencies}
            orderChanged={orderChanged}
            orderSaving={orderSaving}
            onDragStart={handleDragStart}
            onDragOver={handleDragOver}
            onDrop={handleDrop}
            onSaveOrder={handleSaveOrder}
            onUpdateCurrency={handleUpdateCurrency}
            onDeleteCurrency={handleDeleteCurrency}
            getCurrencySymbolPlaceholder={getCurrencySymbolPlaceholder}
            getCurrencyAliasPlaceholder={getCurrencyAliasPlaceholder}
            addCode={addCode}
            onAddCodeChange={setAddCode}
            addableCurrencyCodes={addableCurrencyCodes}
            customCodeOption={customCodeOption}
            addSymbol={addSymbol}
            onAddSymbolChange={setAddSymbol}
            addSymbolPlaceholder={addSymbolPlaceholder}
            addAlias={addAlias}
            onAddAliasChange={setAddAlias}
            addAliasPlaceholder={addAliasPlaceholder}
            addLoading={addLoading}
            customCode={customCode}
            onCustomCodeChange={setCustomCode}
            onAddCurrency={handleAddCurrency}
          />

          <SettingsAccountTab
            user={user}
            onUserChange={setUser}
            newEmail={newEmail}
            onNewEmailChange={setNewEmail}
            emailChangePassword={emailChangePassword}
            onEmailChangePasswordChange={setEmailChangePassword}
            emailVerificationCode={emailVerificationCode}
            onEmailVerificationCodeChange={setEmailVerificationCode}
            emailCodeLoading={emailCodeLoading}
            emailChangeLoading={emailChangeLoading}
            emailCodeSent={emailCodeSent}
            emailChangeError={emailChangeError}
            onSendEmailChangeCode={handleSendEmailChangeCode}
            onConfirmEmailChange={handleConfirmEmailChange}
            passwordError={passwordError}
            passwordSuccess={passwordSuccess}
            currentPassword={currentPassword}
            newPassword={newPassword}
            confirmPassword={confirmPassword}
            passwordLoading={passwordLoading}
            onCurrentPasswordChange={setCurrentPassword}
            onNewPasswordChange={setNewPassword}
            onConfirmPasswordChange={setConfirmPassword}
            onChangePassword={handleChangePassword}
            onLogout={handleLogout}
          />
        </Tabs>
      </main>
    </div>
  )
}
