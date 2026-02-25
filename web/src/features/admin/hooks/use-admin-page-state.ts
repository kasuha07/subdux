import { useCallback, useEffect, useState } from "react"

import { updateSiteTitle } from "@/hooks/useSiteSettings"
import { api, getUser } from "@/lib/api"
import { toast } from "sonner"
import type {
  AdminStats,
  AdminUser,
  ExchangeRateStatus,
  SystemSettings,
  UpdateSettingsInput,
} from "@/types"

interface AdminSettingsFormState {
  allowImageUpload: boolean
  currencyApiKey: string
  currencyApiKeyConfigured: boolean
  emailDomainWhitelist: string
  exchangeRateSource: string
  maxIconFileSize: number
  oidcAudience: string
  oidcAuthorizationEndpoint: string
  oidcAutoCreateUser: boolean
  oidcClientID: string
  oidcClientSecret: string
  oidcClientSecretConfigured: boolean
  oidcEnabled: boolean
  oidcExtraAuthParams: string
  oidcIssuerURL: string
  oidcProviderName: string
  oidcRedirectURL: string
  oidcResource: string
  oidcScopes: string
  oidcTokenEndpoint: string
  oidcUserinfoEndpoint: string
  registrationEmailVerificationEnabled: boolean
  registrationEnabled: boolean
  siteName: string
  siteUrl: string
  smtpAuthMethod: string
  smtpEnabled: boolean
  smtpEncryption: string
  smtpFromEmail: string
  smtpFromName: string
  smtpHeloName: string
  smtpHost: string
  smtpPassword: string
  smtpPasswordConfigured: boolean
  smtpPort: number
  smtpSkipTLSVerify: boolean
  smtpTimeoutSeconds: number
  smtpUsername: string
}

interface UseAdminPageStateOptions {
  t: (key: string) => string
}

interface UseAdminPageStateResult {
  createDialogOpen: boolean
  handleCreateUser: () => Promise<void>
  handleDeleteUser: (id: number) => Promise<void>
  handleDownloadBackup: () => Promise<void>
  handleRefreshRates: () => Promise<void>
  handleRegistrationEmailVerificationChange: (enabled: boolean) => void
  handleRestore: () => Promise<void>
  handleSaveSettings: () => Promise<void>
  handleTestSMTP: () => Promise<void>
  handleToggleRole: (user: AdminUser) => Promise<void>
  handleToggleStatus: (user: AdminUser) => Promise<void>
  includeAssetsInBackup: boolean
  loading: boolean
  newEmail: string
  newPassword: string
  newRole: "user" | "admin"
  newUsername: string
  rateStatus: ExchangeRateStatus | null
  refreshing: boolean
  restoreConfirmOpen: boolean
  restoreFile: File | null
  setCreateDialogOpen: (open: boolean) => void
  setIncludeAssetsInBackup: (value: boolean) => void
  setNewEmail: (value: string) => void
  setNewPassword: (value: string) => void
  setNewRole: (value: "user" | "admin") => void
  setNewUsername: (value: string) => void
  setRestoreConfirmOpen: (value: boolean) => void
  setRestoreFile: (file: File | null) => void
  setSettingsField: <K extends keyof AdminSettingsFormState>(
    key: K,
    value: AdminSettingsFormState[K]
  ) => void
  setSMTPTestRecipient: (value: string) => void
  settingsForm: AdminSettingsFormState
  smtpTestRecipient: string
  smtpTesting: boolean
  stats: AdminStats | null
  users: AdminUser[]
}

function createSettingsForm(settings?: SystemSettings): AdminSettingsFormState {
  return {
    allowImageUpload: settings?.allow_image_upload ?? true,
    currencyApiKey: "",
    currencyApiKeyConfigured: settings?.currencyapi_key_configured ?? false,
    emailDomainWhitelist: settings?.email_domain_whitelist || "",
    exchangeRateSource: settings?.exchange_rate_source || "auto",
    maxIconFileSize: settings?.max_icon_file_size
      ? Math.round(settings.max_icon_file_size / 1024)
      : 64,
    oidcAudience: settings?.oidc_audience || "",
    oidcAuthorizationEndpoint: settings?.oidc_authorization_endpoint || "",
    oidcAutoCreateUser: settings?.oidc_auto_create_user ?? false,
    oidcClientID: settings?.oidc_client_id || "",
    oidcClientSecret: "",
    oidcClientSecretConfigured: settings?.oidc_client_secret_configured ?? false,
    oidcEnabled: settings?.oidc_enabled ?? false,
    oidcExtraAuthParams: settings?.oidc_extra_auth_params || "",
    oidcIssuerURL: settings?.oidc_issuer_url || "",
    oidcProviderName: settings?.oidc_provider_name || "OIDC",
    oidcRedirectURL: settings?.oidc_redirect_url || "",
    oidcResource: settings?.oidc_resource || "",
    oidcScopes: settings?.oidc_scopes || "openid profile email",
    oidcTokenEndpoint: settings?.oidc_token_endpoint || "",
    oidcUserinfoEndpoint: settings?.oidc_userinfo_endpoint || "",
    registrationEmailVerificationEnabled:
      settings?.registration_email_verification_enabled ?? false,
    registrationEnabled: settings?.registration_enabled ?? true,
    siteName: settings?.site_name || "Subdux",
    siteUrl: settings?.site_url || "",
    smtpAuthMethod: settings?.smtp_auth_method || "auto",
    smtpEnabled: settings?.smtp_enabled ?? false,
    smtpEncryption: settings?.smtp_encryption || "starttls",
    smtpFromEmail: settings?.smtp_from_email || "",
    smtpFromName: settings?.smtp_from_name || "",
    smtpHeloName: settings?.smtp_helo_name || "",
    smtpHost: settings?.smtp_host || "",
    smtpPassword: "",
    smtpPasswordConfigured: settings?.smtp_password_configured ?? false,
    smtpPort: settings?.smtp_port || 587,
    smtpSkipTLSVerify: settings?.smtp_skip_tls_verify ?? false,
    smtpTimeoutSeconds: settings?.smtp_timeout_seconds || 10,
    smtpUsername: settings?.smtp_username || "",
  }
}

