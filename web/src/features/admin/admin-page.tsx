import { Suspense, lazy, useEffect, useLayoutEffect, useRef, useState } from "react"
import { Link } from "react-router"
import { useTranslation } from "react-i18next"
import {
  ArrowLeft,
  Database,
  FileClock,
  Mail,
  RefreshCw,
  ServerCog,
  Settings,
  ShieldCheck,
  Users,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Tooltip } from "@/components/ui/tooltip"
import { useAdminPageState } from "@/features/admin/hooks/use-admin-page-state"
import { cn } from "@/lib/utils"

import AdminLoadingSkeleton, { AdminTabLoading } from "./admin-loading-skeleton"

export { AdminTabLoading }

const AdminBackupTab = lazy(() => import("./admin-backup-tab"))
const AdminAuditTab = lazy(() => import("./admin-audit-tab"))
const AdminBackgroundTasksTab = lazy(() => import("./admin-background-tasks-tab"))
const AdminExchangeRatesTab = lazy(() => import("./admin-exchange-rates-tab"))
const AdminSettingsOIDCTab = lazy(() => import("./admin-settings-oidc-tab"))
const AdminSettingsSMTPTab = lazy(() => import("./admin-settings-smtp-tab"))
const AdminSettingsTab = lazy(() => import("./admin-settings-tab"))
const AdminUsersTab = lazy(() => import("./admin-users-tab"))

type AdminTab = "users" | "settings" | "smtp" | "auth" | "exchange-rates" | "background-tasks" | "audit" | "backup"

function isAdminTab(value: string): value is AdminTab {
  return value === "users" ||
    value === "settings" ||
    value === "smtp" ||
    value === "auth" ||
    value === "exchange-rates" ||
    value === "background-tasks" ||
    value === "audit" ||
    value === "backup"
}

const TAB_DEFINITIONS: {
  value: AdminTab
  labelKey: string
  icon: typeof Users
  loader: () => Promise<unknown>
}[] = [
  { value: "users", labelKey: "admin.tabs.users", icon: Users, loader: () => import("./admin-users-tab") },
  { value: "settings", labelKey: "admin.tabs.settings", icon: Settings, loader: () => import("./admin-settings-tab") },
  { value: "smtp", labelKey: "admin.tabs.email", icon: Mail, loader: () => import("./admin-settings-smtp-tab") },
  { value: "auth", labelKey: "admin.tabs.authentication", icon: ShieldCheck, loader: () => import("./admin-settings-oidc-tab") },
  { value: "exchange-rates", labelKey: "admin.exchangeRates.title", icon: RefreshCw, loader: () => import("./admin-exchange-rates-tab") },
  { value: "background-tasks", labelKey: "admin.tabs.backgroundTasks", icon: ServerCog, loader: () => import("./admin-background-tasks-tab") },
  { value: "audit", labelKey: "admin.tabs.audit", icon: FileClock, loader: () => import("./admin-audit-tab") },
  { value: "backup", labelKey: "admin.tabs.backup", icon: Database, loader: () => import("./admin-backup-tab") },
]

