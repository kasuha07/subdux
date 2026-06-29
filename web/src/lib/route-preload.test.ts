import { beforeEach, describe, expect, it, vi } from "vitest"

// Records every feature-page module that actually gets dynamically imported.
const loaded: string[] = []

// Maps the real import specifier to the label we record when it loads.
const FEATURE_MODULES: ReadonlyArray<readonly [string, string]> = [
  ["@/features/auth/login-page", "auth/login"],
  ["@/features/auth/register-page", "auth/register"],
  ["@/features/auth/forgot-password-page", "auth/forgotPassword"],
  ["@/features/auth/reset-password-page", "auth/resetPassword"],
  ["@/features/dashboard/dashboard-page", "protected/dashboard"],
  ["@/features/actions/actions-page", "protected/actions"],
  ["@/features/reports/reports-page", "protected/reports"],
  ["@/features/settings/settings-page", "protected/settings"],
  ["@/features/calendar/calendar-page", "protected/calendar"],
  ["@/features/admin/admin-page", "protected/admin"],
]

// Flush microtasks/macrotasks so fire-and-forget `void import(...)` calls resolve.
async function flush() {
  await new Promise((resolve) => setTimeout(resolve, 0))
}

type RoutePreload = typeof import("@/lib/route-preload")

// `vi.doMock` (unlike `vi.mock`) is not hoisted and re-runs its factory after
// each `vi.resetModules()`, giving real per-test isolation: a fresh module-scope
// preload cache AND a fresh recording of which chunks were imported.
async function freshModule(): Promise<RoutePreload> {
  vi.resetModules()
  loaded.length = 0
  for (const [specifier, label] of FEATURE_MODULES) {
    vi.doMock(specifier, () => {
      loaded.push(label)
      return { default: () => null }
    })
  }
  return import("@/lib/route-preload")
}

describe("preloadRouteForPath", () => {
  let mod: RoutePreload

  beforeEach(async () => {
    mod = await freshModule()
  })

  it.each([
    ["/", "protected/dashboard"],
    ["/actions", "protected/actions"],
    ["/reports", "protected/reports"],
    ["/settings", "protected/settings"],
    ["/calendar", "protected/calendar"],
    ["/admin", "protected/admin"],
    ["/login", "auth/login"],
    ["/register", "auth/register"],
    ["/forgot-password", "auth/forgotPassword"],
    ["/reset-password", "auth/resetPassword"],
  ])("preloads the chunk for %s", async (pathname, expected) => {
    mod.preloadRouteForPath(pathname)
    await flush()
    expect(loaded).toEqual([expected])
  })

  it("does nothing for an unknown path", async () => {
    mod.preloadRouteForPath("/does-not-exist")
    mod.preloadRouteForPath("/subscriptions/42")
    await flush()
    expect(loaded).toEqual([])
  })

  it("deduplicates repeated preloads of the same route", async () => {
    mod.preloadRouteForPath("/settings")
    mod.preloadRouteForPath("/settings")
    mod.preloadRouteForPath("/settings")
    await flush()
    expect(loaded).toEqual(["protected/settings"])
  })
})

describe("preloadRoute", () => {
  let mod: RoutePreload

  beforeEach(async () => {
    mod = await freshModule()
  })

  it("preloads a valid kind/route pair", async () => {
    mod.preloadRoute("protected", "dashboard")
    await flush()
    expect(loaded).toEqual(["protected/dashboard"])
  })

  it("ignores an unknown route name without throwing", async () => {
    expect(() => mod.preloadRoute("protected", "nonexistent")).not.toThrow()
    await flush()
    expect(loaded).toEqual([])
  })

  it("ignores a route name from the wrong kind bucket", async () => {
    // "dashboard" is a protected route, not an auth route.
    mod.preloadRoute("auth", "dashboard")
    await flush()
    expect(loaded).toEqual([])
  })
})

describe("preloadNeighborRoutesForPath", () => {
  let mod: RoutePreload

  beforeEach(async () => {
    mod = await freshModule()
  })

  it("preloads sibling protected routes from the dashboard", async () => {
    mod.preloadNeighborRoutesForPath("/")
    await flush()
    expect([...loaded].sort()).toEqual(
      ["protected/actions", "protected/calendar", "protected/reports"].sort()
    )
  })

  it.each(["/login", "/register", "/forgot-password", "/reset-password"])(
    "preloads the full auth cluster from %s",
    async (pathname) => {
      mod.preloadNeighborRoutesForPath(pathname)
      await flush()
      expect([...loaded].sort()).toEqual(
        ["auth/forgotPassword", "auth/login", "auth/register", "auth/resetPassword"].sort()
      )
    }
  )

  it("preloads no neighbors for paths without a defined cluster", async () => {
    mod.preloadNeighborRoutesForPath("/settings")
    mod.preloadNeighborRoutesForPath("/admin")
    await flush()
    expect(loaded).toEqual([])
  })

  it("does not re-load a route already preloaded directly", async () => {
    mod.preloadRouteForPath("/actions")
    await flush()
    expect(loaded).toEqual(["protected/actions"])

    // Dashboard neighbors include /actions, which is already cached.
    mod.preloadNeighborRoutesForPath("/")
    await flush()
    // /actions appears only once overall — proof the dedup cache held.
    expect(loaded.filter((entry) => entry === "protected/actions")).toHaveLength(1)
    expect([...loaded].sort()).toEqual(
      ["protected/actions", "protected/calendar", "protected/reports"].sort()
    )
  })
})
