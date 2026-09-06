import { Suspense, lazy, type ReactNode, useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { BrowserRouter, Routes, Route, Navigate, useLocation } from "react-router"
import { Loader2 } from "lucide-react"
import { isAuthenticated, isAdmin, restoreSession } from "@/lib/api"
import { AppToaster } from "@/components/app-toaster"
import { TooltipProvider } from "@/components/ui/tooltip"
import { useSiteTitle } from "@/hooks/useSiteSettings"
import { scheduleNeighborRoutePreload, preloadRouteForPath } from "@/lib/route-preload"

const LoginPage = lazy(() => import("@/features/auth/login-page"))
const RegisterPage = lazy(() => import("@/features/auth/register-page"))
const ForgotPasswordPage = lazy(() => import("@/features/auth/forgot-password-page"))
const ResetPasswordPage = lazy(() => import("@/features/auth/reset-password-page"))
const DashboardPage = lazy(() => import("@/features/dashboard/dashboard-page"))
const ActionsPage = lazy(() => import("@/features/actions/actions-page"))
const SettingsPage = lazy(() => import("@/features/settings/settings-page"))
const AdminPage = lazy(() => import("@/features/admin/admin-page"))
const CalendarPage = lazy(() => import("@/features/calendar/calendar-page"))
const ReportsPage = lazy(() => import("@/features/reports/reports-page"))
const OIDCReauthCallback = lazy(() => import("@/features/auth/oidc-reauth-callback"))

function ProtectedRoute({ children, authReady }: { children: ReactNode, authReady: boolean }) {
  if (!authReady) {
    return <RouteLoading />
  }
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function PublicRoute({ children, authReady }: { children: ReactNode, authReady: boolean }) {
  if (!authReady) {
    return <RouteLoading />
  }
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function AdminRoute({ children, authReady }: { children: ReactNode, authReady: boolean }) {
  if (!authReady) {
    return <RouteLoading />
  }
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  if (!isAdmin()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function RouteLoading() {
  const { t } = useTranslation()
  return (
    <div role="status" className="flex min-h-screen items-center justify-center gap-2.5 px-4 text-sm text-muted-foreground">
      <Loader2 className="size-4 animate-spin text-muted-foreground" aria-hidden="true" />
      <span>{t("common.loading")}</span>
    </div>
  )
}

function LazyRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<RouteLoading />}>{children}</Suspense>
}

function RoutePreloader({ authReady }: { authReady: boolean }) {
  const location = useLocation()

  useEffect(() => {
    if (!authReady) return
    const path = location.pathname
    if ((path === "/" || ["/actions", "/reports", "/settings", "/calendar", "/admin"].includes(path)) && !isAuthenticated()) return
    if (path === "/admin" && !isAdmin()) return
    preloadRouteForPath(path)
    return scheduleNeighborRoutePreload(path)
  }, [location.pathname, authReady])

  return null
}

export default function App() {
  const [authReady, setAuthReady] = useState(() => isAuthenticated())
  useSiteTitle()

  useEffect(() => {
    let cancelled = false

    if (isAuthenticated()) {
      return () => {
        cancelled = true
      }
    }

    void restoreSession().finally(() => {
      if (!cancelled) {
        setAuthReady(true)
      }
    })

    return () => {
      cancelled = true
    }
  }, [])

  return (
    <BrowserRouter>
      <TooltipProvider delayDuration={200}>
        <AppToaster />
        <RoutePreloader authReady={authReady} />
        <Routes>
          <Route path="/login" element={<LazyRoute><PublicRoute authReady={authReady}><LoginPage /></PublicRoute></LazyRoute>} />
          <Route path="/register" element={<LazyRoute><PublicRoute authReady={authReady}><RegisterPage /></PublicRoute></LazyRoute>} />
          <Route path="/forgot-password" element={<LazyRoute><PublicRoute authReady={authReady}><ForgotPasswordPage /></PublicRoute></LazyRoute>} />
          <Route path="/reset-password" element={<LazyRoute><PublicRoute authReady={authReady}><ResetPasswordPage /></PublicRoute></LazyRoute>} />
          <Route path="/" element={<LazyRoute><ProtectedRoute authReady={authReady}><DashboardPage /></ProtectedRoute></LazyRoute>} />
          <Route path="/actions" element={<LazyRoute><ProtectedRoute authReady={authReady}><ActionsPage /></ProtectedRoute></LazyRoute>} />
          <Route path="/reports" element={<LazyRoute><ProtectedRoute authReady={authReady}><ReportsPage /></ProtectedRoute></LazyRoute>} />
          <Route path="/settings" element={<LazyRoute><ProtectedRoute authReady={authReady}><SettingsPage /></ProtectedRoute></LazyRoute>} />
          <Route path="/calendar" element={<LazyRoute><ProtectedRoute authReady={authReady}><CalendarPage /></ProtectedRoute></LazyRoute>} />
          <Route path="/admin" element={<LazyRoute><AdminRoute authReady={authReady}><AdminPage /></AdminRoute></LazyRoute>} />
          <Route path="/oidc/reauth" element={<LazyRoute><OIDCReauthCallback /></LazyRoute>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </TooltipProvider>
    </BrowserRouter>
  )
}
