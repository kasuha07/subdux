import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import { SubscriptionIcon } from "./subscription-icon"

describe("SubscriptionIcon", () => {
  it("renders fallback initial letter when icon is empty", () => {
    const markup = renderToStaticMarkup(<SubscriptionIcon name="Netflix" icon="" />)
    expect(markup).toContain("N")
  })

  it("renders emoji icon directly", () => {
    const markup = renderToStaticMarkup(<SubscriptionIcon name="Movie" icon="🎬" />)
    expect(markup).toContain("🎬")
  })

  it("renders image tag for remote http url", () => {
    const markup = renderToStaticMarkup(
      <SubscriptionIcon name="Custom" icon="https://example.com/logo.png" />
    )
    expect(markup).toContain('<img src="https://example.com/logo.png"')
    expect(markup).toContain('alt="Custom"')
  })
})
