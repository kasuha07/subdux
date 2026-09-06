import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import { CopyButton } from "./copy-button"

describe("CopyButton", () => {
  it("renders a copy button with accessible labels", () => {
    const markup = renderToStaticMarkup(
      <CopyButton text="my-secret-token" label="Copy token" copiedLabel="Copied token" />
    )

    expect(markup).toContain('type="button"')
    expect(markup).toContain('aria-label="Copy token"')
    expect(markup).toContain('title="Copy token"')
    expect(markup).toContain("lucide-copy")
  })
})
