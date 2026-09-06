import { renderToStaticMarkup } from "react-dom/server"
import { MemoryRouter, Routes, Route } from "react-router"
import { describe, expect, it } from "vitest"

import { PageTransition } from "./page-transition"

describe("PageTransition", () => {
  it("renders children with page-transition class and path attribute", () => {
    const markup = renderToStaticMarkup(
      <MemoryRouter initialEntries={["/actions"]}>
        <Routes>
          <Route
            path="/actions"
            element={
              <PageTransition>
                <div data-testid="page-content">Actions Content</div>
              </PageTransition>
            }
          />
        </Routes>
      </MemoryRouter>
    )

    expect(markup).toContain('class="page-transition flex min-h-screen flex-col"')
    expect(markup).toContain('data-page-path="/actions"')
    expect(markup).toContain("Actions Content")
  })
})
