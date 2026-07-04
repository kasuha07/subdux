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
} from "@/types"
import {
  buildAdminSettingsPayload,
  createAdminSettingsForm,
  mergeAdminSettingsFormScope,
  type AdminSettingsFormState,
  type AdminSettingsSaveScope,
} from "./admin-settings-form"

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
  handleCreateUser: (reauthTicket?: string) => Promise<void>
  handleRefreshBackgroundTasks: () => Promise<void>
  handleRefreshLocalBackups: () => Promise<void>
  handleDeleteUser: (id: number, reauthTicket: string) => Promise<void>
  handleDisableUserPasskeys: (user: AdminUser) => Promise<void>
  handleDisableUserTOTP: (user: AdminUser) => Promise<void>
  handleDownloadBackup: (reauthTicket: string) => Promise<boolean>
  handleRefreshRates: () => Promise<void>
  handleRegistrationEmailVerificationChange: (enabled: boolean) => void
  handleRestore: (reauthTicket: string) => Promise<boolean>
  handleValidateRestoreInputs: () => Promise<boolean>
  handleRunBackupNow: () => Promise<void>
  handleSaveAuthSettings: () => Promise<void>
  handleSaveBackupSettings: (reauthTicket: string) => Promise<void>
  handleSaveExchangeRateSettings: () => Promise<void>
  handleSaveGeneralSettings: () => Promise<void>
  handleSaveSMTPSettings: () => Promise<void>
  handleConfirmToggleRole: (reauthTicket: string) => Promise<void>
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
  roleReauthUser: AdminUser | null
  setCreateDialogOpen: (open: boolean) => void
  setDownloadPassword: (value: string) => void
  setIncludeAssetsInBackup: (value: boolean) => void
  setNewEmail: (value: string) => void
  setNewPassword: (value: string) => void
  setNewRole: (value: "user" | "admin") => void
  setNewUsername: (value: string) => void
  setRoleReauthUser: (user: AdminUser | null) => void
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
    createAdminSettingsForm()
  )
  const [savedSettingsForm, setSavedSettingsForm] = useState<AdminSettingsFormState>(() =>
    createAdminSettingsForm()
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
  const [roleReauthUser, setRoleReauthUser] = useState<AdminUser | null>(null)
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
        const form = createAdminSettingsForm(settingsData)
        setSettingsForm(form)
        setSavedSettingsForm(form)
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
    setRoleReauthUser(user)
  }

  async function handleConfirmToggleRole(reauthTicket: string) {
    if (!roleReauthUser) {
      return
    }

    const target = users.find((user) => user.id === roleReauthUser.id) ?? roleReauthUser
    const nextRole = target.role === "admin" ? "user" : "admin"
    try {
      await api.put(
        `/admin/users/${target.id}/role`,
        { role: nextRole },
        { headers: { "X-Reauth-Ticket": reauthTicket } }
      )
      setUsers((prev) =>
        prev.map((item) => (item.id === target.id ? { ...item, role: nextRole } : item))
      )
      setRoleReauthUser(null)
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

  async function handleDeleteUser(id: number, reauthTicket: string) {
    if (!confirm(t("admin.users.deleteConfirm"))) {
      return
    }

    try {
      await api.delete(`/admin/users/${id}`, { headers: { "X-Reauth-Ticket": reauthTicket } })
      setUsers((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("admin.users.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleDisableUserTOTP(user: AdminUser) {
    if (user.role === "admin" || !user.totp_enabled) {
      return
    }
    if (!confirm(t("admin.users.disable2FAConfirm"))) {
      return
    }

    try {
      await api.post(`/admin/users/${user.id}/disable-totp`, {})
      setUsers((prev) =>
        prev.map((item) => (item.id === user.id ? { ...item, totp_enabled: false } : item))
      )
      toast.success(t("admin.users.disable2FASuccess"))
    } catch {
      void 0
    }
  }

  async function handleDisableUserPasskeys(user: AdminUser) {
    if (user.role === "admin" || user.passkey_count <= 0) {
      return
    }
    if (!confirm(t("admin.users.disablePasskeysConfirm"))) {
      return
    }

    try {
      await api.post(`/admin/users/${user.id}/disable-passkeys`, {})
      setUsers((prev) =>
        prev.map((item) => (item.id === user.id ? { ...item, passkey_count: 0 } : item))
      )
      toast.success(t("admin.users.disablePasskeysSuccess"))
    } catch {
      void 0
    }
  }

  async function handleCreateUser(reauthTicket?: string) {
    if (!newUsername || !newEmail || !newPassword) {
      return
    }
    if (newPassword.length < 8) {
      toast.error(t("admin.users.passwordTooShort"))
      return
    }

    try {
      const headers =
        newRole === "admin" && reauthTicket ? { "X-Reauth-Ticket": reauthTicket } : undefined
      const user = await api.post<AdminUser>(
        "/admin/users",
        {
          username: newUsername,
          email: newEmail,
          password: newPassword,
          role: newRole,
        },
        { headers }
      )
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

  async function saveSettingsScope(scope: AdminSettingsSaveScope, reauthTicket?: string) {
    try {
      const payload = buildAdminSettingsPayload(settingsForm, scope)
      const headers = reauthTicket ? { "X-Reauth-Ticket": reauthTicket } : undefined
      const res = await api.fetch("/admin/settings", {
        method: "PUT",
        headers,
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const message = await readErrorMessage(res)
        toast.error(message ?? t("common.requestFailed"))
        return
      }
      const fresh = await api.get<SystemSettings>("/admin/settings")
      setSavedSettingsForm(createAdminSettingsForm(fresh))
      setSettingsForm((current) => mergeAdminSettingsFormScope(current, fresh, scope))
      setBackupStatus(createBackupStatus(fresh))
      if (scope === "general") {
        updateSiteTitle(fresh.site_name)
      }
      toast.success(t("admin.settings.saveSuccess"))
    } catch {
      void 0
    }
  }

  function handleSaveGeneralSettings() {
    return saveSettingsScope("general")
  }

  function handleSaveSMTPSettings() {
    return saveSettingsScope("smtp")
  }

  function handleSaveAuthSettings() {
    return saveSettingsScope("auth")
  }

  function handleSaveExchangeRateSettings() {
    return saveSettingsScope("exchange-rates")
  }

  function handleSaveBackupSettings(reauthTicket: string) {
    return saveSettingsScope("backup", reauthTicket)
  }

  function handleRegistrationEmailVerificationChange(enabled: boolean) {
    if (!enabled) {
      setSettingsField("registrationEmailVerificationEnabled", false)
      return
    }

    if (!hasSMTPConfigForRegistrationVerification(savedSettingsForm)) {
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
        headers: { "X-Reauth-Ticket": reauthTicket },
        body: JSON.stringify({
          include_assets: includeAssetsInBackup,
          password,
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
    handleDisableUserPasskeys,
    handleDisableUserTOTP,
    handleDownloadBackup,
    handleRefreshRates,
    handleRegistrationEmailVerificationChange,
    handleRestore,
    handleValidateRestoreInputs: validateRestoreInputs,
    handleRunBackupNow,
    handleSaveAuthSettings,
    handleSaveBackupSettings,
    handleSaveExchangeRateSettings,
    handleSaveGeneralSettings,
    handleSaveSMTPSettings,
    handleConfirmToggleRole,
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
    roleReauthUser,
    setCreateDialogOpen,
    setDownloadPassword,
    setIncludeAssetsInBackup,
    setNewEmail,
    setNewPassword,
    setNewRole,
    setNewUsername,
    setRoleReauthUser,
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
