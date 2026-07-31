import { RefreshCw } from "lucide-react"

import { LoadErrorPresentation } from "@/components/load-error-state"
import type { LoadErrorCopy } from "@/lib/load-error"

export function DashboardLoadError({
  copy,
  onRetry,
}: {
  copy: LoadErrorCopy
  onRetry: () => void
}) {
  return (
    <LoadErrorPresentation
      className="flex flex-col items-center justify-center rounded-lg border border-dashed py-16 text-center"
      copy={copy}
      descriptionClassName="mt-1 max-w-md text-sm text-muted-foreground"
      icon={
        <div className="mb-4 rounded-full bg-muted p-4">
          <RefreshCw className="size-6 text-muted-foreground" />
        </div>
      }
      onRetry={onRetry}
      titleClassName="font-medium"
      titleElement="h2"
    />
  )
}
