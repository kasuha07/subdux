import { LoadErrorPresentation } from "@/components/load-error-boundary"
import type { LoadErrorCopy } from "@/lib/load-error"

export function ReportsLoadError({
  copy,
  onRetry,
}: {
  copy: LoadErrorCopy
  onRetry: () => void
}) {
  return (
    <LoadErrorPresentation
      className="rounded-lg border border-dashed p-8 text-center"
      copy={copy}
      descriptionClassName="mt-1 text-sm text-muted-foreground"
      onRetry={onRetry}
      titleClassName="font-medium"
    />
  )
}
