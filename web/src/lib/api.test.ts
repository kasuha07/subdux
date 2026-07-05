import { beforeEach, describe, expect, it, vi } from "vitest"

const { toastError } = vi.hoisted(() => ({
  toastError: vi.fn(),
}))

vi.mock("@/lib/toast", () => ({
  toast: {
    error: toastError,
  },
}))

import i18n, { i18nReady } from "@/i18n"
import { api, BackendAPIError, localizeBackendError, localizeBackendMessage } from "@/lib/api"

beforeEach(() => {
  toastError.mockReset()
  vi.restoreAllMocks()
})

describe("localizeBackendError", () => {
  it("translates account rate-limit errors from the backend", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    expect(localizeBackendError("too_many_attempts_for_this_account_please_try_again_later")).toBe(
      "Too many attempts for this account. Please try again later."
    )
  })

  it("interpolates oidc callback errors by code in non-english locales", async () => {
    await i18nReady
    await i18n.changeLanguage("zh-CN")

    expect(localizeBackendError("oidc_userinfo_endpoint_returned_status", { status: 404 })).toBe(
      "oidc 用户信息端点返回 404"
    )
  })

  it("hides passkey decode internals behind the stable translation key", async () => {
    await i18nReady
    await i18n.changeLanguage("ja")

    expect(localizeBackendError("failed_to_decode_passkey", { detail: "raw decoder error" })).toBe(
      "パスキーをデコードできませんでした"
    )
  })

  it("falls back to the generic request error when a code is missing", async () => {
    await i18nReady
    await i18n.changeLanguage("zh-CN")

    expect(localizeBackendError(undefined)).toBe("请求失败")
  })

  it("interpolates backend message params by code", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    expect(localizeBackendMessage("backup_file_is_too_large_max_mb", { max_mb: 32 })).toBe(
      "backup file is too large (max 32 MB)"
    )
  })

  it("keeps request errors silent unless the caller opts into a toast", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          error_code: "too_many_attempts_for_this_account_please_try_again_later",
        }),
        {
          status: 429,
          headers: { "Content-Type": "application/json" },
        }
      )
    )

    const error = await api.post("/test", {}).catch((err: unknown) => err)

    expect(error).toBeInstanceOf(BackendAPIError)
    expect(error).toMatchObject({
      message: "Too many attempts for this account. Please try again later.",
      code: "too_many_attempts_for_this_account_please_try_again_later",
      status: 429,
    })
    expect(toastError).not.toHaveBeenCalled()
  })

  it("shows a request error toast when the caller opts in explicitly", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          error_code: "too_many_attempts_for_this_account_please_try_again_later",
        }),
        {
          status: 429,
          headers: { "Content-Type": "application/json" },
        }
      )
    )

    const error = await api.get("/test", { errorHandling: "toast" }).catch((err: unknown) => err)

    expect(error).toBeInstanceOf(BackendAPIError)
    expect(error).toMatchObject({
      message: "Too many attempts for this account. Please try again later.",
      code: "too_many_attempts_for_this_account_please_try_again_later",
      status: 429,
    })
    expect(toastError).toHaveBeenCalledWith(
      "Too many attempts for this account. Please try again later."
    )
  })

  it("keeps malformed success payloads typed as BackendAPIError", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("ok", {
        status: 200,
        headers: { "Content-Type": "text/plain" },
      })
    )

    const error = await api.get("/test").catch((err: unknown) => err)

    expect(error).toBeInstanceOf(BackendAPIError)
    expect(error).toMatchObject({
      message: i18n.t("common.requestFailed"),
      status: 200,
    })
    expect(toastError).not.toHaveBeenCalled()
  })

  it("throws localized backend errors for raw response requests", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(
        JSON.stringify({
          error_code: "too_many_attempts_for_this_account_please_try_again_later",
        }),
        {
          status: 429,
          headers: { "Content-Type": "application/json" },
        }
      )
    )

    const error = await api.response("/test").catch((err: unknown) => err)

    expect(error).toBeInstanceOf(BackendAPIError)
    expect(error).toMatchObject({
      message: "Too many attempts for this account. Please try again later.",
      code: "too_many_attempts_for_this_account_please_try_again_later",
      status: 429,
    })
    expect(toastError).not.toHaveBeenCalled()
  })

  it("uses the caller fallback key for raw response errors without a JSON backend payload", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("boom", {
        status: 500,
        headers: { "Content-Type": "text/plain" },
      })
    )

    const error = await api
      .response("/test", { errorFallbackKey: "settings.account.exportFailed" })
      .catch((err: unknown) => err)

    expect(error).toBeInstanceOf(BackendAPIError)
    expect(error).toMatchObject({
      message: i18n.t("settings.account.exportFailed"),
      status: 500,
    })
    expect(toastError).not.toHaveBeenCalled()
  })

  it("preserves successful raw response bodies for download and stream callers", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response("ok", {
        status: 200,
        headers: { "Content-Type": "text/plain" },
      })
    )

    const response = await api.response("/test")

    expect(response).toBeInstanceOf(Response)
    expect(await response.text()).toBe("ok")
    expect(toastError).not.toHaveBeenCalled()
  })
})
