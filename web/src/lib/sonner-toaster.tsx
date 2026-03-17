import { XIcon } from "lucide-react"
import { useSyncExternalStore } from "react"

import { cn } from "@/lib/utils"

import {
  dismissToast,
  getSnapshot,
  subscribe,
  type ToastPosition,
  type ToastRecord,
  type ToastTheme,
} from "./toast-store"

export interface ToasterProps {
  theme?: ToastTheme
  richColors?: boolean
  closeButton?: boolean
  position?: ToastPosition
  toastOptions?: {
    duration?: number
    className?: string
    classNames?: ToastRecord["classNames"]
  }
}

const POSITION_CLASS_NAMES: Record<ToastPosition, string> = {
  "top-left": "top-4 left-4 items-start",
  "top-center": "top-4 left-1/2 -translate-x-1/2 items-center",
  "top-right": "top-4 right-4 items-end",
  "bottom-left": "bottom-4 left-4 items-start",
  "bottom-center": "bottom-4 left-1/2 -translate-x-1/2 items-center",
  "bottom-right": "right-4 bottom-4 items-end",
}

function getToastTypeClassName(type: ToastRecord["type"], richColors: boolean): string {
  if (!richColors) {
    return "border-border bg-background text-foreground"
  }

  switch (type) {
    case "success":
      return "border-emerald-200 bg-emerald-50 text-emerald-950 dark:border-emerald-900/70 dark:bg-emerald-950/60 dark:text-emerald-100"
    case "error":
      return "border-destructive/30 bg-destructive/10 text-destructive dark:border-destructive/40 dark:bg-destructive/20"
    case "warning":
      return "border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900/70 dark:bg-amber-950/60 dark:text-amber-100"
    case "info":
      return "border-sky-200 bg-sky-50 text-sky-950 dark:border-sky-900/70 dark:bg-sky-950/60 dark:text-sky-100"
    case "loading":
      return "border-border bg-background text-foreground"
    default:
      return "border-border bg-background text-foreground"
  }
}

export function Toaster({
  closeButton = false,
  position = "top-right",
  richColors = false,
  theme,
  toastOptions,
}: ToasterProps) {
  const currentToasts = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

  return (
    <section
      aria-atomic="false"
      aria-live="polite"
      aria-relevant="additions text"
      className={cn(
        "toaster pointer-events-none fixed z-[100] flex max-h-screen w-full max-w-sm flex-col gap-2 px-4 sm:px-0",
        POSITION_CLASS_NAMES[position]
      )}
      data-app-toaster="true"
      data-sonner-toaster="true"
      data-theme={theme}
    >
      {currentToasts.map((item) => {
        const dismissible = item.dismissible ?? true

        return (
          <div
            key={item.id}
            className={cn(
              "ulw-toast pointer-events-auto w-full rounded-lg border shadow-lg",
              "transition-[opacity,transform,filter]",
              toastOptions?.className,
              toastOptions?.classNames?.toast,
              item.className,
              item.classNames?.toast,
              getToastTypeClassName(item.type, richColors)
            )}
            data-mounted={item.visible}
            data-removed={!item.visible}
            data-rich-colors={richColors}
            data-sonner-toast=""
            data-swipe-out="false"
            data-type={item.type}
            role="status"
          >
            <div className="flex items-start gap-3 p-4">
              <div className="min-w-0 flex-1">
                <p className="text-sm font-medium leading-5">{item.title}</p>
                {item.description && (
                  <p
                    className={cn(
                      "mt-1 text-sm text-muted-foreground",
                      toastOptions?.classNames?.description,
                      item.classNames?.description
                    )}
                  >
                    {item.description}
                  </p>
                )}
              </div>

              {closeButton && dismissible && (
                <button
                  aria-label="Close notification"
                  className="rounded-sm p-1 text-foreground/60 transition-colors hover:bg-black/5 hover:text-foreground focus:outline-none focus:ring-2 focus:ring-ring dark:hover:bg-white/10"
                  onClick={() => dismissToast(item.id)}
                  type="button"
                >
                  <XIcon className="size-4" />
                </button>
              )}
            </div>
          </div>
        )
      })}
    </section>
  )
}
