import { Suspense, lazy, type ReactNode } from "react"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { isAuthenticated, isAdmin } from "@/lib/api"
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

function ProtectedRoute({ children }: { children: ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function PublicRoute({ children }: { children: ReactNode }) {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function AdminRoute({ children }: { children: ReactNode }) {
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
  useSiteSettings()

  return (
    <BrowserRouter>
      <AppToaster />
      <Routes>
        <Route path="/login" element={<LazyRoute><PublicRoute><LoginPage /></PublicRoute></LazyRoute>} />
        <Route path="/register" element={<LazyRoute><PublicRoute><RegisterPage /></PublicRoute></LazyRoute>} />
        <Route path="/forgot-password" element={<LazyRoute><PublicRoute><ForgotPasswordPage /></PublicRoute></LazyRoute>} />
        <Route path="/reset-password" element={<LazyRoute><PublicRoute><ResetPasswordPage /></PublicRoute></LazyRoute>} />
        <Route path="/" element={<LazyRoute><ProtectedRoute><DashboardPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/settings" element={<LazyRoute><ProtectedRoute><SettingsPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/calendar" element={<LazyRoute><ProtectedRoute><CalendarPage /></ProtectedRoute></LazyRoute>} />
        <Route path="/admin" element={<LazyRoute><AdminRoute><AdminPage /></AdminRoute></LazyRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
