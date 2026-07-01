import { useCallback, useEffect, useState } from "react"

import { updateSiteTitle } from "@/hooks/useSiteSettings"
import { api, getUser, localizeBackendError } from "@/lib/api"
import {
  detectZipEncryption,
  verifyZipPassword,
  type EncryptedZipEntry,
} from "@/lib/zip-encryption"
import { toast } from "sonner"
import type {
  AdminUser,
  BackgroundTask,
  ExchangeRateStatus,
  LocalBackupInfo,
  LocalBackupList,
  SSRFTestResult,
  SystemSettings,
  UpdateSettingsInput,
} from "@/types"

// Best-effort extraction of a JSON `{ "error": string }` message from a raw
// Response returned by api.fetch. Returns undefined when the body is absent,
// not JSON, or has no usable error string, letting callers fall back to a
// generic message.
async function readErrorMessage(res: Response): Promise<string | undefined> {
  try {
    const data = (await res.clone().json()) as { error?: unknown }
    if (typeof data?.error === "string" && data.error.trim() !== "") {
      return localizeBackendError(data.error)
    }
  } catch {
    void 0
  }
  return undefined
}

interface AdminSettingsFormState {
  allowImageUpload: boolean
  backupScheduleEnabled: boolean
  backupTimeOfDay: string
  backupIncludeAssets: boolean
  backupEncryptEnabled: boolean
  backupEncryptionPassword: string
  backupEncryptionPasswordConfigured: boolean
  backupLocalDir: string
  backupRetentionCount: number
  currencyApiKey: string
  currencyApiKeyConfigured: boolean
  emailDomainWhitelist: string
  exchangeRateSource: string
  iconProxyDomainWhitelist: string
  iconProxyEnabled: boolean
  maxIconFileSize: number
  mcpEnabled: boolean
  auditEnabled: boolean
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
  smtpRateLimitSeconds: number
  smtpSkipTLSVerify: boolean
  smtpTimeoutSeconds: number
  smtpUsername: string
  ssrfAllowPrivateIP: boolean
  ssrfDomainFilterList: string
  ssrfDomainFilterMode: string
  ssrfFilterResolvedIPs: boolean
  ssrfIPFilterList: string
  ssrfIPFilterMode: string
  ssrfProtectionEnabled: boolean
  systemProxyEnabled: boolean
  systemProxyType: string
  systemProxyUrl: string
  systemProxyUrlConfigured: boolean
}

interface UseAdminPageStateOptions {
  t: (key: string) => string
}

interface BackupStatus {
  lastRunAt: string
  lastStatus: string
  lastError: string
}

interface UseAdminPageStateResult {
  backgroundTasks: BackgroundTask[]
  backgroundTasksRefreshing: boolean
  backupStatus: BackupStatus
  createDialogOpen: boolean
  downloadPassword: string
  handleCreateUser: () => Promise<void>
  handleRefreshBackgroundTasks: () => Promise<void>
  handleRefreshLocalBackups: () => Promise<void>
  handleDeleteUser: (id: number) => Promise<void>
  handleDownloadBackup: (reauthTicket: string) => Promise<boolean>
  handleRefreshRates: () => Promise<void>
  handleRegistrationEmailVerificationChange: (enabled: boolean) => void
  handleRestore: (reauthTicket: string) => Promise<boolean>
  handleValidateRestoreInputs: () => Promise<boolean>
  handleRunBackupNow: () => Promise<void>
  handleSaveSettings: () => Promise<void>
  handleTestSSRF: () => Promise<void>
  handleTestSMTP: () => Promise<void>
  handleToggleRole: (user: AdminUser) => Promise<void>
  handleToggleStatus: (user: AdminUser) => Promise<void>
  includeAssetsInBackup: boolean
  loading: boolean
  localBackupDir: string
  localBackups: LocalBackupInfo[]
  localBackupsRefreshing: boolean
  newEmail: string
  newPassword: string
  newRole: "user" | "admin"
  newUsername: string
  rateStatus: ExchangeRateStatus | null
  refreshing: boolean
  restoreConfirmOpen: boolean
  restoreEncrypted: boolean
  restoreFile: File | null
  restorePassword: string
  runningBackup: boolean
  setCreateDialogOpen: (open: boolean) => void
  setDownloadPassword: (value: string) => void
  setIncludeAssetsInBackup: (value: boolean) => void
  setNewEmail: (value: string) => void
  setNewPassword: (value: string) => void
  setNewRole: (value: "user" | "admin") => void
  setNewUsername: (value: string) => void
  setRestoreConfirmOpen: (value: boolean) => void
  setRestoreFile: (file: File | null) => void
  setRestorePassword: (value: string) => void
  setSSRFTestTarget: (value: string) => void
  setSettingsField: <K extends keyof AdminSettingsFormState>(
    key: K,
    value: AdminSettingsFormState[K]
  ) => void
  setSMTPTestRecipient: (value: string) => void
  settingsForm: AdminSettingsFormState
  smtpTestRecipient: string
  smtpTesting: boolean
  ssrfTestResult: SSRFTestResult | null
  ssrfTestTarget: string
  ssrfTesting: boolean
  users: AdminUser[]
}

