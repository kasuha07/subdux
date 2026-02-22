import { useTranslation } from "react-i18next"

import { Badge } from "@/components/ui/badge"
import { formatDate } from "@/lib/utils"
import type { NotificationLog } from "@/types"

interface Props {
  logs: NotificationLog[]
}

export function NotificationLogList({ logs }: Props) {
  const { t, i18n } = useTranslation()

  return (
    <div className="space-y-3">
      <div>
        <h2 className="text-base font-semibold tracking-tight">{t("settings.notifications.logs.title")}</h2>
        <p className="mt-0.5 text-sm text-muted-foreground">
          {t("settings.notifications.logs.description")}
        </p>
      </div>

      {logs.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("settings.notifications.logs.empty")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th className="pb-2 pr-3 font-medium">{t("settings.notifications.logs.channel")}</th>
                <th className="pb-2 pr-3 font-medium">{t("settings.notifications.logs.date")}</th>
                <th className="pb-2 pr-3 font-medium">{t("settings.notifications.logs.status")}</th>
                <th className="pb-2 font-medium">{t("settings.notifications.logs.sentAt")}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id} className="border-b last:border-0">
                  <td className="py-2 pr-3">{log.channel_type}</td>
                  <td className="py-2 pr-3">{formatDate(log.notify_date, i18n.language)}</td>
                  <td className="py-2 pr-3">
                    <Badge variant={log.status === "sent" ? "default" : "destructive"}>
                      {log.status === "sent"
                        ? t("settings.notifications.logs.statusSent")
                        : t("settings.notifications.logs.statusFailed")}
                    </Badge>
                  </td>
                  <td className="py-2">{formatDate(log.sent_at, i18n.language)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
