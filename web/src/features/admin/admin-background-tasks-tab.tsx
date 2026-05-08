import { useTranslation } from "react-i18next"
import { AlertCircle, CheckCircle2, Clock, Loader2, RefreshCw } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TabsContent } from "@/components/ui/tabs"
import type { BackgroundTask } from "@/types"

interface AdminBackgroundTasksTabProps {
  tasks: BackgroundTask[]
  refreshing: boolean
  onRefresh: () => void | Promise<void>
}

function formatDateTime(value: string | null, locale: string): string {
  if (!value) {
    return "—"
  }

  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return value
  }

  return parsed.toLocaleString(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  })
}

function formatInterval(seconds: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  if (seconds <= 0) {
    return "—"
  }
  if (seconds % 86400 === 0) {
    return t("admin.backgroundTasks.intervalDays", { count: seconds / 86400 })
  }
  if (seconds % 3600 === 0) {
    return t("admin.backgroundTasks.intervalHours", { count: seconds / 3600 })
  }
  if (seconds % 60 === 0) {
    return t("admin.backgroundTasks.intervalMinutes", { count: seconds / 60 })
  }
  return t("admin.backgroundTasks.intervalSeconds", { count: seconds })
}

function formatDuration(milliseconds: number, t: (key: string, options?: Record<string, unknown>) => string): string {
  if (milliseconds <= 0) {
    return "—"
  }
  if (milliseconds < 1000) {
    return t("admin.backgroundTasks.durationMs", { count: milliseconds })
  }
  return t("admin.backgroundTasks.durationSeconds", { count: Math.round(milliseconds / 1000) })
}

function statusVariant(status: BackgroundTask["status"]): "default" | "secondary" | "destructive" | "outline" {
  if (status === "running") {
    return "secondary"
  }
  if (status === "succeeded") {
    return "default"
  }
  if (status === "failed") {
    return "destructive"
  }
  return "outline"
}

function StatusIcon({ status }: { status: BackgroundTask["status"] }) {
  if (status === "running") {
    return <Loader2 className="size-4 animate-spin text-muted-foreground" />
  }
  if (status === "succeeded") {
    return <CheckCircle2 className="size-4 text-emerald-600" />
  }
  if (status === "failed") {
    return <AlertCircle className="size-4 text-destructive" />
  }
  return <Clock className="size-4 text-muted-foreground" />
}

export default function AdminBackgroundTasksTab({
  tasks,
  refreshing,
  onRefresh,
}: AdminBackgroundTasksTabProps) {
  const { t, i18n } = useTranslation()

  return (
    <TabsContent value="background-tasks" className="space-y-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold">{t("admin.backgroundTasks.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("admin.backgroundTasks.description")}
          </p>
        </div>
        <Button variant="outline" onClick={() => void onRefresh()} disabled={refreshing}>
          <RefreshCw className={`mr-2 size-4 ${refreshing ? "animate-spin" : ""}`} />
          {t("admin.backgroundTasks.refresh")}
        </Button>
      </div>

      {tasks.length === 0 ? (
        <div className="rounded-md border border-dashed px-4 py-8 text-sm text-muted-foreground">
          {t("admin.backgroundTasks.empty")}
        </div>
      ) : (
        <div className="grid gap-4">
          {tasks.map((task) => (
            <Card key={task.key}>
              <CardHeader>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div className="flex gap-3">
                    <div className="mt-1">
                      <StatusIcon status={task.status} />
                    </div>
                    <div>
                      <CardTitle className="text-base">{task.name}</CardTitle>
                      <CardDescription>{task.description}</CardDescription>
                    </div>
                  </div>
                  <Badge variant={statusVariant(task.status)}>
                    {t(`admin.backgroundTasks.status.${task.status}`)}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-4">
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.lastStarted")}
                    </p>
                    <p>{formatDateTime(task.last_started_at, i18n.language)}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.lastFinished")}
                    </p>
                    <p>{formatDateTime(task.last_finished_at, i18n.language)}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.nextRun")}
                    </p>
                    <p>{formatDateTime(task.next_run_at, i18n.language)}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.interval")}
                    </p>
                    <p>{formatInterval(task.interval_seconds, t)}</p>
                  </div>
                </div>

                <div className="grid gap-3 text-sm sm:grid-cols-3">
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.lastDuration")}
                    </p>
                    <p>{formatDuration(task.last_duration_ms, t)}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.successCount")}
                    </p>
                    <p className="tabular-nums">{task.success_count}</p>
                  </div>
                  <div>
                    <p className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                      {t("admin.backgroundTasks.failureCount")}
                    </p>
                    <p className="tabular-nums">{task.failure_count}</p>
                  </div>
                </div>

                {task.last_error && (
                  <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
                    <p className="font-medium">{t("admin.backgroundTasks.lastError")}</p>
                    <p className="mt-1 break-words">{task.last_error}</p>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </TabsContent>
  )
}
