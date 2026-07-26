import { useCallback, useEffect, useState } from "react"

import { updateSiteTitle } from "@/hooks/useSiteSettings"
import { api, getAPIErrorMessage, getUser, localizeBackendMessage, localizeBackendMessageResponse } from "@/lib/api"
import {
  detectZipEncryption,
  verifyZipPassword,
  type EncryptedZipEntry,
} from "@/lib/zip-encryption"
import { toast } from "@/lib/toast"
import type {
  AdminUser,
  BackgroundTask,
  BackupDestination,
  BackupDestinationRunResult,
  BackupRunList,
  BackupRunRecord,
  BackupRunResponse,
  DestinationBackup,
  DestinationBackupList,
  ExchangeRateStatus,
  SSRFTestResult,
  SystemSettings,
} from "@/types"
import { summarizeBackupRun } from "./backup-run"
import { mutationSucceeded, type DestinationProbeRequest } from "./backup-destinations"
import {
  buildAdminSettingsPayload,
  createAdminSettingsForm,
  mergeAdminSettingsFormScope,
  type AdminSettingsFormState,
  type AdminSettingsSaveScope,
} from "./admin-settings-form"

interface UseAdminPageStateOptions {
  t: (key: string, options?: Record<string, unknown>) => string
}

interface RestoreBackupResponse {
  skipped_asset_count?: number
}

interface UseAdminPageStateResult {
  backgroundTasks: BackgroundTask[]
  backgroundTasksRefreshing: boolean
  createDialogOpen: boolean
  destinations: BackupDestination[]
  destinationsRefreshing: boolean
  downloadPassword: string
  handleCreateUser: (reauthTicket?: string) => Promise<void>
  handleRefreshBackgroundTasks: () => Promise<void>
  handleRefreshBackupRuns: () => Promise<void>
  handleRefreshDestinations: () => Promise<void>
  handleCreateDestination: (body: { type: string; enabled: boolean; config: string; sort_order: number }, reauthTicket: string) => Promise<boolean>
  handleUpdateDestination: (id: number, body: { revision: number; enabled?: boolean; config?: string; sort_order?: number; cleared_secret_fields?: string[] }, reauthTicket: string) => Promise<boolean>
  handleDeleteDestination: (id: number, revision: number, reauthTicket: string) => Promise<boolean>
  handleTestDestination: (id: number) => Promise<void>
  handleListDestinationBackups: (id: number) => Promise<DestinationBackup[]>
  handleTestDestinationConfig: (body: DestinationProbeRequest) => Promise<void>
  handleRunDestinationBackup: (id: number, reauthTicket: string) => Promise<void>
  handleDeleteUser: (id: number, reauthTicket: string) => Promise<void>
  handleDisableUserPasskeys: (user: AdminUser, reauthTicket: string) => Promise<void>
  handleDisableUserTOTP: (user: AdminUser, reauthTicket: string) => Promise<void>
  handleDownloadBackup: (reauthTicket: string) => Promise<boolean>
  handleRefreshRates: () => Promise<void>
  handleRegistrationEmailVerificationChange: (enabled: boolean) => void
  handleRestore: (reauthTicket: string) => Promise<boolean>
  handleRestoreFromDestination: (
    id: number,
    archiveName: string,
    password: string,
    reauthTicket: string
  ) => Promise<boolean>
  handleValidateRestoreInputs: () => Promise<boolean>
  handleSaveAuthSettings: () => Promise<void>
  handleSaveExchangeRateSettings: () => Promise<void>
  handleSaveGeneralSettings: () => Promise<void>
  handleSaveSMTPSettings: () => Promise<void>
  handleConfirmToggleRole: (reauthTicket: string) => Promise<void>
  handleTestSSRF: () => Promise<void>
  handleTestSMTP: () => Promise<void>
  handleToggleRole: (user: AdminUser) => Promise<void>
  handleToggleStatus: (user: AdminUser) => Promise<void>
  backupRuns: BackupRunRecord[]
  backupRunsRefreshing: boolean
  includeAssetsInBackup: boolean
  loading: boolean
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
  runningDestinationId: number | null
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

  // Id of the destination whose manual run is in flight, or null when idle.
  const [runningDestinationId, setRunningDestinationId] = useState<number | null>(null)
  const [backupRuns, setBackupRuns] = useState<BackupRunRecord[]>([])
  const [backupRunsRefreshing, setBackupRunsRefreshing] = useState(false)

  const [rateStatus, setRateStatus] = useState<ExchangeRateStatus | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [backgroundTasksRefreshing, setBackgroundTasksRefreshing] = useState(false)
  const [smtpTestRecipient, setSMTPTestRecipient] = useState(() => getUser()?.email ?? "")
  const [smtpTesting, setSMTPTesting] = useState(false)
  const [ssrfTestTarget, setSSRFTestTarget] = useState("")
  const [ssrfTestResult, setSSRFTestResult] = useState<SSRFTestResult | null>(null)
  const [ssrfTesting, setSSRFTesting] = useState(false)

