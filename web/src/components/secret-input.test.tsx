import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"

import { SecretInput } from "./secret-input"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { language: "en" },
  }),
}))

describe("SecretInput", () => {
  it("renders masked value when configured and not editing", () => {
    const markup = renderToStaticMarkup(
      <SecretInput
        configured={true}
        value=""
        onValueChange={() => void 0}
        type="password"
      />
    )

    expect(markup).toContain('value="••••••••"')
    expect(markup).toContain('type="password"')
    expect(markup).toContain("common.showPassword")
  })

  it("renders plain input without eye toggle for non-password types", () => {
    const markup = renderToStaticMarkup(
      <SecretInput
        configured={false}
        value="plain-secret"
        onValueChange={() => void 0}
        type="text"
      />
    )

    expect(markup).toContain('value="plain-secret"')
    expect(markup).toContain('type="text"')
    expect(markup).not.toContain("common.showPassword")
  })
})