export default function AdminPage() {
  const { t } = useTranslation()
  const admin = useAdminPageState({ t })
  const { settingsForm } = admin
  const [activeTab, setActiveTab] = useState<AdminTab>("users")
  const [visitedTabs, setVisitedTabs] = useState<AdminTab[]>(["users"])

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

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4">
          <Tooltip content={t("common.back")}>
            <Button variant="ghost" size="icon-sm" asChild>
              <Link to="/" aria-label={t("common.back")}>
                <ArrowLeft className="size-4" />
              </Link>
            </Button>
          </Tooltip>
          <h1 className="text-lg font-bold tracking-tight">{t("admin.title")}</h1>
        </div>
      </header>

      <main className="page-stage-enter mx-auto max-w-6xl px-4 py-6">
        {admin.loading ? (
          <AdminLoadingSkeleton />
        ) : (
          <Tabs
            value={activeTab}
            onValueChange={(value) => {
              if (isAdminTab(value)) {
                setActiveTab(value)
                setVisitedTabs((previous) => (
                  previous.includes(value) ? previous : [...previous, value]
                ))
              }
            }}
            className="admin-tabs page-content-enter space-y-6"
          >
            <div className="w-full overflow-x-auto pb-1">
              <TabsList
                ref={tabsListRef}
                className={cn(
                  "relative w-max min-w-max",
                  indicatorStyle && "admin-tabs-list"
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

            {visitedTabs.includes("users") && (
              <Suspense fallback={<AdminTabLoading value="users" />}>
                <AdminUsersTab
                  users={admin.users}
                  createDialogOpen={admin.createDialogOpen}
                  onCreateDialogOpenChange={admin.setCreateDialogOpen}
                  newUsername={admin.newUsername}
                  onNewUsernameChange={admin.setNewUsername}
                  newEmail={admin.newEmail}
                  onNewEmailChange={admin.setNewEmail}
                  newPassword={admin.newPassword}
                  onNewPasswordChange={admin.setNewPassword}
                  newRole={admin.newRole}
                  onNewRoleChange={admin.setNewRole}
                  onCreateUser={admin.handleCreateUser}
                  roleReauthUser={admin.roleReauthUser}
                  onRoleReauthUserChange={admin.setRoleReauthUser}
                  onConfirmToggleRole={admin.handleConfirmToggleRole}
                  onToggleRole={admin.handleToggleRole}
                  onToggleStatus={admin.handleToggleStatus}
                  onDisableUserTOTP={admin.handleDisableUserTOTP}
                  onDisableUserPasskeys={admin.handleDisableUserPasskeys}
                  onDeleteUser={admin.handleDeleteUser}
                />
              </Suspense>
            )}

            {visitedTabs.includes("settings") && (
              <Suspense fallback={<AdminTabLoading value="settings" />}>
                <AdminSettingsTab
                  allowImageUpload={settingsForm.allowImageUpload}
                  iconProxyEnabled={settingsForm.iconProxyEnabled}
                  iconProxyDomainWhitelist={settingsForm.iconProxyDomainWhitelist}
                  siteName={settingsForm.siteName}
                  onSiteNameChange={(value) => admin.setSettingsField("siteName", value)}
                  siteUrl={settingsForm.siteUrl}
                  onSiteUrlChange={(value) => admin.setSettingsField("siteUrl", value)}
                  maxIconFileSize={settingsForm.maxIconFileSize}
                  mcpEnabled={settingsForm.mcpEnabled}
                  auditEnabled={settingsForm.auditEnabled}
                  onAllowImageUploadChange={(enabled) =>
                    admin.setSettingsField("allowImageUpload", enabled)
                  }
                  onIconProxyEnabledChange={(enabled) =>
                    admin.setSettingsField("iconProxyEnabled", enabled)
                  }
                  onIconProxyDomainWhitelistChange={(value) =>
                    admin.setSettingsField("iconProxyDomainWhitelist", value)
                  }
                  onMaxIconFileSizeChange={(value) => admin.setSettingsField("maxIconFileSize", value)}
                  onMCPEnabledChange={(enabled) => admin.setSettingsField("mcpEnabled", enabled)}
                  onAuditEnabledChange={(enabled) => admin.setSettingsField("auditEnabled", enabled)}
                  ssrfProtectionEnabled={settingsForm.ssrfProtectionEnabled}
                  onSSRFProtectionEnabledChange={(enabled) =>
                    admin.setSettingsField("ssrfProtectionEnabled", enabled)
                  }
                  ssrfAllowPrivateIP={settingsForm.ssrfAllowPrivateIP}
                  onSSRFAllowPrivateIPChange={(enabled) =>
                    admin.setSettingsField("ssrfAllowPrivateIP", enabled)
                  }
                  ssrfDomainFilterMode={settingsForm.ssrfDomainFilterMode}
                  onSSRFDomainFilterModeChange={(value) =>
                    admin.setSettingsField("ssrfDomainFilterMode", value)
                  }
                  ssrfDomainFilterList={settingsForm.ssrfDomainFilterList}
                  onSSRFDomainFilterListChange={(value) =>
                    admin.setSettingsField("ssrfDomainFilterList", value)
                  }
                  ssrfIPFilterMode={settingsForm.ssrfIPFilterMode}
                  onSSRFIPFilterModeChange={(value) =>
                    admin.setSettingsField("ssrfIPFilterMode", value)
                  }
                  ssrfIPFilterList={settingsForm.ssrfIPFilterList}
                  onSSRFIPFilterListChange={(value) =>
                    admin.setSettingsField("ssrfIPFilterList", value)
                  }
                  ssrfFilterResolvedIPs={settingsForm.ssrfFilterResolvedIPs}
                  onSSRFFilterResolvedIPsChange={(enabled) =>
                    admin.setSettingsField("ssrfFilterResolvedIPs", enabled)
                  }
                  ssrfTestTarget={admin.ssrfTestTarget}
                  onSSRFTestTargetChange={admin.setSSRFTestTarget}
                  ssrfTestResult={admin.ssrfTestResult}
                  ssrfTesting={admin.ssrfTesting}
                  onSSRFTest={admin.handleTestSSRF}
                  systemProxyEnabled={settingsForm.systemProxyEnabled}
                  onSystemProxyEnabledChange={(enabled) =>
                    admin.setSettingsField("systemProxyEnabled", enabled)
                  }
                  systemProxyType={settingsForm.systemProxyType}
                  onSystemProxyTypeChange={(value) => admin.setSettingsField("systemProxyType", value)}
                  systemProxyUrl={settingsForm.systemProxyUrl}
                  systemProxyUrlConfigured={settingsForm.systemProxyUrlConfigured}
                  onSystemProxyUrlChange={(value) => admin.setSettingsField("systemProxyUrl", value)}
                  onSave={admin.handleSaveGeneralSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("smtp") && (
              <Suspense fallback={<AdminTabLoading value="smtp" />}>
                <AdminSettingsSMTPTab
                  smtpEnabled={settingsForm.smtpEnabled}
                  onSMTPEnabledChange={(enabled) => admin.setSettingsField("smtpEnabled", enabled)}
                  smtpHost={settingsForm.smtpHost}
                  onSMTPHostChange={(value) => admin.setSettingsField("smtpHost", value)}
                  smtpPort={settingsForm.smtpPort}
                  onSMTPPortChange={(value) => admin.setSettingsField("smtpPort", value)}
                  smtpTestRecipient={admin.smtpTestRecipient}
                  onSMTPTestRecipientChange={admin.setSMTPTestRecipient}
                  smtpUsername={settingsForm.smtpUsername}
                  onSMTPUsernameChange={(value) => admin.setSettingsField("smtpUsername", value)}
                  smtpPassword={settingsForm.smtpPassword}
                  smtpPasswordConfigured={settingsForm.smtpPasswordConfigured}
                  onSMTPPasswordChange={(value) => admin.setSettingsField("smtpPassword", value)}
                  smtpFromEmail={settingsForm.smtpFromEmail}
                  onSMTPFromEmailChange={(value) => admin.setSettingsField("smtpFromEmail", value)}
                  smtpFromName={settingsForm.smtpFromName}
                  onSMTPFromNameChange={(value) => admin.setSettingsField("smtpFromName", value)}
                  smtpEncryption={settingsForm.smtpEncryption}
                  onSMTPEncryptionChange={(value) => admin.setSettingsField("smtpEncryption", value)}
                  smtpAuthMethod={settingsForm.smtpAuthMethod}
                  onSMTPAuthMethodChange={(value) => admin.setSettingsField("smtpAuthMethod", value)}
                  smtpHeloName={settingsForm.smtpHeloName}
                  onSMTPHeloNameChange={(value) => admin.setSettingsField("smtpHeloName", value)}
                  smtpTimeoutSeconds={settingsForm.smtpTimeoutSeconds}
                  onSMTPTimeoutSecondsChange={(value) =>
                    admin.setSettingsField("smtpTimeoutSeconds", value)
                  }
                  smtpRateLimitSeconds={settingsForm.smtpRateLimitSeconds}
                  onSMTPRateLimitSecondsChange={(value) =>
                    admin.setSettingsField("smtpRateLimitSeconds", value)
                  }
                  smtpSkipTLSVerify={settingsForm.smtpSkipTLSVerify}
                  onSMTPSkipTLSVerifyChange={(enabled) =>
                    admin.setSettingsField("smtpSkipTLSVerify", enabled)
                  }
                  smtpTesting={admin.smtpTesting}
                  onSMTPTest={admin.handleTestSMTP}
                  onSave={admin.handleSaveSMTPSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("auth") && (
              <Suspense fallback={<AdminTabLoading value="auth" />}>
                <AdminSettingsOIDCTab
                  registrationEnabled={settingsForm.registrationEnabled}
                  registrationEmailVerificationEnabled={settingsForm.registrationEmailVerificationEnabled}
                  emailDomainWhitelist={settingsForm.emailDomainWhitelist}
                  onRegistrationEnabledChange={(enabled) =>
                    admin.setSettingsField("registrationEnabled", enabled)
                  }
                  onEmailDomainWhitelistChange={(value) =>
                    admin.setSettingsField("emailDomainWhitelist", value)
                  }
                  onRegistrationEmailVerificationEnabledChange={
                    admin.handleRegistrationEmailVerificationChange
                  }
                  oidcEnabled={settingsForm.oidcEnabled}
                  onOIDCEnabledChange={(enabled) => admin.setSettingsField("oidcEnabled", enabled)}
                  oidcProviderName={settingsForm.oidcProviderName}
                  onOIDCProviderNameChange={(value) => admin.setSettingsField("oidcProviderName", value)}
                  oidcIssuerURL={settingsForm.oidcIssuerURL}
                  onOIDCIssuerURLChange={(value) => admin.setSettingsField("oidcIssuerURL", value)}
                  oidcClientID={settingsForm.oidcClientID}
                  onOIDCClientIDChange={(value) => admin.setSettingsField("oidcClientID", value)}
                  oidcClientSecret={settingsForm.oidcClientSecret}
                  oidcClientSecretConfigured={settingsForm.oidcClientSecretConfigured}
                  onOIDCClientSecretChange={(value) => admin.setSettingsField("oidcClientSecret", value)}
                  oidcRedirectURL={settingsForm.oidcRedirectURL}
                  onOIDCRedirectURLChange={(value) => admin.setSettingsField("oidcRedirectURL", value)}
                  oidcScopes={settingsForm.oidcScopes}
                  onOIDCScopesChange={(value) => admin.setSettingsField("oidcScopes", value)}
                  oidcAutoCreateUser={settingsForm.oidcAutoCreateUser}
                  onOIDCAutoCreateUserChange={(enabled) =>
                    admin.setSettingsField("oidcAutoCreateUser", enabled)
                  }
                  oidcAuthorizationEndpoint={settingsForm.oidcAuthorizationEndpoint}
                  onOIDCAuthorizationEndpointChange={(value) =>
                    admin.setSettingsField("oidcAuthorizationEndpoint", value)
                  }
                  oidcTokenEndpoint={settingsForm.oidcTokenEndpoint}
                  onOIDCTokenEndpointChange={(value) => admin.setSettingsField("oidcTokenEndpoint", value)}
                  oidcUserinfoEndpoint={settingsForm.oidcUserinfoEndpoint}
                  onOIDCUserinfoEndpointChange={(value) =>
                    admin.setSettingsField("oidcUserinfoEndpoint", value)
                  }
                  oidcAudience={settingsForm.oidcAudience}
                  onOIDCAudienceChange={(value) => admin.setSettingsField("oidcAudience", value)}
                  oidcResource={settingsForm.oidcResource}
                  onOIDCResourceChange={(value) => admin.setSettingsField("oidcResource", value)}
                  oidcExtraAuthParams={settingsForm.oidcExtraAuthParams}
                  onOIDCExtraAuthParamsChange={(value) =>
                    admin.setSettingsField("oidcExtraAuthParams", value)
                  }
                  oidcReauthACRMFA={settingsForm.oidcReauthACRMFA}
                  onOIDCReauthACRMFAChange={(value) =>
                    admin.setSettingsField("oidcReauthACRMFA", value)
                  }
                  oidcReauthACRPhishingResistant={settingsForm.oidcReauthACRPhishingResistant}
                  onOIDCReauthACRPhishingResistantChange={(value) =>
                    admin.setSettingsField("oidcReauthACRPhishingResistant", value)
                  }
                  onSave={admin.handleSaveAuthSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("exchange-rates") && (
              <Suspense fallback={<AdminTabLoading value="exchange-rates" />}>
                <AdminExchangeRatesTab
                  currencyApiKey={settingsForm.currencyApiKey}
                  currencyApiKeyConfigured={settingsForm.currencyApiKeyConfigured}
                  onCurrencyApiKeyChange={(value) => admin.setSettingsField("currencyApiKey", value)}
                  exchangeRateSource={settingsForm.exchangeRateSource}
                  onExchangeRateSourceChange={(value) =>
                    admin.setSettingsField("exchangeRateSource", value)
                  }
                  rateStatus={admin.rateStatus}
                  refreshing={admin.refreshing}
                  onRefresh={admin.handleRefreshRates}
                  onSave={admin.handleSaveExchangeRateSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("background-tasks") && (
              <Suspense fallback={<AdminTabLoading value="background-tasks" />}>
                <AdminBackgroundTasksTab
                  tasks={admin.backgroundTasks}
                  refreshing={admin.backgroundTasksRefreshing}
                  onRefresh={admin.handleRefreshBackgroundTasks}
                />
              </Suspense>
            )}

            {visitedTabs.includes("audit") && (
              <Suspense fallback={<AdminTabLoading value="audit" />}>
                <AdminAuditTab />
              </Suspense>
            )}

            {visitedTabs.includes("backup") && (
              <Suspense fallback={<AdminTabLoading value="backup" />}>
                <AdminBackupTab
                  includeAssetsInBackup={admin.includeAssetsInBackup}
                  downloadPassword={admin.downloadPassword}
                  onDownloadPasswordChange={admin.setDownloadPassword}
                  restoreFile={admin.restoreFile}
                  restoreEncrypted={admin.restoreEncrypted}
                  restorePassword={admin.restorePassword}
                  onRestorePasswordChange={admin.setRestorePassword}
                  onIncludeAssetsInBackupChange={admin.setIncludeAssetsInBackup}
                  onRestoreFileChange={admin.setRestoreFile}
                  restoreConfirmOpen={admin.restoreConfirmOpen}
                  onRestoreConfirmOpenChange={admin.setRestoreConfirmOpen}
                  onDownloadBackup={admin.handleDownloadBackup}
                  onRestore={admin.handleRestore}
                  onValidateRestoreInputs={admin.handleValidateRestoreInputs}
                  backupRuns={admin.backupRuns}
                  backupRunsRefreshing={admin.backupRunsRefreshing}
                  onRefreshBackupRuns={admin.handleRefreshBackupRuns}
                  destinations={admin.destinations}
                  destinationsRefreshing={admin.destinationsRefreshing}
                  runningDestinationId={admin.runningDestinationId}
                  onRefreshDestinations={admin.handleRefreshDestinations}
                  onCreateDestination={admin.handleCreateDestination}
                  onUpdateDestination={admin.handleUpdateDestination}
                  onDeleteDestination={admin.handleDeleteDestination}
                  onRunDestination={admin.handleRunDestinationBackup}
                  onTestDestination={admin.handleTestDestination}
                  onLoadDestinationBackups={admin.handleListDestinationBackups}
                  onRestoreFromDestination={admin.handleRestoreFromDestination}
                  onTestDestinationConfig={admin.handleTestDestinationConfig}
                />
              </Suspense>
            )}
          </Tabs>
        )}
      </main>
    </div>
  )
}