function parseFilenameFromContentDisposition(contentDisposition: string | null): string | null {
  if (!contentDisposition) {
    return null
  }

  const match = contentDisposition.match(/filename="?([^"]+)"?/i)
  if (!match || !match[1]) {
    return null
  }

  return match[1]
}

function hasSMTPConfigForRegistrationVerification(form: AdminSettingsFormState): boolean {
  if (!form.smtpEnabled) {
    return false
  }

  const host = form.smtpHost.trim()
  const fromEmail = form.smtpFromEmail.trim()
  const username = form.smtpUsername.trim()
  const hasPassword = form.smtpPassword.trim() !== "" || form.smtpPasswordConfigured

  if (host === "" || fromEmail === "") {
    return false
  }
  if (!Number.isInteger(form.smtpPort) || form.smtpPort < 1 || form.smtpPort > 65535) {
    return false
  }

  const authMethod = form.smtpAuthMethod.trim().toLowerCase()
  if (!["auto", "plain", "login", "cram_md5", "none"].includes(authMethod)) {
    return false
  }
  if (authMethod !== "auto" && authMethod !== "none" && (username === "" || !hasPassword)) {
    return false
  }

  return true
}

export function useAdminPageState({ t }: UseAdminPageStateOptions): UseAdminPageStateResult {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [settingsForm, setSettingsForm] = useState<AdminSettingsFormState>(() =>
    createSettingsForm()
  )

  const [includeAssetsInBackup, setIncludeAssetsInBackup] = useState(false)
  const [restoreFile, setRestoreFile] = useState<File | null>(null)
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false)

  const [rateStatus, setRateStatus] = useState<ExchangeRateStatus | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [smtpTestRecipient, setSMTPTestRecipient] = useState(() => getUser()?.email ?? "")
  const [smtpTesting, setSMTPTesting] = useState(false)

  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [newUsername, setNewUsername] = useState("")
  const [newEmail, setNewEmail] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [newRole, setNewRole] = useState<"user" | "admin">("user")

  const setSettingsField = useCallback(
    <K extends keyof AdminSettingsFormState>(key: K, value: AdminSettingsFormState[K]) => {
      setSettingsForm((prev) => ({
        ...prev,
        [key]: value,
      }))
    },
    []
  )

  useEffect(() => {
    Promise.all([
      api.get<AdminUser[]>("/admin/users"),
      api.get<SystemSettings>("/admin/settings"),
      api.get<AdminStats>("/admin/stats"),
      api.get<ExchangeRateStatus>("/admin/exchange-rates/status"),
    ])
      .then(([usersData, settingsData, statsData, rateStatusData]) => {
        setUsers(usersData || [])
        setSettingsForm(createSettingsForm(settingsData))
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
    if (!confirm(t("admin.users.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/admin/users/${id}`)
      setUsers((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("admin.users.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleCreateUser() {
    if (!newUsername || !newEmail || !newPassword) {
      return
    }
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
        registration_enabled: settingsForm.registrationEnabled,
        registration_email_verification_enabled: settingsForm.registrationEmailVerificationEnabled,
        email_domain_whitelist: settingsForm.emailDomainWhitelist,
        site_name: settingsForm.siteName,
        site_url: settingsForm.siteUrl,
        exchange_rate_source: settingsForm.exchangeRateSource,
        allow_image_upload: settingsForm.allowImageUpload,
        max_icon_file_size: settingsForm.maxIconFileSize * 1024,
        smtp_enabled: settingsForm.smtpEnabled,
        smtp_host: settingsForm.smtpHost,
        smtp_port: settingsForm.smtpPort,
        smtp_username: settingsForm.smtpUsername,
        smtp_from_email: settingsForm.smtpFromEmail,
        smtp_from_name: settingsForm.smtpFromName,
        smtp_encryption: settingsForm.smtpEncryption,
        smtp_auth_method: settingsForm.smtpAuthMethod,
        smtp_helo_name: settingsForm.smtpHeloName,
        smtp_timeout_seconds: settingsForm.smtpTimeoutSeconds,
        smtp_skip_tls_verify: settingsForm.smtpSkipTLSVerify,
        oidc_enabled: settingsForm.oidcEnabled,
        oidc_provider_name: settingsForm.oidcProviderName,
        oidc_issuer_url: settingsForm.oidcIssuerURL,
        oidc_client_id: settingsForm.oidcClientID,
        oidc_redirect_url: settingsForm.oidcRedirectURL,
        oidc_scopes: settingsForm.oidcScopes,
        oidc_auto_create_user: settingsForm.oidcAutoCreateUser,
        oidc_authorization_endpoint: settingsForm.oidcAuthorizationEndpoint,
        oidc_token_endpoint: settingsForm.oidcTokenEndpoint,
        oidc_userinfo_endpoint: settingsForm.oidcUserinfoEndpoint,
        oidc_audience: settingsForm.oidcAudience,
        oidc_resource: settingsForm.oidcResource,
        oidc_extra_auth_params: settingsForm.oidcExtraAuthParams,
      }

      if (settingsForm.oidcClientSecret.trim()) {
        payload.oidc_client_secret = settingsForm.oidcClientSecret.trim()
      }
      if (settingsForm.smtpPassword.trim()) {
        payload.smtp_password = settingsForm.smtpPassword.trim()
      }
      if (settingsForm.currencyApiKey.trim()) {
        payload.currencyapi_key = settingsForm.currencyApiKey.trim()
      }

      await api.put("/admin/settings", payload)
      const fresh = await api.get<SystemSettings>("/admin/settings")
      setSettingsForm(createSettingsForm(fresh))
      updateSiteTitle(fresh.site_name)
      toast.success(t("admin.settings.saveSuccess"))
    } catch {
      void 0
    }
  }

  function handleRegistrationEmailVerificationChange(enabled: boolean) {
    if (!enabled) {
      setSettingsField("registrationEmailVerificationEnabled", false)
      return
    }

    if (!hasSMTPConfigForRegistrationVerification(settingsForm)) {
      toast.error(t("admin.settings.registrationEmailVerificationSmtpWarning"))
      return
    }

    setSettingsField("registrationEmailVerificationEnabled", true)
  }

  async function handleTestSMTP() {
    setSMTPTesting(true)
    try {
      await api.post<{ message: string }>("/admin/settings/smtp/test", {
        recipient_email: smtpTestRecipient.trim(),
      })
      toast.success(t("admin.settings.smtpTestSuccess"))
    } catch {
      void 0
    } finally {
      setSMTPTesting(false)
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
      const params = new URLSearchParams()
      if (includeAssetsInBackup) {
        params.set("include_assets", "true")
      }
      const endpoint = params.size > 0 ? `/api/admin/backup?${params.toString()}` : "/api/admin/backup"

      const res = await fetch(endpoint, {
        headers: { Authorization: `Bearer ${token}` },
      })
      if (!res.ok) {
        throw new Error()
      }

      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const anchor = document.createElement("a")
      anchor.href = url
      const filename =
        parseFilenameFromContentDisposition(res.headers.get("content-disposition")) ??
        `subdux-backup-${new Date().toISOString().split("T")[0]}${includeAssetsInBackup ? ".zip" : ".db"}`
      anchor.download = filename
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
    if (!restoreFile) {
      return
    }

    const formData = new FormData()
    formData.append("backup", restoreFile)

    try {
      const token = localStorage.getItem("token")
      const res = await fetch("/api/admin/restore", {
        method: "POST",
        headers: { Authorization: `Bearer ${token}` },
        body: formData,
      })
      if (!res.ok) {
        throw new Error()
      }

      setRestoreConfirmOpen(false)
      toast.success(t("admin.backup.restoreSuccess"))
    } catch {
      toast.error(t("admin.backup.restoreFailed"))
    }
  }

  return {
    createDialogOpen,
    handleCreateUser,
    handleDeleteUser,
    handleDownloadBackup,
    handleRefreshRates,
    handleRegistrationEmailVerificationChange,
    handleRestore,
    handleSaveSettings,
    handleTestSMTP,
    handleToggleRole,
    handleToggleStatus,
    includeAssetsInBackup,
    loading,
    newEmail,
    newPassword,
    newRole,
    newUsername,
    rateStatus,
    refreshing,
    restoreConfirmOpen,
    restoreFile,
    setCreateDialogOpen,
    setIncludeAssetsInBackup,
    setNewEmail,
    setNewPassword,
    setNewRole,
    setNewUsername,
    setRestoreConfirmOpen,
    setRestoreFile,
    setSettingsField,
    setSMTPTestRecipient,
    settingsForm,
    smtpTestRecipient,
    smtpTesting,
    stats,
    users,
  }
}
