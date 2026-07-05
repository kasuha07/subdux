import { DismissableLayerBranch } from "@radix-ui/react-dismissable-layer"
import { createPortal } from "react-dom"
import { Toaster as SonnerToaster, type ToasterProps } from "sonner"
import { appToasterProps } from "@/lib/toast"
import { useTheme } from "@/lib/theme"

export function AppToaster() {
  const theme = useTheme()

  // Sonner renders its live region inline in the React tree instead of
  // portaling it. Wrap it in a Radix dismissable-layer Branch so interacting
  // with a toast (or its action button) is not treated as an outside
  // interaction that closes an open Dialog, and portal it to <body> so the
  // toaster stays a stable top-level sibling rather than a dialog-scoped child.
  return createPortal(
    <DismissableLayerBranch>
      <SonnerToaster
        theme={theme as ToasterProps["theme"]}
        {...appToasterProps}
      />
    </DismissableLayerBranch>,
    document.body
  )
}
