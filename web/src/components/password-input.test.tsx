import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import { PasswordInput } from "./password-input"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}))

describe("PasswordInput", () => {
  it("renders a password input with an eye toggle button", () => {
    const markup = renderToStaticMarkup(
      <PasswordInput
        id="test-pwd"
        placeholder="••••••••"
        defaultValue="my-secret"
      />
    )

    expect(markup).toContain('type="password"')
    expect(markup).toContain('id="test-pwd"')
    expect(markup).toContain("common.showPassword")
    expect(markup).toContain("pr-9")
  })
})
