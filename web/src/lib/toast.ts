import type { ToasterProps } from "sonner"
import { toast as sonnerToast } from "sonner"

export const appToasterProps = {
  richColors: true,
  position: "top-right",
  expand: false,
  visibleToasts: 4,
  gap: 10,
  closeButton: true,
  toastOptions: {
    duration: 4000,
    classNames: {
      toast: "ulw-toast group toast font-sans text-sm border shadow-lg backdrop-blur-xs",
      title: "text-sm font-medium tracking-tight",
      description: "text-xs text-muted-foreground font-normal leading-relaxed mt-0.5",
      actionButton:
        "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground group-[.toast]:hover:bg-primary/90 group-[.toast]:transition-colors group-[.toast]:font-medium group-[.toast]:text-xs group-[.toast]:rounded-md group-[.toast]:h-7 group-[.toast]:px-2.5",
      cancelButton:
        "group-[.toast]:bg-secondary group-[.toast]:text-secondary-foreground group-[.toast]:hover:bg-secondary/80 group-[.toast]:transition-colors group-[.toast]:font-medium group-[.toast]:text-xs group-[.toast]:rounded-md group-[.toast]:h-7 group-[.toast]:px-2.5",
      closeButton:
        "ulw-toast-close !absolute !top-2 !right-2 !left-auto !translate-x-0 !translate-y-0 !transform-none !size-5 flex !items-center !justify-center !rounded-md !border !border-transparent !bg-transparent hover:!bg-destructive/10 !text-destructive hover:!text-destructive !opacity-0 group-hover:!opacity-100 hover:!opacity-100 focus-visible:!opacity-100 transition-all",
    },
  },
} satisfies Pick<
  ToasterProps,
  "position" | "richColors" | "expand" | "visibleToasts" | "gap" | "closeButton" | "toastOptions"
>

/**
 * Display an error toast with safe extraction of Error message or fallback.
 */
export function toastError(
  error: unknown,
  fallbackMessage?: string,
  options?: Parameters<typeof sonnerToast.error>[1]
): string | number {
  let message: string
  if (typeof error === "string" && error.trim().length > 0) {
    message = error
  } else if (error instanceof Error && error.message) {
    message = error.message
  } else if (fallbackMessage) {
    message = fallbackMessage
  } else {
    message = "An unexpected error occurred"
  }
  return sonnerToast.error(message, options)
}

export const toast = Object.assign(sonnerToast, {
  fromError: toastError,
})