function createSettingsForm(settings?: SystemSettings): AdminSettingsFormState {
  return {
    allowImageUpload: settings?.allow_image_upload ?? true,
    backupScheduleEnabled: settings?.backup_schedule_enabled ?? false,
    backupTimeOfDay: settings?.backup_time_of_day || "03:00",
    backupIncludeAssets: settings?.backup_include_assets ?? false,
    backupEncryptEnabled: settings?.backup_encrypt_enabled ?? false,
    backupEncryptionPassword: "",
    backupEncryptionPasswordConfigured: settings?.backup_encryption_password_configured ?? false,
    backupLocalDir: settings?.backup_local_dir || "",
    backupRetentionCount: settings?.backup_retention_count ?? 7,
    currencyApiKey: "",
    currencyApiKeyConfigured: settings?.currencyapi_key_configured ?? false,
    emailDomainWhitelist: settings?.email_domain_whitelist || "",
    exchangeRateSource: settings?.exchange_rate_source || "auto",
    iconProxyDomainWhitelist: settings?.icon_proxy_domain_whitelist || "",
    iconProxyEnabled: settings?.icon_proxy_enabled ?? true,
    maxIconFileSize: settings?.max_icon_file_size
      ? Math.round(settings.max_icon_file_size / 1024)
      : 64,
    mcpEnabled: settings?.mcp_enabled ?? false,
    auditEnabled: settings?.audit_enabled ?? true,
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
    registrationEnabled: settings?.registration_enabled ?? false,
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
    smtpRateLimitSeconds: settings?.smtp_rate_limit_seconds ?? 0,
    smtpSkipTLSVerify: settings?.smtp_skip_tls_verify ?? false,
    smtpTimeoutSeconds: settings?.smtp_timeout_seconds || 10,
    smtpUsername: settings?.smtp_username || "",
    ssrfAllowPrivateIP: settings?.ssrf_allow_private_ip ?? false,
    ssrfDomainFilterList: settings?.ssrf_domain_filter_list || "",
    ssrfDomainFilterMode: settings?.ssrf_domain_filter_mode || "blacklist",
    ssrfFilterResolvedIPs: settings?.ssrf_filter_resolved_ips ?? true,
    ssrfIPFilterList: settings?.ssrf_ip_filter_list || "",
    ssrfIPFilterMode: settings?.ssrf_ip_filter_mode || "blacklist",
    ssrfProtectionEnabled: settings?.ssrf_protection_enabled ?? true,
    systemProxyEnabled: settings?.system_proxy_enabled ?? false,
    systemProxyType: settings?.system_proxy_type || "http",
    systemProxyUrl: "",
    systemProxyUrlConfigured: settings?.system_proxy_url_configured ?? false,
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

function createBackupStatus(settings?: SystemSettings): BackupStatus {
  return {
    lastRunAt: settings?.backup_last_run_at || "",
    lastStatus: settings?.backup_last_status || "",
    lastError: settings?.backup_last_error || "",
  }
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
  const [backgroundTasks, setBackgroundTasks] = useState<BackgroundTask[]>([])
  const [loading, setLoading] = useState(true)
  const [settingsForm, setSettingsForm] = useState<AdminSettingsFormState>(() =>
    createSettingsForm()
  )

  const [includeAssetsInBackup, setIncludeAssetsInBackup] = useState(false)
  const [downloadPassword, setDownloadPassword] = useState("")
  const [restoreFile, setRestoreFile] = useState<File | null>(null)
  const [restorePassword, setRestorePassword] = useState("")
  const [restoreEncrypted, setRestoreEncrypted] = useState(false)
  const [restoreEncryptedEntry, setRestoreEncryptedEntry] = useState<EncryptedZipEntry | null>(
    null
  )
  const [restoreConfirmOpen, setRestoreConfirmOpen] = useState(false)

  const [backupStatus, setBackupStatus] = useState<BackupStatus>(() => createBackupStatus())
  const [runningBackup, setRunningBackup] = useState(false)
  const [localBackups, setLocalBackups] = useState<LocalBackupInfo[]>([])
  const [localBackupDir, setLocalBackupDir] = useState("")
  const [localBackupsRefreshing, setLocalBackupsRefreshing] = useState(false)

  const [rateStatus, setRateStatus] = useState<ExchangeRateStatus | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [backgroundTasksRefreshing, setBackgroundTasksRefreshing] = useState(false)
  const [smtpTestRecipient, setSMTPTestRecipient] = useState(() => getUser()?.email ?? "")
  const [smtpTesting, setSMTPTesting] = useState(false)
  const [ssrfTestTarget, setSSRFTestTarget] = useState("")
  const [ssrfTestResult, setSSRFTestResult] = useState<SSRFTestResult | null>(null)
  const [ssrfTesting, setSSRFTesting] = useState(false)

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
      api.get<ExchangeRateStatus>("/admin/exchange-rates/status"),
      api.get<BackgroundTask[]>("/admin/background-tasks"),
    ])
      .then(([usersData, settingsData, rateStatusData, backgroundTasksData]) => {
        setUsers(usersData || [])
        setSettingsForm(createSettingsForm(settingsData))
        setBackupStatus(createBackupStatus(settingsData))
        setRateStatus(rateStatusData)
        setBackgroundTasks(backgroundTasksData || [])
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))

    // Fetch the local backup list separately so a failure (e.g. an unreadable
    // backup directory returning 500) can't blank out the whole admin page.
    api
      .get<LocalBackupList>("/admin/backup/local")
      .then((localBackupsData) => {
        setLocalBackups(localBackupsData?.backups || [])
        setLocalBackupDir(localBackupsData?.directory || "")
      })
      .catch(() => void 0)
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
    if (newPassword.length < 8) {
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
        icon_proxy_enabled: settingsForm.iconProxyEnabled,
        icon_proxy_domain_whitelist: settingsForm.iconProxyDomainWhitelist,
        exchange_rate_source: settingsForm.exchangeRateSource,
        allow_image_upload: settingsForm.allowImageUpload,
        max_icon_file_size: settingsForm.maxIconFileSize * 1024,
        mcp_enabled: settingsForm.mcpEnabled,
        audit_enabled: settingsForm.auditEnabled,
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
        smtp_rate_limit_seconds: settingsForm.smtpRateLimitSeconds,
        smtp_skip_tls_verify: settingsForm.smtpSkipTLSVerify,
        ssrf_protection_enabled: settingsForm.ssrfProtectionEnabled,
        ssrf_allow_private_ip: settingsForm.ssrfAllowPrivateIP,
        ssrf_domain_filter_mode: settingsForm.ssrfDomainFilterMode,
        ssrf_domain_filter_list: settingsForm.ssrfDomainFilterList,
        ssrf_ip_filter_mode: settingsForm.ssrfIPFilterMode,
        ssrf_ip_filter_list: settingsForm.ssrfIPFilterList,
        ssrf_filter_resolved_ips: settingsForm.ssrfFilterResolvedIPs,
        system_proxy_enabled: settingsForm.systemProxyEnabled,
        system_proxy_type: settingsForm.systemProxyType,
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
        backup_schedule_enabled: settingsForm.backupScheduleEnabled,
        backup_time_of_day: settingsForm.backupTimeOfDay,
        backup_include_assets: settingsForm.backupIncludeAssets,
        backup_encrypt_enabled: settingsForm.backupEncryptEnabled,
        backup_local_dir: settingsForm.backupLocalDir,
        backup_retention_count: settingsForm.backupRetentionCount,
      }

      if (settingsForm.oidcClientSecret.trim()) {
        payload.oidc_client_secret = settingsForm.oidcClientSecret.trim()
      }
      if (settingsForm.smtpPassword.trim()) {
        payload.smtp_password = settingsForm.smtpPassword.trim()
      }
      if (settingsForm.systemProxyUrl.trim()) {
        payload.system_proxy_url = settingsForm.systemProxyUrl.trim()
      }
      if (settingsForm.currencyApiKey.trim()) {
        payload.currencyapi_key = settingsForm.currencyApiKey.trim()
      }
      if (settingsForm.backupEncryptionPassword.trim()) {
        payload.backup_encryption_password = settingsForm.backupEncryptionPassword.trim()
      }

      await api.put("/admin/settings", payload)
      const fresh = await api.get<SystemSettings>("/admin/settings")
      setSettingsForm(createSettingsForm(fresh))
      setBackupStatus(createBackupStatus(fresh))
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

  async function handleTestSSRF() {
    const target = ssrfTestTarget.trim()
    if (!target) {
      toast.error(t("admin.settings.ssrfTestTargetRequired"))
      return
    }

    setSSRFTesting(true)
    try {
      const result = await api.post<SSRFTestResult>("/admin/settings/ssrf/test", { target })
      setSSRFTestResult(result)
      toast.success(
        result.allowed
          ? t("admin.settings.ssrfTestAllowedToast")
          : t("admin.settings.ssrfTestBlockedToast")
      )
    } catch {
      void 0
    } finally {
      setSSRFTesting(false)
    }
  }

  async function handleRefreshBackgroundTasks() {
    setBackgroundTasksRefreshing(true)
    try {
      const tasks = await api.get<BackgroundTask[]>("/admin/background-tasks")
      setBackgroundTasks(tasks || [])
    } catch {
      void 0
    } finally {
      setBackgroundTasksRefreshing(false)
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

  async function handleDownloadBackup(reauthTicket: string): Promise<boolean> {
    try {
      const password = downloadPassword.trim()
      const res = await api.fetch("/admin/backup", {
        method: "POST",
        body: JSON.stringify({
          include_assets: includeAssetsInBackup,
          password,
          reauth_ticket: reauthTicket,
        }),
      })
      if (!res.ok) {
        const message = await readErrorMessage(res)
        toast.error(message ?? t("admin.backup.downloadFailed"))
        return false
      }

      const encrypted = password !== ""
      const blob = await res.blob()
      const url = window.URL.createObjectURL(blob)
      const anchor = document.createElement("a")
      anchor.href = url
      let filename =
        parseFilenameFromContentDisposition(res.headers.get("content-disposition")) ??
        `subdux-backup-${new Date().toISOString().split("T")[0]}${includeAssetsInBackup || encrypted ? ".zip" : ".db"}`
      // Encrypted archives are always zip containers.
      if (encrypted && !filename.toLowerCase().endsWith(".zip")) {
        filename = `${filename.replace(/\.[^./\\]+$/, "")}.zip`
      }
      anchor.download = filename
      document.body.appendChild(anchor)
      anchor.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(anchor)
      toast.success(t("admin.backup.downloadSuccess"))
      return true
    } catch {
      toast.error(t("admin.backup.downloadFailed"))
      return false
    }
  }

  async function handleRestoreFileChange(file: File | null) {
    setRestoreFile(file)
    setRestorePassword("")
    setRestoreEncrypted(false)
    setRestoreEncryptedEntry(null)
    setRestoreConfirmOpen(false)

    if (!file) {
      return
    }
    // Only ZIP archives can be encrypted; a plain .db is never encrypted.
    if (!file.name.toLowerCase().endsWith(".zip")) {
      return
    }

    try {
      const detection = await detectZipEncryption(file)
      if (detection.encrypted && detection.firstEncryptedEntry) {
        setRestoreEncrypted(true)
        setRestoreEncryptedEntry(detection.firstEncryptedEntry)
        toast.info(t("admin.backup.restoreEncryptedDetected"))
      }
    } catch {
      void 0
    }
  }

  // Client-side pre-checks for the restore inputs. Run this BEFORE opening the
  // re-auth dialog so a single-use re-auth ticket is never minted (and then
  // wasted) for a request that can't be sent. The restore file/password fields
  // live behind the modal, so a failure here can't be corrected from the dialog
  // anyway — the admin fixes the inputs, then re-authenticates.
  async function validateRestoreInputs(): Promise<boolean> {
    if (!restoreFile) {
      return false
    }

    const password = restorePassword.trim()

    if (restoreEncrypted) {
      if (password === "") {
        toast.error(t("admin.backup.restorePasswordRequired"))
        return false
      }
      if (restoreEncryptedEntry) {
        // Fast client-side WinZip-AES password check. When verification could
        // actually run and the password is wrong, stop early; otherwise fall
        // through and let the server perform the authoritative validation.
        const verification = await verifyZipPassword(restoreFile, restoreEncryptedEntry, password)
        if (verification.verified && !verification.valid) {
          toast.error(t("admin.backup.restoreWrongPassword"))
          return false
        }
      }
    }

    return true
  }

  async function handleRestore(reauthTicket: string): Promise<boolean> {
    // Inputs were validated via validateRestoreInputs() before re-auth; guard
    // against a missing file for type-safety without re-running the checks.
    if (!restoreFile) {
      return false
    }

    const password = restorePassword.trim()

    const formData = new FormData()
    formData.append("backup", restoreFile)
    if (password !== "") {
      formData.append("password", password)
    }

    try {
      const res = await api.fetch("/admin/restore", {
        method: "POST",
        headers: { "X-Reauth-Ticket": reauthTicket },
        body: formData,
      })
      if (!res.ok) {
        const message = await readErrorMessage(res)
        toast.error(message ?? t("admin.backup.restoreFailed"))
        return false
      }

      setRestoreConfirmOpen(false)
      toast.success(t("admin.backup.restoreSuccess"))
      return true
    } catch {
      toast.error(t("admin.backup.restoreFailed"))
      return false
    }
  }

  async function handleRefreshLocalBackups() {
    setLocalBackupsRefreshing(true)
    try {
      const data = await api.get<LocalBackupList>("/admin/backup/local")
      setLocalBackups(data?.backups || [])
      setLocalBackupDir(data?.directory || "")
    } catch {
      void 0
    } finally {
      setLocalBackupsRefreshing(false)
    }
  }

  async function handleRunBackupNow() {
    setRunningBackup(true)
    try {
      const result = await api.post<{ message: string; file: string }>("/admin/backup/run", {})
      toast.success(result?.message || t("admin.backup.runNowSuccess"))
      const [data, fresh] = await Promise.all([
        api.get<LocalBackupList>("/admin/backup/local"),
        api.get<SystemSettings>("/admin/settings"),
      ])
      setLocalBackups(data?.backups || [])
      setLocalBackupDir(data?.directory || "")
      setBackupStatus(createBackupStatus(fresh))
    } catch {
      toast.error(t("admin.backup.runNowFailed"))
    } finally {
      setRunningBackup(false)
    }
  }

  return {
    backgroundTasks,
    backgroundTasksRefreshing,
    backupStatus,
    createDialogOpen,
    downloadPassword,
    handleCreateUser,
    handleRefreshBackgroundTasks,
    handleRefreshLocalBackups,
    handleDeleteUser,
    handleDownloadBackup,
    handleRefreshRates,
    handleRegistrationEmailVerificationChange,
    handleRestore,
    handleValidateRestoreInputs: validateRestoreInputs,
    handleRunBackupNow,
    handleSaveSettings,
    handleTestSSRF,
    handleTestSMTP,
    handleToggleRole,
    handleToggleStatus,
    includeAssetsInBackup,
    loading,
    localBackupDir,
    localBackups,
    localBackupsRefreshing,
    newEmail,
    newPassword,
    newRole,
    newUsername,
    rateStatus,
    refreshing,
    restoreConfirmOpen,
    restoreEncrypted,
    restoreFile,
    restorePassword,
    runningBackup,
    setCreateDialogOpen,
    setDownloadPassword,
    setIncludeAssetsInBackup,
    setNewEmail,
    setNewPassword,
    setNewRole,
    setNewUsername,
    setRestoreConfirmOpen,
    setRestoreFile: handleRestoreFileChange,
    setRestorePassword,
    setSSRFTestTarget,
    setSettingsField,
    setSMTPTestRecipient,
    settingsForm,
    smtpTestRecipient,
    smtpTesting,
    ssrfTestResult,
    ssrfTestTarget,
    ssrfTesting,
    users,
  }
}
