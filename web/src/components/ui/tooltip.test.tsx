import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it } from "vitest"

import {
  Tooltip,
  TooltipProvider,
  TooltipTrigger,
  SimpleTooltip,
} from "./tooltip"

describe("Tooltip component", () => {
  it("renders trigger element with compound components", () => {
    const markup = renderToStaticMarkup(
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <button type="button">Trigger</button>
          </TooltipTrigger>
        </Tooltip>
      </TooltipProvider>
    )

    expect(markup).toContain('data-slot="tooltip-trigger"')
    expect(markup).toContain("Trigger")
  })

  it("renders trigger element with convenient content prop", () => {
    const markup = renderToStaticMarkup(
      <TooltipProvider>
        <Tooltip content="Quick Help">
          <button type="button">Hover Me</button>
        </Tooltip>
      </TooltipProvider>
    )

    expect(markup).toContain('data-slot="tooltip-trigger"')
    expect(markup).toContain("Hover Me")
  })

  it("passes through children directly when content is falsy", () => {
    const markup = renderToStaticMarkup(
      <TooltipProvider>
        <Tooltip content={null}>
          <button type="button">Raw Button</button>
        </Tooltip>
      </TooltipProvider>
    )

    expect(markup).not.toContain('data-slot="tooltip-trigger"')
    expect(markup).toContain("Raw Button")
  })

  it("supports SimpleTooltip alias", () => {
    const markup = renderToStaticMarkup(
      <TooltipProvider>
        <SimpleTooltip content="Simple Help">
          <button type="button">Alias Button</button>
        </SimpleTooltip>
      </TooltipProvider>
    )

    expect(markup).toContain('data-slot="tooltip-trigger"')
    expect(markup).toContain("Alias Button")
  })

  it("preserves trigger accessibility attributes and class names", () => {
    const markup = renderToStaticMarkup(
      <TooltipProvider>
        <Tooltip content="Help">
          <button type="button" className="btn-custom" aria-label="custom-label">
            Accessible Button
          </button>
        </Tooltip>
      </TooltipProvider>
    )

    expect(markup).toContain('class="btn-custom"')
    expect(markup).toContain('aria-label="custom-label"')
    expect(markup).toContain("Accessible Button")
  })
})
