import { Suspense, lazy, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import {
  ArrowLeft,
  BarChart3,
  Database,
  Mail,
  RefreshCw,
  Settings,
  ShieldCheck,
  Users,
} from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAdminPageState } from "@/features/admin/hooks/use-admin-page-state"

import AdminLoadingSkeleton from "./admin-loading-skeleton"

const AdminBackupTab = lazy(() => import("./admin-backup-tab"))
const AdminExchangeRatesTab = lazy(() => import("./admin-exchange-rates-tab"))
const AdminSettingsOIDCTab = lazy(() => import("./admin-settings-oidc-tab"))
const AdminSettingsSMTPTab = lazy(() => import("./admin-settings-smtp-tab"))
const AdminSettingsTab = lazy(() => import("./admin-settings-tab"))
const AdminStatsTab = lazy(() => import("./admin-stats-tab"))
const AdminUsersTab = lazy(() => import("./admin-users-tab"))

type AdminTab = "users" | "settings" | "smtp" | "auth" | "exchange-rates" | "stats" | "backup"

function isAdminTab(value: string): value is AdminTab {
  return value === "users" ||
    value === "settings" ||
    value === "smtp" ||
    value === "auth" ||
    value === "exchange-rates" ||
    value === "stats" ||
    value === "backup"
}

function AdminTabLoading({ value }: { value: AdminTab }) {
  return (
    <TabsContent value={value}>
      <div className="rounded-md border border-dashed px-4 py-8 text-sm text-muted-foreground">
        Loading...
      </div>
    </TabsContent>
  )
}

export default function AdminPage() {
  const { t } = useTranslation()
  const admin = useAdminPageState({ t })
  const { settingsForm } = admin
  const [activeTab, setActiveTab] = useState<AdminTab>("users")
  const [visitedTabs, setVisitedTabs] = useState<AdminTab[]>(["users"])

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-6xl items-center gap-3 px-4">
          <Button variant="ghost" size="icon-sm" asChild>
            <Link to="/">
              <ArrowLeft className="size-4" />
            </Link>
          </Button>
          <h1 className="text-lg font-bold tracking-tight">{t("admin.title")}</h1>
        </div>
      </header>

      <main className="mx-auto max-w-6xl px-4 py-6">
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
            className="space-y-6"
          >
            <TabsList>
              <TabsTrigger value="users" className="gap-2">
                <Users className="size-4" />
                {t("admin.tabs.users")}
              </TabsTrigger>
              <TabsTrigger value="settings" className="gap-2">
                <Settings className="size-4" />
                {t("admin.tabs.settings")}
              </TabsTrigger>
              <TabsTrigger value="smtp" className="gap-2">
                <Mail className="size-4" />
                {t("admin.tabs.email")}
              </TabsTrigger>
              <TabsTrigger value="auth" className="gap-2">
                <ShieldCheck className="size-4" />
                {t("admin.tabs.authentication")}
              </TabsTrigger>
              <TabsTrigger value="exchange-rates" className="gap-2">
                <RefreshCw className="size-4" />
                {t("admin.exchangeRates.title")}
              </TabsTrigger>
              <TabsTrigger value="stats" className="gap-2">
                <BarChart3 className="size-4" />
                {t("admin.tabs.statistics")}
              </TabsTrigger>
              <TabsTrigger value="backup" className="gap-2">
                <Database className="size-4" />
                {t("admin.tabs.backup")}
              </TabsTrigger>
            </TabsList>

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
                  onToggleRole={admin.handleToggleRole}
                  onToggleStatus={admin.handleToggleStatus}
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
                  maxIconFileSize={settingsForm.maxIconFileSize}
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
                  onSave={admin.handleSaveSettings}
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
                  smtpSkipTLSVerify={settingsForm.smtpSkipTLSVerify}
                  onSMTPSkipTLSVerifyChange={(enabled) =>
                    admin.setSettingsField("smtpSkipTLSVerify", enabled)
                  }
                  smtpTesting={admin.smtpTesting}
                  onSMTPTest={admin.handleTestSMTP}
                  onSave={admin.handleSaveSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("auth") && (
              <Suspense fallback={<AdminTabLoading value="auth" />}>
                <AdminSettingsOIDCTab
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
                  onSave={admin.handleSaveSettings}
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
                  onSave={admin.handleSaveSettings}
                />
              </Suspense>
            )}

            {visitedTabs.includes("stats") && (
              <Suspense fallback={<AdminTabLoading value="stats" />}>
                <AdminStatsTab stats={admin.stats} />
              </Suspense>
            )}

            {visitedTabs.includes("backup") && (
              <Suspense fallback={<AdminTabLoading value="backup" />}>
                <AdminBackupTab
                  includeAssetsInBackup={admin.includeAssetsInBackup}
                  restoreFile={admin.restoreFile}
                  onIncludeAssetsInBackupChange={admin.setIncludeAssetsInBackup}
                  onRestoreFileChange={admin.setRestoreFile}
                  restoreConfirmOpen={admin.restoreConfirmOpen}
                  onRestoreConfirmOpenChange={admin.setRestoreConfirmOpen}
                  onDownloadBackup={admin.handleDownloadBackup}
                  onRestore={admin.handleRestore}
                />
              </Suspense>
            )}
          </Tabs>
        )}
      </main>
    </div>
  )
}
