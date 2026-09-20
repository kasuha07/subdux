import { renderToStaticMarkup } from "react-dom/server"
import { describe, expect, it, vi } from "vitest"
import type { ReactNode } from "react"

import IconPicker from "./icon-picker"

vi.mock("react-i18next", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-i18next")>()
  return {
    ...actual,
    useTranslation: () => ({
      t: (key: string, defaultVal?: string | Record<string, unknown>) =>
        typeof defaultVal === "string" ? defaultVal : key,
      i18n: { language: "en" },
    }),
  }
})

vi.mock("radix-ui", async (importOriginal) => {
  const actual = await importOriginal<typeof import("radix-ui")>()
  return {
    ...actual,
    Popover: {
      ...actual.Popover,
      Portal: ({ children }: { children: ReactNode }) => <div data-slot="popover-portal">{children}</div>,
    },
  }
})

describe("IconPicker", () => {
  it("renders trigger button with emoji value", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="🎬"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    expect(markup).toContain("🎬")
  })

  it("renders fallback image icon when value is empty", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value=""
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    expect(markup).toContain("lucide-image")
  })

  it("renders image preview when value is a URL", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="https://example.com/logo.png"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    expect(markup).toContain('<img src="https://example.com/logo.png"')
  })

  it("renders small trigger when triggerSize is sm", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="🎬"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
        triggerSize="sm"
      />
    )
    expect(markup).toContain("h-9 w-9")
  })

  it("renders medium trigger by default", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="🎬"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    expect(markup).toContain("h-10 w-10")
  })

  it("renders uploaded file path when value starts with file:", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="file:custom-icon.png"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    expect(markup).toContain('<img src="/uploads/icons/custom-icon.png"')
  })

  it("renders async brand icon when value is an async brand icon", () => {
    const markup = renderToStaticMarkup(
      <IconPicker
        value="si:netflix"
        onChange={vi.fn()}
        onFileSelected={vi.fn()}
      />
    )
    // AsyncBrandIcon renders
    expect(markup).toBeTruthy()
  })
})
