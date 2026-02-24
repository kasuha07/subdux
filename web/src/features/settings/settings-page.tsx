import { useEffect, useRef, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ArrowLeft, Bell, CircleUserRound, CreditCard, Info, KeyRound, Settings } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useSettingsAccount } from "@/features/settings/hooks/use-settings-account"
import { useSettingsPayment } from "@/features/settings/hooks/use-settings-payment"
import { api } from "@/lib/api"
import {
  getDisplayAllAmountsInPrimaryCurrency,
  getDisplayRecurringAmountsAsMonthlyCost,
  getDisplaySubscriptionCycleProgress,
  setDisplayAllAmountsInPrimaryCurrency,
  setDisplayRecurringAmountsAsMonthlyCost,
  setDisplaySubscriptionCycleProgress,
} from "@/lib/display-preferences"
import type { VersionInfo } from "@/types"
import {
  applyTheme,
  applyThemeColorScheme,
  getCustomThemeColors,
  getDefaultCustomThemeColors,
  getTheme,
  getThemeColorScheme,
  type CustomThemeColors,
  type Theme,
  type ThemeColorScheme,
} from "@/lib/theme"

import SettingsAboutTab from "./settings-about-tab"
import SettingsAccountTab from "./settings-account-tab"
import SettingsAPIKeyTab from "./settings-apikey-tab"
import SettingsGeneralTab from "./settings-general-tab"
import SettingsNotificationTab from "./settings-notification-tab"
import SettingsPaymentTab from "./settings-payment-tab"

type SettingsTab = "general" | "payment" | "notification" | "account" | "apikey" | "about"

function isSettingsTab(value: string): value is SettingsTab {
  return value === "general" || value === "payment" || value === "notification" || value === "account" || value === "apikey" || value === "about"
}