  const [destinations, setDestinations] = useState<BackupDestination[]>([])
  const [destinationsRefreshing, setDestinationsRefreshing] = useState(false)

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
        setRateStatus(rateStatusData)
        setBackgroundTasks(backgroundTasksData || [])
      })
      .catch(() => void 0)
      .finally(() => setLoading(false))

    // Fetch the backup history separately so a failure can't blank out the
    // whole admin page.
    api
      .get<BackupRunList>("/admin/backup/runs")
      .then((data) => setBackupRuns(data?.runs || []))
      .catch(() => void 0)

    // Fetch backup destinations separately; a failure here also shouldn't
    // prevent the rest of the admin page from loading.
    api
      .get<BackupDestination[]>("/admin/backup/destinations")
      .then((data) => setDestinations(data || []))
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
        { headers: { "X-Reauth-Ticket": reauthTicket }, errorHandling: "toast" }
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
      await api.put(`/admin/users/${user.id}/status`, { status: nextStatus }, { errorHandling: "toast" })
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
      await api.delete(`/admin/users/${id}`, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorHandling: "toast",
      })
      setUsers((prev) => prev.filter((item) => item.id !== id))
      toast.success(t("admin.users.deleteSuccess"))
    } catch {
      void 0
    }
  }

  async function handleDisableUserTOTP(user: AdminUser, reauthTicket: string) {
    if (user.role === "admin" || !user.totp_enabled) {
      return
    }

    try {
      await api.post(
        `/admin/users/${user.id}/disable-totp`,
        {},
        { headers: { "X-Reauth-Ticket": reauthTicket }, errorHandling: "toast" }
      )
      setUsers((prev) =>
        prev.map((item) => (item.id === user.id ? { ...item, totp_enabled: false } : item))
      )
      toast.success(t("admin.users.disable2FASuccess"))
    } catch {
      void 0
    }
  }

  async function handleDisableUserPasskeys(user: AdminUser, reauthTicket: string) {
    if (user.role === "admin" || user.passkey_count <= 0) {
      return
    }

    try {
      await api.post(
        `/admin/users/${user.id}/disable-passkeys`,
        {},
        { headers: { "X-Reauth-Ticket": reauthTicket }, errorHandling: "toast" }
      )
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
        headers ? { headers, errorHandling: "toast" } : { errorHandling: "toast" }
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

  async function saveSettingsScope(scope: AdminSettingsSaveScope) {
    try {
      const payload = buildAdminSettingsPayload(settingsForm, scope)
      await api.put("/admin/settings", payload, { errorHandling: "toast" })
      const fresh = await api.get<SystemSettings>("/admin/settings", { errorHandling: "toast" })
      setSavedSettingsForm(createAdminSettingsForm(fresh))
      setSettingsForm((current) => mergeAdminSettingsFormScope(current, fresh, scope))
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
      }, { errorHandling: "toast" })
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
      const result = await api.post<SSRFTestResult>(
        "/admin/settings/ssrf/test",
        { target },
        { errorHandling: "toast" }
      )
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
      const tasks = await api.get<BackgroundTask[]>("/admin/background-tasks", {
        errorHandling: "toast",
      })
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
      const res = await api.response("/admin/backup", {
        method: "POST",
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorFallbackKey: "admin.backup.downloadFailed",
        body: JSON.stringify({
          include_assets: includeAssetsInBackup,
          password,
        }),
      })

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
      // The download is recorded as a backup run; refresh the history so it
      // appears immediately. Best-effort: the download already succeeded.
      api
        .get<BackupRunList>("/admin/backup/runs")
        .then((data) => setBackupRuns(data?.runs || []))
        .catch(() => void 0)
      return true
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "admin.backup.downloadFailed"))
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
      const result = await api.uploadFile<RestoreBackupResponse | undefined>("/admin/restore", formData, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorFallbackKey: "admin.backup.restoreFailed",
      })

      setRestoreConfirmOpen(false)
      toast.success(t("admin.backup.restoreSuccess"))
      if ((result?.skipped_asset_count ?? 0) > 0) {
        toast.warning(t("admin.backup.restoreSkippedAssets"))
      }
      return true
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "admin.backup.restoreFailed"))
      return false
    }
  }

  // Mirrors handleRestore: same success/warning toasts, and deliberately no
  // logout or reload — the upload restore does not do either, and the server
  // only asks the admin to refresh.
  async function handleRestoreFromDestination(
    id: number,
    archiveName: string,
    password: string,
    reauthTicket: string
  ): Promise<boolean> {
    try {
      const result = await api.post<RestoreBackupResponse | undefined>(
        `/admin/backup/destinations/${id}/restore`,
        { archive_name: archiveName, password: password.trim() },
        {
          headers: { "X-Reauth-Ticket": reauthTicket },
          errorFallbackKey: "admin.backup.restoreFailed",
        }
      )
      toast.success(t("admin.backup.restoreSuccess"))
      if ((result?.skipped_asset_count ?? 0) > 0) {
        toast.warning(t("admin.backup.restoreSkippedAssets"))
      }
      return true
    } catch (error) {
      toast.error(getAPIErrorMessage(error, "admin.backup.restoreFailed"))
      return false
    }
  }

  async function handleRefreshDestinations() {
    setDestinationsRefreshing(true)
    try {
      const data = await api.get<BackupDestination[]>("/admin/backup/destinations", {
        errorHandling: "toast",
      })
      setDestinations(data || [])
    } catch {
      void 0
    } finally {
      setDestinationsRefreshing(false)
    }
  }

  async function handleCreateDestination(
    body: { type: string; enabled: boolean; config: string; sort_order: number },
    reauthTicket: string
  ): Promise<boolean> {
    try {
      const created = await api.post<BackupDestination>("/admin/backup/destinations", body, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorHandling: "toast",
      })
      if (mutationSucceeded(created)) {
        setDestinations((prev) => [...prev, created])
        toast.success(t("admin.backup.destinations.createSuccess"))
        return true
      }
    } catch {
      return false
    }
    return false
  }

  async function handleUpdateDestination(
    id: number,
    body: { revision: number; enabled?: boolean; config?: string; sort_order?: number; cleared_secret_fields?: string[] },
    reauthTicket: string
  ): Promise<boolean> {
    try {
      const updated = await api.put<BackupDestination>(`/admin/backup/destinations/${id}`, body, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorHandling: "toast",
      })
      if (mutationSucceeded(updated)) {
        setDestinations((prev) => prev.map((d) => (d.id === id ? updated : d)))
        toast.success(t("admin.backup.destinations.updateSuccess"))
        return true
      }
    } catch {
      return false
    }
    return false
  }

  async function handleDeleteDestination(id: number, revision: number, reauthTicket: string): Promise<boolean> {
    try {
      await api.delete(`/admin/backup/destinations/${id}?revision=${encodeURIComponent(String(revision))}`, {
        headers: { "X-Reauth-Ticket": reauthTicket },
        errorHandling: "toast",
      })
      setDestinations((prev) => prev.filter((d) => d.id !== id))
      toast.success(t("admin.backup.destinations.deleteSuccess"))
      return true
    } catch {
      return false
    }
  }

  async function handleTestDestination(id: number) {
    try {
      const result = await api.post<{
        message_code?: string
        message_params?: Record<string, unknown>
      }>(`/admin/backup/destinations/${id}/test`, {}, { errorHandling: "toast" })
      toast.success(localizeBackendMessage(result?.message_code, result?.message_params, "admin.backup.destinations.testSuccess"))
    } catch {
      void 0
    }
  }

  // Reads the archives stored at one destination. No reauth ticket: the
  // endpoint is read-only against an already-saved destination, matching the
  // connectivity probe.
  async function handleListDestinationBackups(id: number): Promise<DestinationBackup[]> {
    try {
      const data = await api.get<DestinationBackupList>(
        `/admin/backup/destinations/${id}/backups`,
        { errorHandling: "toast" }
      )
      return data?.backups || []
    } catch {
      return []
    }
  }

  // handleTestDestinationConfig probes a destination the admin is still editing,
  // so it takes the form's own config instead of a saved id. It carries no
  // reauth ticket for the same reason the saved-destination probe does not: it
  // only reads. The server refuses to pair a stored secret with an endpoint
  // changed in the same request, and that refusal surfaces here as an ordinary
  // toast telling the admin to re-enter the secret.
  async function handleTestDestinationConfig(body: DestinationProbeRequest) {
    try {
      const result = await api.post<{
        message_code?: string
        message_params?: Record<string, unknown>
      }>("/admin/backup/destinations/test", body, { errorHandling: "toast" })
      toast.success(localizeBackendMessage(result?.message_code, result?.message_params, "admin.backup.destinations.testSuccess"))
    } catch {
      void 0
    }
  }

  async function handleRefreshBackupRuns() {
    setBackupRunsRefreshing(true)
    try {
      const data = await api.get<BackupRunList>("/admin/backup/runs", {
        errorHandling: "toast",
      })
      setBackupRuns(data?.runs || [])
    } catch {
      void 0
    } finally {
      setBackupRunsRefreshing(false)
    }
  }

  // handleRunDestinationBackup runs ONE destination's backup plan on demand. The
  // result handling is identical to a scheduled run's, so the same
  // summarizeBackupRun toasts apply.
  async function handleRunDestinationBackup(id: number, reauthTicket: string) {
    setRunningDestinationId(id)
    try {
      let result: BackupRunResponse
      try {
        result = await api.post<BackupRunResponse>(`/admin/backup/destinations/${id}/run`, {}, {
          headers: { "X-Reauth-Ticket": reauthTicket },
        })
      } catch (error) {
        // Only a failed backup POST means the backup itself did not complete.
        toast.error(getAPIErrorMessage(error, "admin.backup.runNowFailed"))
        return
      }

      const { failedDestinations, retentionFailures, bookkeepingFailures, topLevelFailure, globalBookkeepingFailure } = summarizeBackupRun(result)
      const formatDestination = (destination: BackupDestinationRunResult) => {
        const typeLabel =
          destination.type === "s3"
            ? t("admin.backup.destinations.typeS3")
            : destination.type === "webdav"
              ? t("admin.backup.destinations.typeWebDAV")
              : destination.type === "local"
                ? t("admin.backup.destinations.typeLocal")
                : destination.type
        return `${typeLabel} #${destination.destination_id}`
      }

      if (failedDestinations.length > 0) {
        toast.warning(
          t("admin.backup.runNowPartialSuccess", {
            destinations: failedDestinations.map(formatDestination).join(", "),
          })
        )
      }
      if (retentionFailures.length > 0) {
        toast.warning(
          t("admin.backup.runNowRetentionWarning", {
            destinations: retentionFailures.map(formatDestination).join(", "),
          })
        )
      }
      if (bookkeepingFailures.length > 0) {
        toast.warning(
          t("admin.backup.runNowBookkeepingWarning", {
            destinations: bookkeepingFailures.map(formatDestination).join(", "),
          })
        )
      }
      if (globalBookkeepingFailure) {
        toast.warning(
          t("admin.backup.runNowStatusWarning", {
            error:
              result.global_bookkeeping_error ||
              result.error ||
              t("admin.backup.runNowStatusWarningFallback"),
          })
        )
      }
      if (
        topLevelFailure &&
        failedDestinations.length === 0 &&
        retentionFailures.length === 0 &&
        bookkeepingFailures.length === 0 &&
        !globalBookkeepingFailure
      ) {
        toast.warning(
          t("admin.backup.runNowStatusWarning", {
            error: result.error || t("admin.backup.runNowStatusWarningFallback"),
          })
        )
      }
      if (!topLevelFailure && failedDestinations.length === 0 && retentionFailures.length === 0 && bookkeepingFailures.length === 0) {
        toast.success(localizeBackendMessageResponse(result, "admin.backup.runNowSuccess"))
      }
      // The backup has already completed at this point. Refreshes are
      // best-effort and must not turn a delivered backup into a failed run.
      try {
        const [backupRunsResult] = await Promise.allSettled([
          api.get<BackupRunList>("/admin/backup/runs"),
        ])
        if (backupRunsResult.status === "fulfilled") {
          setBackupRuns(backupRunsResult.value?.runs || [])
        }
        // Refresh the destinations so this row's status badges and last-run
        // timestamps reflect the run that just finished.
        await handleRefreshDestinations()
      } catch {
        // A completed backup remains successful even if a refresh helper
        // unexpectedly fails outside its own request-level handling.
        void 0
      }
    } finally {
      setRunningDestinationId(null)
    }
  }

  return {
    backgroundTasks,
    backgroundTasksRefreshing,
    createDialogOpen,
    destinations,
    destinationsRefreshing,
    downloadPassword,
    handleCreateUser,
    handleRefreshBackgroundTasks,
    handleRefreshBackupRuns,
    handleRefreshDestinations,
    handleCreateDestination,
    handleUpdateDestination,
    handleDeleteDestination,
    handleTestDestination,
    handleListDestinationBackups,
    handleTestDestinationConfig,
    handleRunDestinationBackup,
    handleDeleteUser,
    handleDisableUserPasskeys,
    handleDisableUserTOTP,
    handleDownloadBackup,
    handleRefreshRates,
    handleRegistrationEmailVerificationChange,
    handleRestore,
    handleRestoreFromDestination,
    handleValidateRestoreInputs: validateRestoreInputs,
    handleSaveAuthSettings,
    handleSaveExchangeRateSettings,
    handleSaveGeneralSettings,
    handleSaveSMTPSettings,
    handleConfirmToggleRole,
    handleTestSSRF,
    handleTestSMTP,
    handleToggleRole,
    handleToggleStatus,
    backupRuns,
    backupRunsRefreshing,
    includeAssetsInBackup,
    loading,
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
    runningDestinationId,
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
