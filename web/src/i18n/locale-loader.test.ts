import { describe, expect, it } from "vitest"

// Importing this module runs i18n.init() at load time. In the node test
// environment `window`/`navigator` are undefined, so language detection falls
// back to "en" deterministically — which is exactly what these tests assert.
import i18n, { getInitialLanguage, preloadLocale } from "@/i18n"

function hasPath(value: unknown, path: string): boolean {
  return path.split(".").reduce<unknown>((node, key) => {
    if (node && typeof node === "object" && key in (node as Record<string, unknown>)) {
      return (node as Record<string, unknown>)[key]
    }
    return undefined
  }, value) !== undefined
}

describe("preloadLocale", () => {
  it("resolves the requested locale resource", async () => {
    const resource = await preloadLocale("en")
    expect(hasPath(resource, "subscription.form.notification.title")).toBe(true)
  })

  it("resolves zh-CN and ja resources", async () => {
    const [zh, ja] = await Promise.all([preloadLocale("zh-CN"), preloadLocale("ja")])
    expect(hasPath(zh, "subscription.form.notification.title")).toBe(true)
    expect(hasPath(ja, "subscription.form.notification.title")).toBe(true)
  })

  it("falls back to en for an unsupported language code", async () => {
    const fallback = await preloadLocale("fr-FR")
    const english = await preloadLocale("en")
    // Unsupported codes are normalized to the fallback, which is cached — so the
    // returned promise resolves to the very same resource object as "en".
    expect(fallback).toBe(english)
  })

  it("caches loaders so repeated calls share one in-flight promise", () => {
    const first = preloadLocale("en")
    const second = preloadLocale("en")
    expect(first).toBe(second)
  })

  it("does not throw on an empty or junk language string", async () => {
    await expect(preloadLocale("")).resolves.toBeDefined()
    await expect(preloadLocale("not-a-locale")).resolves.toBeDefined()
  })
})

describe("getInitialLanguage", () => {
  it("returns a supported locale code", () => {
    expect(["en", "zh-CN", "ja"]).toContain(getInitialLanguage())
  })

  it("defaults to en when no browser language is available", () => {
    // No window/navigator in the node env → detection fell back to en at init.
    expect(getInitialLanguage()).toBe("en")
  })

  it("reflects i18n.language but never leaks an unsupported value", () => {
    const result = getInitialLanguage()
    if (i18n.language === "zh-CN" || i18n.language === "ja") {
      expect(result).toBe(i18n.language)
    } else {
      expect(result).toBe("en")
    }
  })
})
