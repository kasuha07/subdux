import { useEffect, useRef, useState, type FormEvent } from "react"
import { useNavigate } from "react-router-dom"
import { useTranslation } from "react-i18next"

import { api, clearToken, setAuth } from "@/lib/api"
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
  emailChangePassword: string
  emailCodeLoading: boolean
  emailCodeSent: boolean
  emailVerificationCode: string
  handleChangePassword: (event: FormEvent<HTMLFormElement>) => Promise<void>
  handleConfirmEmailChange: (event: FormEvent<HTMLFormElement>) => Promise<void>
  handleLogout: () => void
  handleSendEmailChangeCode: (event: FormEvent<HTMLFormElement>) => Promise<void>
  newEmail: string
  newPassword: string
  passwordError: string
  passwordLoading: boolean
  passwordSuccess: string
  setConfirmPassword: (value: string) => void
  setCurrentPassword: (value: string) => void
  setEmailChangePassword: (value: string) => void
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
  const [emailChangePassword, setEmailChangePassword] = useState("")
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
    if (newPassword.length < 6) {
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

  async function handleSendEmailChangeCode(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setEmailChangeError("")
    setEmailCodeSent(false)

    if (!newEmail.trim()) {
      setEmailChangeError(t("settings.account.newEmailRequired"))
      return
    }
    if (!emailChangePassword) {
      setEmailChangeError(t("settings.account.emailChangePasswordRequired"))
      return
    }
    if (user?.email && newEmail.trim().toLowerCase() === user.email.toLowerCase()) {
      setEmailChangeError(t("settings.account.newEmailMustBeDifferent"))
      return
    }

    setEmailCodeLoading(true)
    try {
      const payload: SendEmailChangeCodeInput = {
        new_email: newEmail.trim(),
        password: emailChangePassword,
      }
      await api.post<{ message: string }>("/auth/email/change/send-code", payload)
      setEmailCodeSent(true)
      toast.success(t("settings.account.emailCodeSent"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailCodeLoading(false)
    }
  }

  async function handleConfirmEmailChange(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setEmailChangeError("")

    if (!newEmail.trim()) {
      setEmailChangeError(t("settings.account.newEmailRequired"))
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
      const authData = await api.post<AuthResponse>("/auth/email/change/confirm", payload)
      setAuth(authData.token, authData.user)
      setUserState(authData.user)
      setNewEmail("")
      setEmailChangePassword("")
      setEmailVerificationCode("")
      setEmailCodeSent(false)
      toast.success(t("settings.account.emailChangeSuccess"))
    } catch (err) {
      setEmailChangeError(err instanceof Error ? err.message : t("settings.account.emailChangeError"))
    } finally {
      setEmailChangeLoading(false)
    }
  }

  function handleLogout() {
    clearToken()
    toast.success(t("settings.account.logoutSuccess"))
    navigate("/login")
  }

  return {
    confirmPassword,
    currentPassword,
    emailChangeError,
    emailChangeLoading,
    emailChangePassword,
    emailCodeLoading,
    emailCodeSent,
    emailVerificationCode,
    handleChangePassword,
    handleConfirmEmailChange,
    handleLogout,
    handleSendEmailChangeCode,
    newEmail,
    newPassword,
    passwordError,
    passwordLoading,
    passwordSuccess,
    setConfirmPassword,
    setCurrentPassword,
    setEmailChangePassword,
    setEmailVerificationCode,
    setNewEmail,
    setNewPassword,
    setUser: setUserState,
    user,
  }
}
