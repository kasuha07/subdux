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
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useAdminPageState } from "@/features/admin/hooks/use-admin-page-state"

import AdminBackupTab from "./admin-backup-tab"
import AdminExchangeRatesTab from "./admin-exchange-rates-tab"
import AdminLoadingSkeleton from "./admin-loading-skeleton"
import AdminSettingsOIDCTab from "./admin-settings-oidc-tab"
import AdminSettingsSMTPTab from "./admin-settings-smtp-tab"
import AdminSettingsTab from "./admin-settings-tab"
import AdminStatsTab from "./admin-stats-tab"
import AdminUsersTab from "./admin-users-tab"

export default function AdminPage() {
  const { t } = useTranslation()
  const admin = useAdminPageState({ t })
  const { settingsForm } = admin

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
          <Tabs defaultValue="users" className="space-y-6">
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

            <AdminSettingsTab
              allowImageUpload={settingsForm.allowImageUpload}
              siteName={settingsForm.siteName}
              onSiteNameChange={(value) => admin.setSettingsField("siteName", value)}
              siteUrl={settingsForm.siteUrl}
              onSiteUrlChange={(value) => admin.setSettingsField("siteUrl", value)}
              registrationEnabled={settingsForm.registrationEnabled}
              registrationEmailVerificationEnabled={settingsForm.registrationEmailVerificationEnabled}
              onRegistrationEnabledChange={(enabled) =>
                admin.setSettingsField("registrationEnabled", enabled)
              }
              onRegistrationEmailVerificationEnabledChange={
                admin.handleRegistrationEmailVerificationChange
              }
              maxIconFileSize={settingsForm.maxIconFileSize}
              onAllowImageUploadChange={(enabled) =>
                admin.setSettingsField("allowImageUpload", enabled)
              }
              onMaxIconFileSizeChange={(value) => admin.setSettingsField("maxIconFileSize", value)}
              onSave={admin.handleSaveSettings}
            />

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

            <AdminStatsTab stats={admin.stats} />

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
          </Tabs>
        )}
      </main>
    </div>
  )
}
