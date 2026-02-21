import { useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ArrowLeft } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useSettingsAccount } from "@/features/settings/hooks/use-settings-account"
import { useSettingsPayment } from "@/features/settings/hooks/use-settings-payment"
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

import SettingsAccountTab from "./settings-account-tab"
import SettingsGeneralTab from "./settings-general-tab"
import SettingsNotificationTab from "./settings-notification-tab"
import SettingsPaymentTab from "./settings-payment-tab"

type SettingsTab = "general" | "payment" | "notification" | "account"

function isSettingsTab(value: string): value is SettingsTab {
  return value === "general" || value === "payment" || value === "notification" || value === "account"
}

export default function SettingsPage() {
  const { t, i18n } = useTranslation()

  const [theme, setTheme] = useState<Theme>(getTheme())
  const [themeColorScheme, setThemeColorScheme] = useState<ThemeColorScheme>(getThemeColorScheme())
  const [customThemeColors, setCustomThemeColors] = useState<CustomThemeColors>(
    getCustomThemeColors()
  )
  const [activeTab, setActiveTab] = useState<SettingsTab>("general")

  const account = useSettingsAccount({ active: activeTab === "account" })
  const payment = useSettingsPayment({ active: activeTab === "payment" })

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
            <TabsTrigger value="notification">{t("settings.notifications.title")}</TabsTrigger>
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

          <SettingsNotificationTab />

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
        </Tabs>
      </main>
    </div>
  )
}
