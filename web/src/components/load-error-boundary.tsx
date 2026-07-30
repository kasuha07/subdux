import { RefreshCw } from "lucide-react"
import type { ReactNode } from "react"

import { Button } from "@/components/ui/button"
import type { LoadErrorCopy } from "@/lib/load-error"

interface LoadErrorPresentationProps {
  className: string
  copy: LoadErrorCopy
  descriptionClassName: string
  icon?: ReactNode
  onRetry: () => void
  titleClassName: string
  titleElement?: "h2" | "p"
}

export function LoadErrorPresentation({
  className,
  copy,
  descriptionClassName,
  icon,
  onRetry,
  titleClassName,
  titleElement = "p",
}: LoadErrorPresentationProps) {
  const Title = titleElement

  return (
    <div className={className}>
      {icon}
      <Title className={titleClassName}>{copy.title}</Title>
      <p className={descriptionClassName}>{copy.description}</p>
      <Button className="mt-4" onClick={onRetry}>
        <RefreshCw className="size-4" />
        {copy.retry}
      </Button>
    </div>
  )
}

export function LoadErrorBoundary({
  children,
  error,
  fallback,
}: {
  children: ReactNode
  error: unknown
  fallback: ReactNode
}) {
  return error ? fallback : <>{children}</>
}
