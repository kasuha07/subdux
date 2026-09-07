import { describe, expect, it, vi } from "vitest"
import { appToasterProps, toast, toastError } from "./toast"

describe("toast helper", () => {
  it("exports standard toast functions and fromError", () => {
    expect(typeof toast).toBe("function")
    expect(typeof toast.success).toBe("function")
    expect(typeof toast.error).toBe("function")
    expect(typeof toast.info).toBe("function")
    expect(typeof toast.warning).toBe("function")
    expect(typeof toast.loading).toBe("function")
    expect(typeof toast.promise).toBe("function")
    expect(typeof toast.dismiss).toBe("function")
    expect(typeof toast.fromError).toBe("function")
  })

  it("extracts error message from string in toastError", () => {
    const spy = vi.spyOn(toast, "error").mockReturnValue("toast-id")
    toastError("Direct error message")
    expect(spy).toHaveBeenCalledWith("Direct error message", undefined)
    spy.mockRestore()
  })

  it("extracts error message from Error instance in toastError", () => {
    const spy = vi.spyOn(toast, "error").mockReturnValue("toast-id")
    toastError(new Error("Something went wrong"))
    expect(spy).toHaveBeenCalledWith("Something went wrong", undefined)
    spy.mockRestore()
  })

  it("uses fallbackMessage when error is unknown or empty", () => {
    const spy = vi.spyOn(toast, "error").mockReturnValue("toast-id")
    toastError(null, "Failed to load")
    expect(spy).toHaveBeenCalledWith("Failed to load", undefined)

    toastError("", "Custom fallback")
    expect(spy).toHaveBeenCalledWith("Custom fallback", undefined)
    spy.mockRestore()
  })

  it("provides well-formed appToasterProps", () => {
    expect(appToasterProps.richColors).toBe(true)
    expect(appToasterProps.position).toBe("top-right")
    expect(appToasterProps.closeButton).toBe(true)
    expect(appToasterProps.visibleToasts).toBe(4)
    expect(appToasterProps.gap).toBe(10)
    expect(appToasterProps.expand).toBe(false)
    expect(appToasterProps.toastOptions?.duration).toBe(4000)
    expect(appToasterProps.toastOptions?.classNames?.toast).toContain("ulw-toast")
    expect(appToasterProps.toastOptions?.classNames?.closeButton).toBeDefined()
  })
})
