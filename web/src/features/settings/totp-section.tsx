import { useState } from "react"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { api } from "@/lib/api"
import { toast } from "sonner"
import type { User } from "@/types"
import ReauthDialog from "@/features/admin/reauth-dialog"
import TotpSetupDialog from "./totp-setup-dialog"

interface Props {
  user: User | null
  onUserChange: (user: User) => void
}

export default function TotpSection({ user, onUserChange }: Props) {
  const { t } = useTranslation()
  const [setupOpen, setSetupOpen] = useState(false)
  const [setupReauthTicket, setSetupReauthTicket] = useState("")
  const [reauthOpen, setReauthOpen] = useState(false)
  const [disableReauthOpen, setDisableReauthOpen] = useState(false)
  const [disabling, setDisabling] = useState(false)

  const totpEnabled = user?.totp_enabled ?? false

  function handleEnabled() {
    setSetupOpen(false)
    setSetupReauthTicket("")
    toast.success(t("settings.twoFactor.enableSuccess"))
    api.get<User>("/auth/me").then(onUserChange).catch(() => void 0)
  }

  async function handleDisable(reauthTicket: string) {
    setDisabling(true)
    try {
      await api.post("/auth/totp/disable", {}, { headers: { "X-Reauth-Ticket": reauthTicket } })
      toast.success(t("settings.twoFactor.disableSuccess"))
      api.get<User>("/auth/me").then(onUserChange).catch(() => void 0)
    } finally {
      setDisabling(false)
    }
  }

  return (
    <>
      <TotpSetupDialog
        open={setupOpen}
        onOpenChange={(open) => {
          setSetupOpen(open)
          if (!open) {
            setSetupReauthTicket("")
          }
        }}
        reauthTicket={setupReauthTicket}
        onEnabled={handleEnabled}
      />
      <ReauthDialog
        operation="enable_totp"
        open={reauthOpen}
        onOpenChange={setReauthOpen}
        onVerified={(ticket) => {
          setSetupReauthTicket(ticket)
          setSetupOpen(true)
        }}
        title={t("settings.twoFactor.reauth.title")}
        description={t("settings.twoFactor.reauth.description")}
      />
      <ReauthDialog
        operation="disable_totp"
        open={disableReauthOpen}
        onOpenChange={setDisableReauthOpen}
        onVerified={(ticket) => handleDisable(ticket)}
        title={t("settings.twoFactor.disableReauth.title")}
        description={t("settings.twoFactor.disableReauth.description")}
        confirmVariant="destructive"
      />

      <div className="space-y-3">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="text-base font-semibold tracking-tight select-none">{t("settings.twoFactor.title")}</h3>
            <Badge variant={totpEnabled ? "default" : "secondary"} className="text-xs">
              {totpEnabled ? t("settings.twoFactor.enabled") : t("settings.twoFactor.disabled")}
            </Badge>
          </div>
          <p className="text-sm text-muted-foreground mt-0.5">
            {t("settings.twoFactor.description")}
          </p>
        </div>

        {!totpEnabled && (
          <Button size="sm" variant="outline" onClick={() => setReauthOpen(true)}>
            {t("settings.twoFactor.enable")}
          </Button>
        )}

        {totpEnabled && (
          <Button
            size="sm"
            variant="outline"
            className="text-destructive hover:text-destructive"
            disabled={disabling}
            onClick={() => setDisableReauthOpen(true)}
          >
            {disabling ? t("settings.twoFactor.disabling") : t("settings.twoFactor.disable")}
          </Button>
        )}
      </div>
    </>
  )
}
