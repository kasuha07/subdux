import type { SubscriptionBatchInput } from "@/types"

export function createActivateBatchInput(ids: number[]): SubscriptionBatchInput {
  return {
    action: "update",
    ids,
    status: "active",
  }
}
