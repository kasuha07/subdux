import { Suspense, lazy, type ReactNode, useEffect, useState } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { isAuthenticated, isAdmin, restoreSession } from "@/lib/api"
import { useSiteSettings } from "@/hooks/useSiteSettings"
import { AppToaster } from "@/components/app-toaster"

const LoginPage = lazy(() => import("@/features/auth/login-page"))
const RegisterPage = lazy(() => import("@/features/auth/register-page"))
const ForgotPasswordPage = lazy(() => import("@/features/auth/forgot-password-page"))
const ResetPasswordPage = lazy(() => import("@/features/auth/reset-password-page"))
const DashboardPage = lazy(() => import("@/features/dashboard/dashboard-page"))
const SettingsPage = lazy(() => import("@/features/settings/settings-page"))
const AdminPage = lazy(() => import("@/features/admin/admin-page"))
const CalendarPage = lazy(() => import("@/features/calendar/calendar-page"))

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
  return (
    <div className="flex min-h-screen items-center justify-center px-4 text-sm text-muted-foreground">
      Loading...
    </div>
  )
}

function LazyRoute({ children }: { children: ReactNode }) {
  return <Suspense fallback={<RouteLoading />}>{children}</Suspense>
}

export default function App() {
  const [authReady, setAuthReady] = useState(() => isAuthenticated())

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

  useSiteSettings()

  return (
    <BrowserRouter>
      <AppToaster />
      <Routes>
        <Route path="/login" element={<LazyRoute><PublicRoute authReady={authReady}><LoginPage /></PublicRoute></LazyRoute>} />
        <Route path="/register" element={<LazyRoute><PublicRoute authReady={authReady}><RegisterPage /></PublicRoute></LazyRoute>} />
        <Route path="/forgot-password" element={<LazyRoute><PublicRoute authReady={authReady}><ForgotPasswordPage /></PublicRoute></LazyRoute>} />
        <Route path="/reset-password" element={<LazyRoute><PublicRoute authReady={authReady}><ResetPasswordPage /></PublicRoute></LazyRoute>} />
        <Route path="/" element={<LazyRoute><ProtectedRoute authReady={authReady}><DashboardPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/settings" element={<LazyRoute><ProtectedRoute authReady={authReady}><SettingsPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/calendar" element={<LazyRoute><ProtectedRoute authReady={authReady}><CalendarPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/admin" element={<LazyRoute><AdminRoute authReady={authReady}><AdminPage /></AdminRoute></LazyRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
