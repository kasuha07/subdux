import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { isAuthenticated, isAdmin } from "@/lib/api"
import { useSiteSettings } from "@/hooks/useSiteSettings"
import { AppToaster } from "@/components/app-toaster"
import LoginPage from "@/features/auth/login-page"
import RegisterPage from "@/features/auth/register-page"
import ForgotPasswordPage from "@/features/auth/forgot-password-page"
import ResetPasswordPage from "@/features/auth/reset-password-page"
import DashboardPage from "@/features/dashboard/dashboard-page"
import SettingsPage from "@/features/settings/settings-page"
import AdminPage from "@/features/admin/admin-page"

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  if (isAuthenticated()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  if (!isAdmin()) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}

export default function App() {
  useSiteSettings()

  return (
    <BrowserRouter>
      <AppToaster />
      <Routes>
        <Route path="/login" element={<PublicRoute><LoginPage /></PublicRoute>} />
        <Route path="/register" element={<PublicRoute><RegisterPage /></PublicRoute>} />
        <Route path="/forgot-password" element={<PublicRoute><ForgotPasswordPage /></PublicRoute>} />
        <Route path="/reset-password" element={<PublicRoute><ResetPasswordPage /></PublicRoute>} />
        <Route path="/" element={<ProtectedRoute><DashboardPage /></ProtectedRoute>} />
        <Route path="/settings" element={<ProtectedRoute><SettingsPage /></ProtectedRoute>} />
        <Route path="/admin" element={<AdminRoute><AdminPage /></AdminRoute>} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
