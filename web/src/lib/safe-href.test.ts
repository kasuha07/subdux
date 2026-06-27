import { describe, expect, it } from "vitest"

import { safeHref } from "@/lib/safe-href"

describe("safeHref", () => {
  it("returns undefined for nullish or empty input", () => {
    expect(safeHref(undefined)).toBeUndefined()
    expect(safeHref(null)).toBeUndefined()
    expect(safeHref("")).toBeUndefined()
    expect(safeHref("   ")).toBeUndefined()
  })

  it("accepts http and https URLs and normalises them", () => {
    expect(safeHref("https://example.com")).toBe("https://example.com/")
    expect(safeHref("http://example.com/path?q=1")).toBe("http://example.com/path?q=1")
  })

  it("trims surrounding whitespace before parsing", () => {
    expect(safeHref("  https://example.com  ")).toBe("https://example.com/")
  })

  it("rejects non-http(s) protocols", () => {
    expect(safeHref("javascript:alert(1)")).toBeUndefined()
    expect(safeHref("data:text/html,<script>")).toBeUndefined()
    expect(safeHref("ftp://example.com")).toBeUndefined()
    expect(safeHref("mailto:user@example.com")).toBeUndefined()
  })

  it("rejects strings containing interior control characters", () => {
    expect(safeHref("https://exa\tmple.com")).toBeUndefined()
    expect(safeHref("java\nscript:alert(1)")).toBeUndefined()
  })

  it("still trims surrounding control characters before validating", () => {
    // Leading/trailing whitespace (incl. newlines) is trimmed away, so the URL
    // remains valid once the surrounding characters are stripped.
    expect(safeHref("https://example.com\n")).toBe("https://example.com/")
  })

  it("rejects values that are not valid URLs", () => {
    expect(safeHref("not a url")).toBeUndefined()
    expect(safeHref("example.com")).toBeUndefined()
  })

  it("rejects http(s) URLs without a hostname", () => {
    expect(safeHref("https://")).toBeUndefined()
  })
})
