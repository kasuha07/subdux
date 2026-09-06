import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import { ActionsSkeleton } from "./actions-page"

describe("ActionsSkeleton", () => {
  it("renders with page loading animation and staggered card enter styles", () => {
    const markup = renderToStaticMarkup(<ActionsSkeleton />)

    expect(markup).toContain("page-loading-enter")
    expect(markup).toContain("subscription-card-enter")
    expect(markup).toContain("grid-cols-3")
    expect(markup).toContain("animate-pulse")
    expect(markup).toContain("--card-delay")
  })
})
