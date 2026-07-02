import { describe, expect, it } from "vitest"

import i18n, { i18nReady } from "@/i18n"
import { localizeBackendError } from "@/lib/api"

describe("localizeBackendError", () => {
  it("translates account rate-limit errors from the backend", async () => {
    await i18nReady
    await i18n.changeLanguage("en")

    expect(localizeBackendError("too many attempts for this account, please try again later")).toBe(
      "Too many attempts for this account. Please try again later."
    )
  })
})
