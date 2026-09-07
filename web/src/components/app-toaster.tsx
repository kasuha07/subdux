import { DismissableLayerBranch } from "@radix-ui/react-dismissable-layer"
import { createPortal } from "react-dom"
import { Toaster as SonnerToaster, type ToasterProps } from "sonner"
import {
  AlertCircle,
  AlertTriangle,
  CheckCircle2,
  Info,
  Loader2,
  X,
} from "lucide-react"
import { appToasterProps } from "@/lib/toast"
import { useTheme } from "@/lib/theme"

const appToasterIcons: NonNullable<ToasterProps["icons"]> = {
  success: (
    <CheckCircle2
      className="size-4 shrink-0 text-emerald-600 dark:text-emerald-400"
      aria-hidden="true"
    />
  ),
  info: (
    <Info
      className="size-4 shrink-0 text-sky-600 dark:text-sky-400"
      aria-hidden="true"
    />
  ),
  warning: (
    <AlertTriangle
      className="size-4 shrink-0 text-amber-600 dark:text-amber-400"
      aria-hidden="true"
    />
  ),
  error: (
    <AlertCircle
      className="size-4 shrink-0 text-rose-600 dark:text-rose-400"
      aria-hidden="true"
    />
  ),
  loading: (
    <Loader2
      className="size-4 shrink-0 animate-spin text-muted-foreground"
      aria-hidden="true"
    />
  ),
  close: <X className="size-3.5" aria-hidden="true" />,
}

export function AppToaster() {
  const theme = useTheme()

  // Sonner renders its live region inline in the React tree instead of
  // portaling it. Wrap it in a Radix dismissable-layer Branch so interacting
  // with a toast (or its action button) is not treated as an outside
  // interaction that closes an open Dialog, and portal it to <body> so the
  // toaster stays a stable top-level sibling rather than a dialog-scoped child.
  if (typeof document === "undefined") {
    return null
  }

  return createPortal(
    <DismissableLayerBranch>
      <SonnerToaster
        theme={theme as ToasterProps["theme"]}
        icons={appToasterIcons}
        {...appToasterProps}
      />
    </DismissableLayerBranch>,
    document.body
  )
}
