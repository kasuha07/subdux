import { Suspense, lazy, useEffect, useLayoutEffect, useRef, useState } from "react"
import { Link } from "react-router"
import { useTranslation } from "react-i18next"
import { ArrowLeft, Bell, CircleUserRound, CreditCard, FileClock, Info, KeyRound, Settings } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip } from "@/components/ui/tooltip"
import { useSettingsAccount } from "@/features/settings/hooks/use-settings-account"
import { useSettingsPayment } from "@/features/settings/hooks/use-settings-payment"
import { api } from "@/lib/api"
import {
  getDisplayAllAmountsInPrimaryCurrency,
  getDisplayDisabledSubscriptionsLast,
  getDisplayRecurringAmountsAsMonthlyCost,
  getDisplaySubscriptionCycleProgress,
  setDisplayAllAmountsInPrimaryCurrency,
  setDisplayDisabledSubscriptionsLast,
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
import { cn } from "@/lib/utils"

const SettingsAboutTab = lazy(() => import("./settings-about-tab"))
const SettingsAccountTab = lazy(() => import("./settings-account-tab"))
const SettingsAuditTab = lazy(() => import("./settings-audit-tab"))
const SettingsAPIKeyTab = lazy(() => import("./settings-apikey-tab"))
const SettingsGeneralTab = lazy(() => import("./settings-general-tab"))
const SettingsNotificationTab = lazy(() => import("./settings-notification-tab"))
const SettingsPaymentTab = lazy(() => import("./settings-payment-tab"))

import { SettingsTabLoading } from "./settings-loading-skeleton"

export { SettingsTabLoading }

type SettingsTab = "general" | "payment" | "notification" | "account" | "apikey" | "audit" | "about"

function isSettingsTab(value: string): value is SettingsTab {
  return value === "general" || value === "payment" || value === "notification" || value === "account" || value === "apikey" || value === "audit" || value === "about"
}

const TAB_DEFINITIONS: {
  value: SettingsTab
  labelKey: string
  icon: typeof Settings
  loader: () => Promise<unknown>
}[] = [
  { value: "general", labelKey: "settings.general.title", icon: Settings, loader: () => import("./settings-general-tab") },
  { value: "payment", labelKey: "settings.payment.title", icon: CreditCard, loader: () => import("./settings-payment-tab") },
  { value: "notification", labelKey: "settings.notifications.title", icon: Bell, loader: () => import("./settings-notification-tab") },
  { value: "account", labelKey: "settings.account.title", icon: CircleUserRound, loader: () => import("./settings-account-tab") },
  { value: "apikey", labelKey: "settings.apiKeys.title", icon: KeyRound, loader: () => import("./settings-apikey-tab") },
  { value: "audit", labelKey: "settings.audit.title", icon: FileClock, loader: () => import("./settings-audit-tab") },
  { value: "about", labelKey: "settings.about.title", icon: Info, loader: () => import("./settings-about-tab") },
]

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
  const [displayDisabledSubscriptionsLast, setDisplayDisabledSubscriptionsLastState] = useState(
    getDisplayDisabledSubscriptionsLast()
  )
  const [activeTab, setActiveTab] = useState<SettingsTab>("general")
  const [visitedTabs, setVisitedTabs] = useState<SettingsTab[]>(["general"])
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null)
  const versionRequestedRef = useRef(false)

  const tabsListRef = useRef<HTMLDivElement>(null)
  const [indicatorStyle, setIndicatorStyle] = useState<{
    left: number
    top: number
    width: number
    height: number
  } | null>(null)
  const [isTransitionReady, setIsTransitionReady] = useState(false)

  // Preload tab components during browser idle time so tab transitions are instant
  useEffect(() => {
    const preloadAll = () => {
      TAB_DEFINITIONS.forEach(({ loader }) => {
        void loader()
      })
    }

    if (typeof window !== "undefined") {
      if ("requestIdleCallback" in window) {
        const handle = window.requestIdleCallback(preloadAll)
        return () => window.cancelIdleCallback(handle)
      }
      const timer = setTimeout(preloadAll, 200)
      return () => clearTimeout(timer)
    }
  }, [])

  useLayoutEffect(() => {
    const list = tabsListRef.current
    if (!list) return

    const updateIndicator = () => {
      const activeTrigger = list.querySelector<HTMLElement>(
        '[data-slot="tabs-trigger"][data-state="active"]'
      )
      if (activeTrigger) {
        setIndicatorStyle({
          left: activeTrigger.offsetLeft,
          top: activeTrigger.offsetTop,
          width: activeTrigger.offsetWidth,
          height: activeTrigger.offsetHeight,
        })
      }
    }

    updateIndicator()

    const rafId = requestAnimationFrame(() => {
      setIsTransitionReady(true)
    })

    const observer = new ResizeObserver(updateIndicator)
    observer.observe(list)

    return () => {
      cancelAnimationFrame(rafId)
      observer.disconnect()
    }
  }, [activeTab])

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

  function handleDisplayDisabledSubscriptionsLast(enabled: boolean) {
    setDisplayDisabledSubscriptionsLastState(enabled)
    setDisplayDisabledSubscriptionsLast(enabled)
  }

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-4xl items-center gap-3 px-4">
          <Tooltip content={t("common.back")}>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/" aria-label={t("common.back")}>
                <ArrowLeft className="size-4" />
              </Link>
            </Button>
          </Tooltip>
          <h1 className="text-lg font-bold tracking-tight">{t("settings.title")}</h1>
        </div>
      </header>

      <main className="page-stage-enter mx-auto max-w-4xl px-4 py-6">
        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            if (isSettingsTab(value)) {
              setActiveTab(value)
              setVisitedTabs((previous) => (
                previous.includes(value) ? previous : [...previous, value]
              ))
            }
          }}
          className="settings-tabs page-content-enter space-y-6"
        >
          <div className="w-full overflow-x-auto pb-1">
            <TabsList
              ref={tabsListRef}
              className={cn(
                "relative w-max min-w-max",
                indicatorStyle && "settings-tabs-list"
              )}
            >
              {indicatorStyle && (
                <span
                  aria-hidden="true"
                  className={cn(
                    "pointer-events-none absolute rounded-md bg-background shadow-xs dark:bg-input/30 dark:border dark:border-input",
                    isTransitionReady && "transition-all duration-200 ease-[cubic-bezier(0.22,1,0.36,1)]"
                  )}
                  style={{
                    left: `${indicatorStyle.left}px`,
                    top: `${indicatorStyle.top}px`,
                    width: `${indicatorStyle.width}px`,
                    height: `${indicatorStyle.height}px`,
                  }}
                />
              )}
              {TAB_DEFINITIONS.map(({ value, labelKey, icon: Icon, loader }) => (
                <TabsTrigger
                  key={value}
                  value={value}
                  className="flex-none gap-2"
                  onPointerEnter={() => void loader()}
                  onFocus={() => void loader()}
                >
                  <Icon className="size-4" />
                  {t(labelKey)}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>

          {visitedTabs.includes("general") && (
            <Suspense fallback={<SettingsTabLoading value="general" />}>
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
                displayDisabledSubscriptionsLast={displayDisabledSubscriptionsLast}
                onDisplayDisabledSubscriptionsLastChange={handleDisplayDisabledSubscriptionsLast}
                language={i18n.language}
                onLanguageChange={(language) => {
                  void i18n.changeLanguage(language)
                }}
              />
            </Suspense>
          )}

          {visitedTabs.includes("payment") && (
            <Suspense fallback={<SettingsTabLoading value="payment" />}>
              <SettingsPaymentTab
                currency={payment.currency}
                currencySaving={payment.currencySaving}
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
            </Suspense>
          )}

          {visitedTabs.includes("notification") && (
            <Suspense fallback={<SettingsTabLoading value="notification" />}>
              <SettingsNotificationTab active={activeTab === "notification"} />
            </Suspense>
          )}

          {visitedTabs.includes("account") && (
            <Suspense fallback={<SettingsTabLoading value="account" />}>
              <SettingsAccountTab
                user={account.user}
                onUserChange={account.setUser}
                newEmail={account.newEmail}
                onNewEmailChange={account.setNewEmail}
                emailVerificationCode={account.emailVerificationCode}
                onEmailVerificationCodeChange={account.setEmailVerificationCode}
                emailCodeLoading={account.emailCodeLoading}
                emailChangeLoading={account.emailChangeLoading}
                emailCodeSent={account.emailCodeSent}
                emailChangeError={account.emailChangeError}
                onSendEmailChangeCode={account.handleSendEmailChangeCode}
                onValidateEmailChangeCodeRequest={account.validateEmailChangeCodeRequest}
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
                onLogoutAll={account.handleLogoutAll}
              />
            </Suspense>
          )}

          {visitedTabs.includes("apikey") && (
            <Suspense fallback={<SettingsTabLoading value="apikey" />}>
              <SettingsAPIKeyTab active={activeTab === "apikey"} />
            </Suspense>
          )}

          {visitedTabs.includes("audit") && (
            <Suspense fallback={<SettingsTabLoading value="audit" />}>
              <SettingsAuditTab active={activeTab === "audit"} />
            </Suspense>
          )}

          {visitedTabs.includes("about") && (
            <Suspense fallback={<SettingsTabLoading value="about" />}>
              <SettingsAboutTab versionInfo={versionInfo} />
            </Suspense>
          )}
        </Tabs>
      </main>
    </div>
  )
}
