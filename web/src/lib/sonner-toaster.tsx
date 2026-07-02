import { Branch as DismissableLayerBranch } from "@radix-ui/react-dismissable-layer"
import { type KeyboardEvent, useSyncExternalStore } from "react"
import { createPortal } from "react-dom"

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
      return "border-emerald-200 bg-emerald-50 text-emerald-950 dark:border-emerald-900 dark:bg-emerald-950 dark:text-emerald-100"
    case "error":
      return "border-red-200 bg-red-50 text-red-700 dark:border-red-700 dark:bg-red-950 dark:text-red-200"
    case "warning":
      return "border-amber-200 bg-amber-50 text-amber-950 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100"
    case "info":
      return "border-sky-200 bg-sky-50 text-sky-950 dark:border-sky-900 dark:bg-sky-950 dark:text-sky-100"
    case "loading":
      return "border-border bg-background text-foreground"
    default:
      return "border-border bg-background text-foreground"
  }
}

export function Toaster({
  position = "top-right",
  richColors = false,
  theme,
  toastOptions,
}: ToasterProps) {
  const currentToasts = useSyncExternalStore(subscribe, getSnapshot, getSnapshot)

  // Portal to <body> so the live region stays outside #root and remains an
  // aria-live target that Radix Dialog's hideOthers() keeps available.
  // Branch prevents toast clicks from being treated as outside dialog input.
  return createPortal(
    <DismissableLayerBranch asChild>
      <section
        aria-atomic="false"
        aria-live="polite"
        aria-relevant="additions text"
        className={cn(
          "toaster pointer-events-none fixed z-[2147483647] flex max-h-screen w-80 max-w-[calc(100vw-2rem)] flex-col gap-2",
          POSITION_CLASS_NAMES[position]
        )}
        data-app-toaster="true"
        data-sonner-toaster="true"
        data-theme={theme}
      >
        {currentToasts.map((item) => {
          const dismissible = item.dismissible ?? true
          const handleDismissKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
            if (!dismissible || (event.key !== "Enter" && event.key !== " ")) {
              return
            }

            event.preventDefault()
            dismissToast(item.id)
          }

          return (
            <div
              key={item.id}
              className={cn(
                "ulw-toast pointer-events-auto w-full rounded-md border shadow-none",
                "transition-[opacity,transform]",
                dismissible && "cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
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
              onClick={dismissible ? () => dismissToast(item.id) : undefined}
              onKeyDown={handleDismissKeyDown}
              role="status"
              tabIndex={dismissible ? 0 : undefined}
            >
              <div className="flex items-start gap-3 px-4 py-3">
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
              </div>
            </div>
          )
        })}
      </section>
    </DismissableLayerBranch>,
    document.body
  )
}
