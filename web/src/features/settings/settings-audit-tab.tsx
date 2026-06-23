import { useCallback, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"

import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { TabsContent } from "@/components/ui/tabs"
import { api } from "@/lib/api"
import type { AuditEvent } from "@/types"

interface SettingsAuditTabProps {
  active: boolean
}

export default function SettingsAuditTab({ active }: SettingsAuditTabProps) {
  const { t } = useTranslation()
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(false)

  const loadEvents = useCallback(() => {
    setLoading(true)
    api
      .get<AuditEvent[]>("/audit-events?limit=50")
      .then((data) => setEvents(data || []))
      .catch(() => void 0)
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!active) {
      return
    }
    let cancelled = false
    api
      .get<AuditEvent[]>("/audit-events?limit=50")
      .then((data) => {
        if (!cancelled) {
          setEvents(data || [])
        }
      })
      .catch(() => void 0)
    return () => {
      cancelled = true
    }
  }, [active])

  return (
    <TabsContent value="audit">
      <div className="space-y-4">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold tracking-tight">{t("settings.audit.title")}</h2>
            <p className="mt-0.5 text-sm text-muted-foreground">{t("settings.audit.description")}</p>
          </div>
          <Button size="sm" variant="outline" onClick={loadEvents} disabled={loading}>
            {t("settings.audit.refresh")}
          </Button>
        </div>

        {!loading && events.length === 0 && (
          <div className="rounded-md border border-dashed px-4 py-8 text-center text-sm text-muted-foreground">
            {t("settings.audit.empty")}
          </div>
        )}

        <div className="space-y-2">
          {events.map((event) => (
            <AuditEventRow key={event.event_id} event={event} />
          ))}
        </div>
      </div>
    </TabsContent>
  )
}

function AuditEventRow({ event }: { event: AuditEvent }) {
  const { t } = useTranslation()

  return (
    <div className="rounded-md border px-3 py-2.5">
      <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
        <span className="text-sm font-medium">{event.tool_name}</span>
        <Badge variant={event.status === "success" ? "secondary" : "destructive"}>
          {event.status}
        </Badge>
        <span className="text-xs text-muted-foreground">
          {new Date(event.occurred_at).toLocaleString()}
        </span>
      </div>
      <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-muted-foreground">
        <span>{t("settings.audit.action")}: {event.action}</span>
        <span>{t("settings.audit.resource")}: {event.resource_type} #{event.resource_id}</span>
        {event.client_name && <span>{t("settings.audit.client")}: {event.client_name}</span>}
        {event.error && <span className="text-destructive">{event.error}</span>}
      </div>
      <details className="mt-2">
        <summary className="cursor-pointer text-xs text-muted-foreground">{t("settings.audit.details")}</summary>
        <pre className="mt-2 max-h-72 overflow-auto rounded-md bg-muted p-3 text-xs">
          {JSON.stringify({
            request_args_redacted: event.request_args_redacted,
            before_snapshot: event.before_snapshot,
            after_snapshot: event.after_snapshot,
          }, null, 2)}
        </pre>
      </details>
    </div>
  )
}
