import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import type { NotificationChannel } from "@/types"

interface Props {
  channels: NotificationChannel[]
  onAddChannel: () => void
  onDeleteChannel: (channel: NotificationChannel) => void | Promise<void>
  onEditChannel: (channel: NotificationChannel) => void
  onTestChannel: (channel: NotificationChannel) => void | Promise<void>
  onToggleChannel: (channel: NotificationChannel, enabled: boolean) => void | Promise<void>
  testingChannelId: number | null
  updatingChannelId: number | null
}

export function NotificationChannelList({
  channels,
  onAddChannel,
  onDeleteChannel,
  onEditChannel,
  onTestChannel,
  onToggleChannel,
  testingChannelId,
  updatingChannelId,
}: Props) {
  const { t } = useTranslation()

  return (
    <Card>
      <CardContent className="space-y-3 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-medium">{t("settings.notifications.channels.title")}</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {t("settings.notifications.channels.description")}
            </p>
          </div>
          <Button size="sm" onClick={onAddChannel}>
            {t("settings.notifications.channels.addButton")}
          </Button>
        </div>

        {channels.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("settings.notifications.channels.empty")}</p>
        ) : (
          <div className="space-y-2">
            {channels.map((channel) => {
              const isTesting = testingChannelId === channel.id
              const isUpdating = updatingChannelId === channel.id
              return (
                <div
                  key={channel.id}
                  className={cn(
                    "flex flex-col gap-3 rounded-md border bg-card/70 p-3 sm:flex-row sm:items-center sm:justify-between",
                    !channel.enabled && "opacity-80"
                  )}
                >
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">
                        {t(`settings.notifications.channels.type.${channel.type}`)}
                      </p>
                      <Badge variant={channel.enabled ? "default" : "secondary"}>
                        {channel.enabled
                          ? t("settings.notifications.channels.enable")
                          : t("settings.notifications.channels.disable")}
                      </Badge>
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {t(`settings.notifications.channels.typeDescription.${channel.type}`)}
                    </p>
                  </div>

                  <div className="flex flex-wrap items-center gap-2">
                    <Switch
                      checked={channel.enabled}
                      disabled={isUpdating}
                      aria-label={
                        channel.enabled
                          ? t("settings.notifications.channels.disable")
                          : t("settings.notifications.channels.enable")
                      }
                      onCheckedChange={(checked) => void onToggleChannel(channel, checked)}
                    />
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={isTesting}
                      onClick={() => void onTestChannel(channel)}
                    >
                      {isTesting
                        ? t("settings.notifications.channels.testing")
                        : t("settings.notifications.channels.test")}
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => onEditChannel(channel)}>
                      {t("settings.notifications.channels.edit")}
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => void onDeleteChannel(channel)}>
                      {t("settings.notifications.channels.delete")}
                    </Button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
