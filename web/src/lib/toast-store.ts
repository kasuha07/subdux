export type ToastId = string | number
export type ToastTheme = "light" | "dark" | "system"
export type ToastPosition =
  | "top-left"
  | "top-center"
  | "top-right"
  | "bottom-left"
  | "bottom-center"
  | "bottom-right"

export type ToastType = "default" | "success" | "error" | "info" | "warning" | "loading"

export interface ToastClassNames {
  toast?: string
  description?: string
  actionButton?: string
  cancelButton?: string
}

export interface ToastOptions {
  id?: ToastId
  description?: string
  duration?: number
  dismissible?: boolean
  className?: string
  classNames?: ToastClassNames
}

export interface ToastRecord extends ToastOptions {
  id: ToastId
  title: string
  type: ToastType
  visible: boolean
}

interface ToastCallable {
  (message: string, options?: ToastOptions): ToastId
}

interface ToastApi extends ToastCallable {
  dismiss: (id?: ToastId) => void
  success: (message: string, options?: ToastOptions) => ToastId
  error: (message: string, options?: ToastOptions) => ToastId
  info: (message: string, options?: ToastOptions) => ToastId
  warning: (message: string, options?: ToastOptions) => ToastId
  loading: (message: string, options?: ToastOptions) => ToastId
}

const DEFAULT_DURATION = 4000
const EXIT_DURATION_MS = 180
const listeners = new Set<() => void>()

let toastCounter = 0
let toasts: ToastRecord[] = []

function emitChange() {
  listeners.forEach((listener) => listener())
}

export function subscribe(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getSnapshot(): ToastRecord[] {
  return toasts
}

function removeToast(id: ToastId) {
  toasts = toasts.filter((toast) => toast.id !== id)
  emitChange()
}

export function dismissToast(id?: ToastId) {
  const targets = id === undefined ? toasts.map((toast) => toast.id) : [id]

  let changed = false
  toasts = toasts.map((toast) => {
    if (!targets.includes(toast.id) || !toast.visible) {
      return toast
    }

    changed = true
    return { ...toast, visible: false }
  })

  if (!changed) {
    return
  }

  emitChange()
  window.setTimeout(() => {
    targets.forEach(removeToast)
  }, EXIT_DURATION_MS)
}

function createToast(type: ToastType, title: string, options?: ToastOptions): ToastId {
  const id = options?.id ?? ++toastCounter
  const nextToast: ToastRecord = {
    id,
    type,
    title,
    description: options?.description,
    duration: options?.duration,
    dismissible: options?.dismissible ?? true,
    className: options?.className,
    classNames: options?.classNames,
    visible: true,
  }

  const existingIndex = toasts.findIndex((toast) => toast.id === id)
  if (existingIndex >= 0) {
    toasts = toasts.map((toast) => (toast.id === id ? nextToast : toast))
  } else {
    toasts = [nextToast, ...toasts]
  }

  emitChange()

  const duration = nextToast.duration ?? DEFAULT_DURATION
  if (duration > 0) {
    window.setTimeout(() => {
      dismissToast(id)
    }, duration)
  }

  return id
}

const toastBase: ToastCallable = (message, options) => createToast("default", message, options)

export const toast: ToastApi = Object.assign(toastBase, {
  dismiss: dismissToast,
  success: (message: string, options?: ToastOptions) => createToast("success", message, options),
  error: (message: string, options?: ToastOptions) => createToast("error", message, options),
  info: (message: string, options?: ToastOptions) => createToast("info", message, options),
  warning: (message: string, options?: ToastOptions) => createToast("warning", message, options),
  loading: (message: string, options?: ToastOptions) => createToast("loading", message, options),
})
