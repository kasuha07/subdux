import { describe, expect, it } from "vitest"

import type { BackupDestinationRunResult, BackupRunResponse } from "@/types"

import { summarizeBackupRun } from "./backup-run"

function result(overrides: Partial<BackupDestinationRunResult> = {}): BackupDestinationRunResult {
  return {
    destination_id: 1,
    type: "local",
    delivery_status: "success",
    success: true,
    retention_status: "success",
    bookkeeping_status: "success",
    ...overrides,
  }
}

describe("summarizeBackupRun", () => {
  it("separates delivery, retention, and bookkeeping warnings", () => {
    const deliveryFailure = result({ destination_id: 2, success: false, error: "upload failed" })
    const retentionFailure = result({
      destination_id: 3,
      retention_status: "failed",
      retention_error: "cleanup failed",
    })
    const bookkeepingFailure = result({
      destination_id: 4,
      bookkeeping_status: "failed",
      bookkeeping_error: "status persistence failed",
    })

    const summary = summarizeBackupRun({
      file: "backup.zip",
      status: "partial",
      results: [result(), deliveryFailure, retentionFailure, bookkeepingFailure],
    })

    expect(summary.failedDestinations).toEqual([deliveryFailure])
    expect(summary.retentionFailures).toEqual([retentionFailure])
    expect(summary.bookkeepingFailures).toEqual([bookkeepingFailure])
    expect(summary.topLevelFailure).toBe(true)
    expect(summary.globalBookkeepingFailure).toBe(false)
  })

  it("does not report retention as a second failure when delivery failed", () => {
    const failed = result({ success: false, retention_status: "failed" })

    expect(summarizeBackupRun({ file: "backup.zip", results: [failed] })).toEqual({
      failedDestinations: [failed],
      retentionFailures: [],
      bookkeepingFailures: [],
      topLevelFailure: false,
      globalBookkeepingFailure: false,
    })
  })

  it.each([
    "partial",
    "failed",
  ])("marks a top-level %s response as unsuccessful even without destination failures", (status) => {
    const response: BackupRunResponse = { file: "backup.zip", status, results: [result()] }
    expect(summarizeBackupRun(response).topLevelFailure).toBe(true)
  })

  it("marks bookkeeping and global bookkeeping errors as unsuccessful", () => {
    expect(
      summarizeBackupRun({
        file: "backup.zip",
        bookkeeping_status: "failed",
        global_bookkeeping_status: "failed",
        global_bookkeeping_error: "could not persist run",
        results: [result()],
      })
    ).toMatchObject({ topLevelFailure: true, globalBookkeepingFailure: true })
  })

  it("marks a top-level error as unsuccessful", () => {
    expect(summarizeBackupRun({ file: "backup.zip", error: "archive cleanup failed" }).topLevelFailure).toBe(true)
  })
})
