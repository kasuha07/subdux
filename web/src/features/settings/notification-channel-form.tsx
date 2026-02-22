import { useState, type FormEvent } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

import { WEBHOOK_HEADERS_PARSE_ERROR } from "./notification-channel-form/constants"
import { NotificationChannelConfigFields } from "./notification-channel-form/config-fields"
import type {
  ChannelType,
  NotificationChannelFormProps,
  NotificationChannelFormValues,
} from "./notification-channel-form/types"
import { buildConfig, createInitialValues } from "./notification-channel-form/utils"

export function NotificationChannelForm({ channel, onClose, onSave, open, saving }: NotificationChannelFormProps) {
  const { t } = useTranslation()
  const isEditing = !!channel

  const [type, setType] = useState<ChannelType>(channel?.type as ChannelType ?? "smtp")
  const [values, setValues] = useState<NotificationChannelFormValues>(() => createInitialValues(channel))

  function handleValueChange<K extends keyof NotificationChannelFormValues>(
    key: K,
    value: NotificationChannelFormValues[K]
  ) {
    setValues((prev) => ({ ...prev, [key]: value }))
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()

    try {
      const config = buildConfig(type, values)
      void onSave(type, config)
    } catch (error) {
      if (error instanceof Error && error.message === WEBHOOK_HEADERS_PARSE_ERROR) {
        window.alert(t("settings.notifications.channels.configFields.headersInvalid"))
        return
      }
      window.alert(t("settings.notifications.channels.configFields.headersInvalid"))
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onClose() }}>
      <DialogContent className="flex max-h-[calc(100vh-1.5rem)] max-w-md flex-col gap-0 overflow-hidden p-0 sm:max-h-[85vh]">
        <DialogHeader className="border-b px-5 pt-5 pb-4 sm:px-6">
          <DialogTitle>
            {isEditing ? t("settings.notifications.channels.edit") : t("settings.notifications.channels.addButton")}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-5 py-4 sm:px-6">
            <NotificationChannelConfigFields
              isEditing={isEditing}
              type={type}
              values={values}
              onTypeChange={setType}
              onValueChange={handleValueChange}
              t={t}
            />
          </div>

          <div className="sticky bottom-0 z-10 border-t bg-background/95 px-5 py-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:px-6">
            <div className="flex gap-2">
              <Button type="button" variant="outline" className="flex-1" onClick={onClose}>
                {t("settings.notifications.channels.cancel")}
              </Button>
              <Button type="submit" className="flex-1" disabled={saving}>
                {saving ? t("settings.notifications.channels.adding") : t("settings.notifications.channels.save")}
              </Button>
            </div>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  )
}
