import type { BackupDestinationRunResult, BackupRunResponse } from "@/types"

export interface BackupRunSummary {
  failedDestinations: BackupDestinationRunResult[]
  retentionFailures: BackupDestinationRunResult[]
  bookkeepingFailures: BackupDestinationRunResult[]
  topLevelFailure: boolean
  globalBookkeepingFailure: boolean
}

export function summarizeBackupRun(response: BackupRunResponse = { file: "", results: [] }): BackupRunSummary {
  const results = response.results ?? []
  const statusFailure = (status?: string) => Boolean(status && status !== "success")
  return {
    failedDestinations: results.filter((result) => !result.success),
    retentionFailures: results.filter(
      (result) => result.success && result.retention_status === "failed"
    ),
    bookkeepingFailures: results.filter(
      (result) => result.bookkeeping_status !== "success"
    ),
    topLevelFailure:
      statusFailure(response.status) ||
      statusFailure(response.delivery_status) ||
      statusFailure(response.retention_status) ||
      statusFailure(response.bookkeeping_status) ||
      statusFailure(response.global_bookkeeping_status) ||
      Boolean(response.error),
    globalBookkeepingFailure: statusFailure(response.global_bookkeeping_status),
  }
}
