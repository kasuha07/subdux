import { useEffect, useRef, useState, type FormEvent } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"

import { api, localizeBackendError, logout, logoutAll, setAuth } from "@/lib/api"
import { toast } from "sonner"
import type {
  AuthResponse,
  ConfirmEmailChangeInput,
  SendEmailChangeCodeInput,
  User,
} from "@/types"

interface UseSettingsAccountOptions {
  active: boolean
}

interface UseSettingsAccountResult {
  confirmPassword: string
  currentPassword: string
  emailChangeError: string
  emailChangeLoading: boolean
  emailCodeLoading: boolean
  emailCodeSent: boolean
  emailVerificationCode: string
  handleChangePassword: (event: FormEvent<HTMLFormElement>) => Promise<void>
  handleConfirmEmailChange: () => Promise<void>
  handleLogout: () => Promise<void>
  handleLogoutAll: () => Promise<void>
  handleSendEmailChangeCode: (reauthTicket: string) => Promise<void>
  validateEmailChangeCodeRequest: () => boolean
  newEmail: string
  newPassword: string
  passwordError: string
  passwordLoading: boolean
  passwordSuccess: string
  setConfirmPassword: (value: string) => void
  setCurrentPassword: (value: string) => void
  setEmailVerificationCode: (value: string) => void
  setNewEmail: (value: string) => void
  setNewPassword: (value: string) => void
  setUser: (user: User) => void
  user: User | null
}

export function useSettingsAccount({ active }: UseSettingsAccountOptions): UseSettingsAccountResult {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [user, setUserState] = useState<User | null>(null)
  const [newEmail, setNewEmail] = useState("")
  const [emailVerificationCode, setEmailVerificationCode] = useState("")
  const [emailCodeSent, setEmailCodeSent] = useState(false)
  const [emailCodeLoading, setEmailCodeLoading] = useState(false)
  const [emailChangeLoading, setEmailChangeLoading] = useState(false)
  const [emailChangeError, setEmailChangeError] = useState("")
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [passwordLoading, setPasswordLoading] = useState(false)
  const [passwordError, setPasswordError] = useState("")
  const [passwordSuccess] = useState("")

  const accountLoaded = useRef(false)

  useEffect(() => {
    if (!active || accountLoaded.current) {
      return
    }

    accountLoaded.current = true
    api.get<User>("/auth/me").then(setUserState).catch(() => void 0)
  }, [active])

  async function handleChangePassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setPasswordError("")

    if (newPassword !== confirmPassword) {
      setPasswordError(t("settings.account.passwordMismatch"))
      return
    }
    if (newPassword.length < 8) {
      setPasswordError(t("settings.account.passwordTooShort"))
      return
    }

    setPasswordLoading(true)
    try {
      await api.put("/auth/password", {
        current_password: currentPassword,
        new_password: newPassword,
      })
      toast.success(t("settings.account.passwordChanged"))
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : t("settings.account.passwordError"))
    } finally {
      setPasswordLoading(false)
    }
  }

  function validateEmailChangeTarget(): boolean {
    setEmailChangeError("")

    if (!newEmail.trim()) {
      setEmailChangeError(t("settings.account.newEmailRequired"))
      return false
    }
    if (user?.email && newEmail.trim().toLowerCase() === user.email.toLowerCase()) {
      setEmailChangeError(t("settings.account.newEmailMustBeDifferent"))
      return false
    }

    return true
  }

  function validateEmailChangeCodeRequest(): boolean {
    setEmailCodeSent(false)
    return validateEmailChangeTarget()
  }

  async function handleSendEmailChangeCode(reauthTicket: string) {
    if (!validateEmailChangeCodeRequest()) {
      return
    }

    setEmailCodeLoading(true)
    try {
      const payload: SendEmailChangeCodeInput = {
        new_email: newEmail.trim(),
      }
      await api.fetch("/auth/email/change/send-code", {
        method: "POST",
        headers: { "X-Reauth-Ticket": reauthTicket },
        body: JSON.stringify(payload),
      }).then(async (res) => {
        const body = await res.json().catch(() => null) as { error?: unknown } | null
        if (!res.ok) {
          throw new Error(localizeBackendError(body?.error))
        }
      })
      setEmailCodeSent(true)
      toast.success(t("settings.account.emailCodeSent"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailCodeLoading(false)
    }
  }

  async function handleConfirmEmailChange() {
    if (!validateEmailChangeTarget()) {
      return
    }
    if (!emailVerificationCode.trim()) {
      setEmailChangeError(t("settings.account.emailVerificationCodeRequired"))
      return
    }

    setEmailChangeLoading(true)
    try {
      const payload: ConfirmEmailChangeInput = {
        new_email: newEmail.trim(),
        verification_code: emailVerificationCode.trim(),
      }
      const res = await api.fetch("/auth/email/change/confirm", {
        method: "POST",
        body: JSON.stringify(payload),
      })
      const body = await res.json().catch(() => null) as AuthResponse | { error?: unknown } | null
      if (!res.ok) {
        const errorMessage = localizeBackendError(body && "error" in body ? body.error : undefined)
        setEmailChangeError(errorMessage)
        return
      }
      if (!body || !("user" in body)) {
        setEmailChangeError(t("settings.account.emailChangeError"))
        return
      }
      const authData = body
      setAuth(authData.access_token ?? authData.token, authData.user)
      setUserState(authData.user)
      setNewEmail("")
      setEmailVerificationCode("")
      setEmailCodeSent(false)
      toast.success(t("settings.account.emailChangeSuccess"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailChangeLoading(false)
    }
  }

  async function handleLogout() {
    try {
      await logout()
      toast.success(t("settings.account.logoutSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    }
    navigate("/login")
  }

  async function handleLogoutAll() {
    try {
      await logoutAll()
      toast.success(t("settings.account.logoutAllSuccess"))
      navigate("/login")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("common.requestFailed"))
    }
  }

  return {
    confirmPassword,
    currentPassword,
    emailChangeError,
    emailChangeLoading,
    emailCodeLoading,
    emailCodeSent,
    emailVerificationCode,
    handleChangePassword,
    handleConfirmEmailChange,
    handleLogout,
    handleLogoutAll,
    handleSendEmailChangeCode,
    validateEmailChangeCodeRequest,
    newEmail,
    newPassword,
    passwordError,
    passwordLoading,
    passwordSuccess,
    setConfirmPassword,
    setCurrentPassword,
    setEmailVerificationCode,
    setNewEmail,
    setNewPassword,
    setUser: setUserState,
    user,
  }
}
