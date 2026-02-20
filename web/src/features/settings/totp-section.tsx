import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { User } from "@/types"
import TotpSetupDialog from "./totp-setup-dialog"

interface Props {
  user: User | null
  onUserChange: (user: User) => void
}

export default function TotpSection({ user, onUserChange }: Props) {
  const { t } = useTranslation()
  const [setupOpen, setSetupOpen] = useState(false)
  const [showDisableForm, setShowDisableForm] = useState(false)
  const [disablePassword, setDisablePassword] = useState("")
  const [disableCode, setDisableCode] = useState("")
  const [disabling, setDisabling] = useState(false)
  const [disableError, setDisableError] = useState("")

  const totpEnabled = user?.totp_enabled ?? false

  function handleEnabled() {
    setSetupOpen(false)
    toast.success(t("settings.twoFactor.enableSuccess"))
    api.get<User>("/auth/me").then(onUserChange).catch(() => void 0)
  }

  async function handleDisable() {
    if (!disablePassword || !disableCode) return
    setDisableError("")
    setDisabling(true)
    try {
      await api.post("/auth/totp/disable", { password: disablePassword, code: disableCode.trim() })
      toast.success(t("settings.twoFactor.disableSuccess"))
      setShowDisableForm(false)
      setDisablePassword("")
      setDisableCode("")
      api.get<User>("/auth/me").then(onUserChange).catch(() => void 0)
    } catch (err) {
      setDisableError(err instanceof Error ? err.message : t("settings.twoFactor.verifyError"))
    } finally {
      setDisabling(false)
    }
  }

  return (
    <>
      <TotpSetupDialog
        open={setupOpen}
        onOpenChange={setSetupOpen}
        onEnabled={handleEnabled}
      />

      <div className="space-y-3">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-medium">{t("settings.twoFactor.title")}</h3>
            <Badge variant={totpEnabled ? "default" : "secondary"} className="text-xs">
              {totpEnabled ? t("settings.twoFactor.enabled") : t("settings.twoFactor.disabled")}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.twoFactor.description")}
          </p>
        </div>

        {!totpEnabled && (
          <Button size="sm" variant="outline" onClick={() => setSetupOpen(true)}>
            {t("settings.twoFactor.enable")}
          </Button>
        )}

        {totpEnabled && !showDisableForm && (
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:text-destructive"
            onClick={() => setShowDisableForm(true)}
          >
            {t("settings.twoFactor.disable")}
          </Button>
        )}

        {totpEnabled && showDisableForm && (
          <div className="space-y-3 max-w-sm">
            <Separator />
            <div>
              <p className="text-sm font-medium">{t("settings.twoFactor.disableTitle")}</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                {t("settings.twoFactor.disableDescription")}
              </p>
            </div>
            {disableError && (
              <div className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {disableError}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="disable-totp-password">{t("settings.twoFactor.passwordLabel")}</Label>
              <Input
                id="disable-totp-password"
                type="password"
                placeholder="••••••••"
                value={disablePassword}
                onChange={(e) => setDisablePassword(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="disable-totp-code">{t("settings.twoFactor.codeLabel")}</Label>
              <Input
                id="disable-totp-code"
                type="text"
                inputMode="numeric"
                placeholder={t("settings.twoFactor.codePlaceholder")}
                value={disableCode}
                onChange={(e) => setDisableCode(e.target.value)}
                maxLength={8}
              />
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="destructive"
                disabled={disabling || !disablePassword || !disableCode}
                onClick={() => void handleDisable()}
              >
                {disabling ? t("settings.twoFactor.disabling") : t("settings.twoFactor.disableButton")}
              </Button>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setShowDisableForm(false)
                  setDisablePassword("")
                  setDisableCode("")
                  setDisableError("")
                }}
              >
                {t("subscription.form.cancel")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  )
}
