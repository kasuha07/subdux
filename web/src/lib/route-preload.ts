const routeLoaders = {
  auth: {
    login: () => import("@/features/auth/login-page"),
    register: () => import("@/features/auth/register-page"),
    forgotPassword: () => import("@/features/auth/forgot-password-page"),
    resetPassword: () => import("@/features/auth/reset-password-page"),
  },
  protected: {
    dashboard: () => import("@/features/dashboard/dashboard-page"),
    actions: () => import("@/features/actions/actions-page"),
    reports: () => import("@/features/reports/reports-page"),
    settings: () => import("@/features/settings/settings-page"),
    calendar: () => import("@/features/calendar/calendar-page"),
    admin: () => import("@/features/admin/admin-page"),
  },
} as const

type RouteKind = keyof typeof routeLoaders

const routePreloadCache = new Set<string>()

export function preloadRoute(kind: RouteKind, route: string): void {
  const loader = (routeLoaders[kind] as Record<string, () => Promise<unknown>>)[route]
  if (!loader) {
    return
  }

  const key = `${kind}:${route}`
  if (routePreloadCache.has(key)) {
    return
  }

  routePreloadCache.add(key)
  void loader()
}

export function preloadRouteForPath(pathname: string): void {
  switch (pathname) {
    case "/":
      preloadRoute("protected", "dashboard")
      break
    case "/actions":
      preloadRoute("protected", "actions")
      break
    case "/reports":
      preloadRoute("protected", "reports")
      break
    case "/settings":
      preloadRoute("protected", "settings")
      break
    case "/calendar":
      preloadRoute("protected", "calendar")
      break
    case "/admin":
      preloadRoute("protected", "admin")
      break
    case "/login":
      preloadRoute("auth", "login")
      break
    case "/register":
      preloadRoute("auth", "register")
      break
    case "/forgot-password":
      preloadRoute("auth", "forgotPassword")
      break
    case "/reset-password":
      preloadRoute("auth", "resetPassword")
      break
    default:
      break
  }
}

export function preloadNeighborRoutesForPath(pathname: string): void {
  switch (pathname) {
    case "/":
      preloadRoute("protected", "actions")
      preloadRoute("protected", "calendar")
      preloadRoute("protected", "reports")
      break
    case "/login":
    case "/register":
    case "/forgot-password":
    case "/reset-password":
      preloadRoute("auth", "login")
      preloadRoute("auth", "register")
      preloadRoute("auth", "forgotPassword")
      preloadRoute("auth", "resetPassword")
      break
    default:
      break
  }
}
