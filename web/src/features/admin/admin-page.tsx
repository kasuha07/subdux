import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { useTranslation } from "react-i18next"
import { ArrowLeft, BarChart3, Database, RefreshCw, Settings, Users } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { updateSiteTitle } from "@/hooks/useSiteSettings"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type {
  AdminStats,
  AdminUser,
  ExchangeRateStatus,
  SystemSettings,
  UpdateSettingsInput,
} from "@/types"

import AdminBackupTab from "./admin-backup-tab"
import AdminExchangeRatesTab from "./admin-exchange-rates-tab"
import AdminLoadingSkeleton from "./admin-loading-skeleton"
import AdminSettingsTab from "./admin-settings-tab"
import AdminStatsTab from "./admin-stats-tab"
import AdminUsersTab from "./admin-users-tab"

export default function AdminPage() {
  const { t } = useTranslation()

  const [users, setUsers] = useState<AdminUser[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [siteName, setSiteName] = useState("")
  const [siteUrl, setSiteUrl] = useState("")
  const [registrationEnabled, setRegistrationEnabled] = useState(true)
  const [restoreFile, setRestoreFile] = useState<File | null>(null)
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false)

  const [rateStatus, setRateStatus] = useState<ExchangeRateStatus | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [currencyApiKey, setCurrencyApiKey] = useState("")
  const [exchangeRateSource, setExchangeRateSource] = useState("auto")
  const [maxIconFileSize, setMaxIconFileSize] = useState<number>(64)
  const [oidcEnabled, setOIDCEnabled] = useState(false)
  const [oidcProviderName, setOIDCProviderName] = useState("OIDC")
  const [oidcIssuerURL, setOIDCIssuerURL] = useState("")
  const [oidcClientID, setOIDCClientID] = useState("")
  const [oidcClientSecret, setOIDCClientSecret] = useState("")
  const [oidcClientSecretConfigured, setOIDCClientSecretConfigured] = useState(false)
  const [oidcRedirectURL, setOIDCRedirectURL] = useState("")
  const [oidcScopes, setOIDCScopes] = useState("openid profile email")
  const [oidcAutoCreateUser, setOIDCAutoCreateUser] = useState(false)
  const [oidcAuthorizationEndpoint, setOIDCAuthorizationEndpoint] = useState("")
  const [oidcTokenEndpoint, setOIDCTokenEndpoint] = useState("")
  const [oidcUserinfoEndpoint, setOIDCUserinfoEndpoint] = useState("")
  const [oidcAudience, setOIDCAudience] = useState("")
  const [oidcResource, setOIDCResource] = useState("")
  const [oidcExtraAuthParams, setOIDCExtraAuthParams] = useState("")

  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newUsername, setNewUsername] = useState("")
  const [newEmail, setNewEmail] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [newRole, setNewRole] = useState<"user" | "admin">("user")

  useEffect(() => {
    Promise.all([
      api.get<AdminUser[]>("/admin/users"),
      api.get<SystemSettings>("/admin/settings"),
      api.get<AdminStats>("/admin/stats"),
      api.get<ExchangeRateStatus>("/admin/exchange-rates/status"),
    ])
      .then(([usersData, settingsData, statsData, rateStatusData]) => {
        setUsers(usersData || [])
        setSiteName(settingsData?.site_name || "Subdux")
        setSiteUrl(settingsData?.site_url || "")
        setRegistrationEnabled(settingsData?.registration_enabled ?? true)
        setCurrencyApiKey(settingsData?.currencyapi_key || "")
        setExchangeRateSource(settingsData?.exchange_rate_source || "auto")
        setMaxIconFileSize(
          settingsData?.max_icon_file_size
            ? Math.round(settingsData.max_icon_file_size / 1024)
            : 64
        )
        setOIDCEnabled(settingsData?.oidc_enabled ?? false)
        setOIDCProviderName(settingsData?.oidc_provider_name || "OIDC")
        setOIDCIssuerURL(settingsData?.oidc_issuer_url || "")
        setOIDCClientID(settingsData?.oidc_client_id || "")
        setOIDCClientSecret("")
        setOIDCClientSecretConfigured(settingsData?.oidc_client_secret_configured ?? false)
        setOIDCRedirectURL(settingsData?.oidc_redirect_url || "")
        setOIDCScopes(settingsData?.oidc_scopes || "openid profile email")
        setOIDCAutoCreateUser(settingsData?.oidc_auto_create_user ?? false)
        setOIDCAuthorizationEndpoint(settingsData?.oidc_authorization_endpoint || "")
        setOIDCTokenEndpoint(settingsData?.oidc_token_endpoint || "")
        setOIDCUserinfoEndpoint(settingsData?.oidc_userinfo_endpoint || "")
        setOIDCAudience(settingsData?.oidc_audience || "")
        setOIDCResource(settingsData?.oidc_resource || "")
        setOIDCExtraAuthParams(settingsData?.oidc_extra_auth_params || "")
        setStats(statsData)
        setRateStatus(rateStatusData)
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  async function handleToggleRole(user: AdminUser) {
    const nextRole = user.role === "admin" ? "user" : "admin"
    try {
      await api.put(`/admin/users/${user.id}/role`, { role: nextRole })
      setUsers((prev) => prev.map((item) => (item.id === user.id ? { ...item, role: nextRole } : item)))
      toast.success(t("admin.users.roleUpdated"))
    } catch {
      void 0
    }
  }

  async function handleToggleStatus(user: AdminUser) {
    const nextStatus = user.status === "active" ? "disabled" : "active"
    try {
      await api.put(`/admin/users/${user.id}/status`, { status: nextStatus })
      setUsers((prev) =>
        prev.map((item) => (item.id === user.id ? { ...item, status: nextStatus } : item))
      )
      toast.success(t("admin.users.statusUpdated"))
    } catch {
      void 0
    }
  }

  async function handleDeleteUser(id: number) {
    if (!confirm(t("admin.users.deleteConfirm"))) return
    try {
      await api.delete(`/admin/users/${id}`)
      setUsers((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("admin.users.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleCreateUser() {
    if (!newUsername || !newEmail || !newPassword) return
    if (newPassword.length < 6) {
      toast.error(t("admin.users.passwordTooShort"))
      return
    }

    try {
      const user = await api.post<AdminUser>("/admin/users", {
        username: newUsername,
        email: newEmail,
        password: newPassword,
        role: newRole,
      })
      setUsers((prev) => [...prev, user])
      setCreateDialogOpen(false)
      setNewUsername("")
      setNewEmail("")
      setNewPassword("")
      setNewRole("user")
      toast.success(t("admin.users.createSuccess"))
    } catch {
      void 0
    }
  }

  async function handleSaveSettings() {
    try {
      const payload: UpdateSettingsInput = {
        registration_enabled: registrationEnabled,
        site_name: siteName,
        site_url: siteUrl,
        currencyapi_key: currencyApiKey,
        exchange_rate_source: exchangeRateSource,
        max_icon_file_size: maxIconFileSize * 1024,
        oidc_enabled: oidcEnabled,
        oidc_provider_name: oidcProviderName,
        oidc_issuer_url: oidcIssuerURL,
        oidc_client_id: oidcClientID,
        oidc_redirect_url: oidcRedirectURL,
        oidc_scopes: oidcScopes,
        oidc_auto_create_user: oidcAutoCreateUser,
        oidc_authorization_endpoint: oidcAuthorizationEndpoint,
        oidc_token_endpoint: oidcTokenEndpoint,
        oidc_userinfo_endpoint: oidcUserinfoEndpoint,
        oidc_audience: oidcAudience,
        oidc_resource: oidcResource,
        oidc_extra_auth_params: oidcExtraAuthParams,
      }
      if (oidcClientSecret.trim()) {
        payload.oidc_client_secret = oidcClientSecret.trim()
      }

      await api.put("/admin/settings", payload)
      const fresh = await api.get<SystemSettings>("/admin/settings")
      setSiteName(fresh.site_name)
      setSiteUrl(fresh.site_url)
      setRegistrationEnabled(fresh.registration_enabled)
      setCurrencyApiKey(fresh.currencyapi_key)
      setExchangeRateSource(fresh.exchange_rate_source)
      setMaxIconFileSize(fresh.max_icon_file_size ? Math.round(fresh.max_icon_file_size / 1024) : 64)
      setOIDCEnabled(fresh.oidc_enabled ?? false)
      setOIDCProviderName(fresh.oidc_provider_name || "OIDC")
      setOIDCIssuerURL(fresh.oidc_issuer_url || "")
      setOIDCClientID(fresh.oidc_client_id || "")
      setOIDCClientSecret("")
      setOIDCClientSecretConfigured(fresh.oidc_client_secret_configured ?? false)
      setOIDCRedirectURL(fresh.oidc_redirect_url || "")
      setOIDCScopes(fresh.oidc_scopes || "openid profile email")
      setOIDCAutoCreateUser(fresh.oidc_auto_create_user ?? false)
      setOIDCAuthorizationEndpoint(fresh.oidc_authorization_endpoint || "")
      setOIDCTokenEndpoint(fresh.oidc_token_endpoint || "")
      setOIDCUserinfoEndpoint(fresh.oidc_userinfo_endpoint || "")
      setOIDCAudience(fresh.oidc_audience || "")
      setOIDCResource(fresh.oidc_resource || "")
      setOIDCExtraAuthParams(fresh.oidc_extra_auth_params || "")
      updateSiteTitle(fresh.site_name)
      toast.success(t("admin.settings.saveSuccess"))
    } catch {
      void 0
    }
  }

  async function handleRefreshRates() {
    setRefreshing(true)
    try {
      await api.post("/admin/exchange-rates/refresh", {})
      const status = await api.get<ExchangeRateStatus>("/admin/exchange-rates/status")
      setRateStatus(status)
      toast.success(t("admin.exchangeRates.refreshSuccess"))
    } catch {
      toast.error(t("admin.exchangeRates.refreshFailed"))
    } finally {
      setRefreshing(false)
    }
  }

  async function handleDownloadBackup() {
    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/admin/backup", {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) throw new Error()

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const anchor = document.createElement("a")
      anchor.href = url
      anchor.download = `subdux-backup-${new Date().toISOString().split("T")[0]}.db`
      document.body.appendChild(anchor)
      anchor.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(anchor)
      toast.success(t("admin.backup.downloadSuccess"))
    } catch {
      toast.error(t("admin.backup.downloadFailed"))
    }
  }

  async function handleRestore() {
    if (!restoreFile) return

    const formData = new FormData()
    formData.append("backup", restoreFile)

    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/admin/restore", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      })
      if (!res.ok) throw new Error()

      setRestoreConfirmOpen(false)
      toast.success(t("admin.backup.restoreSuccess"))
    } catch {
      toast.error(t("admin.backup.restoreFailed"))
    }
  }

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
        {loading ? (
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
              users={users}
              createDialogOpen={createDialogOpen}
              onCreateDialogOpenChange={setCreateDialogOpen}
              newUsername={newUsername}
              onNewUsernameChange={setNewUsername}
              newEmail={newEmail}
              onNewEmailChange={setNewEmail}
              newPassword={newPassword}
              onNewPasswordChange={setNewPassword}
              newRole={newRole}
              onNewRoleChange={setNewRole}
              onCreateUser={handleCreateUser}
              onToggleRole={handleToggleRole}
              onToggleStatus={handleToggleStatus}
              onDeleteUser={handleDeleteUser}
            />

            <AdminSettingsTab
              siteName={siteName}
              onSiteNameChange={setSiteName}
              siteUrl={siteUrl}
              onSiteUrlChange={setSiteUrl}
              registrationEnabled={registrationEnabled}
              onRegistrationEnabledChange={setRegistrationEnabled}
              maxIconFileSize={maxIconFileSize}
              onMaxIconFileSizeChange={setMaxIconFileSize}
              oidcEnabled={oidcEnabled}
              onOIDCEnabledChange={setOIDCEnabled}
              oidcProviderName={oidcProviderName}
              onOIDCProviderNameChange={setOIDCProviderName}
              oidcIssuerURL={oidcIssuerURL}
              onOIDCIssuerURLChange={setOIDCIssuerURL}
              oidcClientID={oidcClientID}
              onOIDCClientIDChange={setOIDCClientID}
              oidcClientSecret={oidcClientSecret}
              oidcClientSecretConfigured={oidcClientSecretConfigured}
              onOIDCClientSecretChange={setOIDCClientSecret}
              oidcRedirectURL={oidcRedirectURL}
              onOIDCRedirectURLChange={setOIDCRedirectURL}
              oidcScopes={oidcScopes}
              onOIDCScopesChange={setOIDCScopes}
              oidcAutoCreateUser={oidcAutoCreateUser}
              onOIDCAutoCreateUserChange={setOIDCAutoCreateUser}
              oidcAuthorizationEndpoint={oidcAuthorizationEndpoint}
              onOIDCAuthorizationEndpointChange={setOIDCAuthorizationEndpoint}
              oidcTokenEndpoint={oidcTokenEndpoint}
              onOIDCTokenEndpointChange={setOIDCTokenEndpoint}
              oidcUserinfoEndpoint={oidcUserinfoEndpoint}
              onOIDCUserinfoEndpointChange={setOIDCUserinfoEndpoint}
              oidcAudience={oidcAudience}
              onOIDCAudienceChange={setOIDCAudience}
              oidcResource={oidcResource}
              onOIDCResourceChange={setOIDCResource}
              oidcExtraAuthParams={oidcExtraAuthParams}
              onOIDCExtraAuthParamsChange={setOIDCExtraAuthParams}
              onSave={handleSaveSettings}
            />

            <AdminExchangeRatesTab
              currencyApiKey={currencyApiKey}
              onCurrencyApiKeyChange={setCurrencyApiKey}
              exchangeRateSource={exchangeRateSource}
              onExchangeRateSourceChange={setExchangeRateSource}
              rateStatus={rateStatus}
              refreshing={refreshing}
              onRefresh={handleRefreshRates}
              onSave={handleSaveSettings}
            />

            <AdminStatsTab stats={stats} />

            <AdminBackupTab
              restoreFile={restoreFile}
              onRestoreFileChange={setRestoreFile}
              restoreConfirmOpen={restoreConfirmOpen}
              onRestoreConfirmOpenChange={setRestoreConfirmOpen}
              onDownloadBackup={handleDownloadBackup}
              onRestore={handleRestore}
            />
          </Tabs>
        )}
      </main>
    </div>
  )
}
