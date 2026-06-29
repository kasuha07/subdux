import { describe, expect, it } from "vitest"

import en from "@/i18n/en"
import ja from "@/i18n/ja"
import zhCN from "@/i18n/zh-CN"

// Collects every leaf (non-object) key path, e.g. "subscription.form.notification.title".
function collectKeyPaths(value: unknown, prefix = "", out: string[] = []): string[] {
  if (value !== null && typeof value === "object" && !Array.isArray(value)) {
    for (const [key, child] of Object.entries(value as Record<string, unknown>)) {
      collectKeyPaths(child, prefix ? `${prefix}.${key}` : key, out)
    }
  } else {
    out.push(prefix)
  }
  return out
}

const locales = {
  en: new Set(collectKeyPaths(en)),
  ja: new Set(collectKeyPaths(ja)),
  "zh-CN": new Set(collectKeyPaths(zhCN)),
} as const

type LocaleName = keyof typeof locales

function missingFrom(reference: LocaleName, target: LocaleName): string[] {
  return [...locales[reference]].filter((key) => !locales[target].has(key)).sort()
}

describe("locale key parity", () => {
  it("defines a non-trivial set of keys", () => {
    expect(locales.en.size).toBeGreaterThan(1000)
  })

  // Each locale must expose exactly the same key paths. This directly guards the
  // class of regression this change risks: moving a key in one locale's file but
  // forgetting another, leaving raw keys rendered to some users.
  it("has identical key sets across en, ja, and zh-CN", () => {
    expect(missingFrom("en", "ja")).toEqual([])
    expect(missingFrom("ja", "en")).toEqual([])
    expect(missingFrom("en", "zh-CN")).toEqual([])
    expect(missingFrom("zh-CN", "en")).toEqual([])
  })

  it("keeps key counts in lockstep", () => {
    expect(locales.ja.size).toBe(locales.en.size)
    expect(locales["zh-CN"].size).toBe(locales.en.size)
  })
})

describe("subscription notification key migration", () => {
  const newKeys = [
    "subscription.form.notification.title",
    "subscription.form.notification.useDefault",
    "subscription.form.notification.enabled",
    "subscription.form.notification.disabled",
    "subscription.form.notification.daysBeforeOverride",
  ]

  const oldKeys = [
    "settings.notifications.subscription.title",
    "settings.notifications.subscription.useDefault",
    "settings.notifications.subscription.enabled",
    "settings.notifications.subscription.disabled",
    "settings.notifications.subscription.daysBeforeOverride",
  ]

  it("exposes the relocated notification keys in every locale", () => {
    for (const locale of Object.values(locales)) {
      for (const key of newKeys) {
        expect(locale.has(key)).toBe(true)
      }
    }
  })

  it("removes the old settings.notifications.subscription keys in every locale", () => {
    for (const locale of Object.values(locales)) {
      for (const key of oldKeys) {
        expect(locale.has(key)).toBe(false)
      }
    }
  })
})