export default function SettingsPage() {
  const { t, i18n } = useTranslation()

  const [theme, setTheme] = useState<Theme>(getTheme())
  const [themeColorScheme, setThemeColorScheme] = useState<ThemeColorScheme>(getThemeColorScheme())
  const [customThemeColors, setCustomThemeColors] = useState<CustomThemeColors>(
    getCustomThemeColors()
  )
  const [displayAllAmountsInPrimaryCurrency, setDisplayAllAmountsInPrimaryCurrencyState] = useState(
    getDisplayAllAmountsInPrimaryCurrency()
  )
  const [displayRecurringAmountsAsMonthlyCost, setDisplayRecurringAmountsAsMonthlyCostState] = useState(
    getDisplayRecurringAmountsAsMonthlyCost()
  )
  const [displaySubscriptionCycleProgress, setDisplaySubscriptionCycleProgressState] = useState(
    getDisplaySubscriptionCycleProgress()
  )
  const [activeTab, setActiveTab] = useState<SettingsTab>("general")
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null)
  const versionRequestedRef = useRef(false)

  const account = useSettingsAccount({ active: activeTab === "account" })
  const payment = useSettingsPayment({ active: activeTab === "payment" })

  useEffect(() => {
    if (activeTab !== "about" || versionRequestedRef.current) return
    versionRequestedRef.current = true
    api.get<VersionInfo>("/version").then(setVersionInfo).catch(() => {})
  }, [activeTab])

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

  function handleDisplayAllAmountsInPrimaryCurrency(enabled: boolean) {
    setDisplayAllAmountsInPrimaryCurrencyState(enabled)
    setDisplayAllAmountsInPrimaryCurrency(enabled)
  }

  function handleDisplayRecurringAmountsAsMonthlyCost(enabled: boolean) {
    setDisplayRecurringAmountsAsMonthlyCostState(enabled)
    setDisplayRecurringAmountsAsMonthlyCost(enabled)
  }

  function handleDisplaySubscriptionCycleProgress(enabled: boolean) {
    setDisplaySubscriptionCycleProgressState(enabled)
    setDisplaySubscriptionCycleProgress(enabled)
  }

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
            <TabsTrigger value="general" className="gap-2">
              <Settings className="size-4" />
              {t("settings.general.title")}
            </TabsTrigger>
            <TabsTrigger value="payment" className="gap-2">
              <CreditCard className="size-4" />
              {t("settings.payment.title")}
            </TabsTrigger>
            <TabsTrigger value="notification" className="gap-2">
              <Bell className="size-4" />
              {t("settings.notifications.title")}
            </TabsTrigger>
            <TabsTrigger value="account" className="gap-2">
              <CircleUserRound className="size-4" />
              {t("settings.account.title")}
            </TabsTrigger>
            <TabsTrigger value="apikey" className="gap-2">
              <KeyRound className="size-4" />
              {t("settings.apiKeys.title")}
            </TabsTrigger>
            <TabsTrigger value="about" className="gap-2">
              <Info className="size-4" />
              {t("settings.about.title")}
            </TabsTrigger>
          </TabsList>

          <SettingsGeneralTab
            theme={theme}
            onThemeChange={handleTheme}
            colorScheme={themeColorScheme}
            onColorSchemeChange={handleThemeColorScheme}
            customThemeColors={customThemeColors}
            onCustomThemeColorChange={handleCustomThemeColorChange}
            onResetCustomThemeColors={handleResetCustomThemeColors}
            displayAllAmountsInPrimaryCurrency={displayAllAmountsInPrimaryCurrency}
            onDisplayAllAmountsInPrimaryCurrencyChange={handleDisplayAllAmountsInPrimaryCurrency}
            displayRecurringAmountsAsMonthlyCost={displayRecurringAmountsAsMonthlyCost}
            onDisplayRecurringAmountsAsMonthlyCostChange={handleDisplayRecurringAmountsAsMonthlyCost}
            displaySubscriptionCycleProgress={displaySubscriptionCycleProgress}
            onDisplaySubscriptionCycleProgressChange={handleDisplaySubscriptionCycleProgress}
            language={i18n.language}
            onLanguageChange={(language) => {
              void i18n.changeLanguage(language)
            }}
          />

          <SettingsPaymentTab
            currency={payment.currency}
            preferredCurrencyCodes={payment.preferredCurrencyCodes}
            onCurrencyChange={payment.handleCurrency}
            userCurrencies={payment.userCurrencies}
            orderChanged={payment.orderChanged}
            orderSaving={payment.orderSaving}
            onDragStart={payment.handleDragStart}
            onDragOver={payment.handleDragOver}
            onDrop={payment.handleDrop}
            onSaveOrder={payment.handleSaveOrder}
            onUpdateCurrency={payment.handleUpdateCurrency}
            onDeleteCurrency={payment.handleDeleteCurrency}
            getCurrencySymbolPlaceholder={payment.getCurrencySymbolPlaceholder}
            getCurrencyAliasPlaceholder={payment.getCurrencyAliasPlaceholder}
            addCode={payment.addCode}
            onAddCodeChange={payment.setAddCode}
            addableCurrencyCodes={payment.addableCurrencyCodes}
            customCodeOption={payment.customCodeOption}
            addSymbol={payment.addSymbol}
            onAddSymbolChange={payment.setAddSymbol}
            addSymbolPlaceholder={payment.addSymbolPlaceholder}
            addAlias={payment.addAlias}
            onAddAliasChange={payment.setAddAlias}
            addAliasPlaceholder={payment.addAliasPlaceholder}
            addLoading={payment.addLoading}
            customCode={payment.customCode}
            onCustomCodeChange={payment.setCustomCode}
            onAddCurrency={payment.handleAddCurrency}
          />

          <SettingsNotificationTab active={activeTab === "notification"} />

          <SettingsAccountTab
            user={account.user}
            onUserChange={account.setUser}
            newEmail={account.newEmail}
            onNewEmailChange={account.setNewEmail}
            emailChangePassword={account.emailChangePassword}
            onEmailChangePasswordChange={account.setEmailChangePassword}
            emailVerificationCode={account.emailVerificationCode}
            onEmailVerificationCodeChange={account.setEmailVerificationCode}
            emailCodeLoading={account.emailCodeLoading}
            emailChangeLoading={account.emailChangeLoading}
            emailCodeSent={account.emailCodeSent}
            emailChangeError={account.emailChangeError}
            onSendEmailChangeCode={account.handleSendEmailChangeCode}
            onConfirmEmailChange={account.handleConfirmEmailChange}
            passwordError={account.passwordError}
            passwordSuccess={account.passwordSuccess}
            currentPassword={account.currentPassword}
            newPassword={account.newPassword}
            confirmPassword={account.confirmPassword}
            passwordLoading={account.passwordLoading}
            onCurrentPasswordChange={account.setCurrentPassword}
            onNewPasswordChange={account.setNewPassword}
            onConfirmPasswordChange={account.setConfirmPassword}
            onChangePassword={account.handleChangePassword}
            onLogout={account.handleLogout}
          />

          <SettingsAPIKeyTab active={activeTab === "apikey"} />

          <SettingsAboutTab versionInfo={versionInfo} />
        </Tabs>
      </main>
    </div>
  )
}
