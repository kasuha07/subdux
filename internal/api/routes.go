package api

import "github.com/kasuha07/subdux/internal/api/apimw"

// RegisterRoutes wires the build-version endpoints (public, unauthenticated).
func (h *VersionHandler) RegisterRoutes(g RouteGroups) {
	g.Public.GET("/version", h.Get)
	g.Public.GET("/version/latest", h.GetLatest)
}

// RegisterRoutes wires the icon-proxy endpoint (public, IP rate limited).
func (h *IconProxyHandler) RegisterRoutes(g RouteGroups) {
	g.Public.GET("/icon-proxy/:provider", h.Get, g.Limiters.iconProxy)
}

// RegisterRoutes wires the public site-info endpoint.
func (h *SiteInfoHandler) RegisterRoutes(g RouteGroups) {
	g.Public.GET("/site-info", h.Get)
}

// RegisterRoutes wires the auth endpoints. Unauthenticated flows land on the
// Auth group (strict body limit + per-flow rate limits); "me" is a Protected
// read; the remaining account-management endpoints require a human session.
func (h *AuthHandler) RegisterRoutes(g RouteGroups) {
	l := g.Limiters

	g.Auth.GET("/register/config", h.GetRegistrationConfig)
	g.Auth.POST("/register/send-code", h.SendRegisterVerificationCode, l.authIP, l.registerAccount)
	g.Auth.POST("/register", h.Register, l.authIP, l.registerAccount)
	g.Auth.POST("/login", h.Login, l.authIP, l.loginAccount)
	g.Auth.POST("/password/forgot", h.ForgotPassword, l.authIP, l.passwordAccount)
	g.Auth.POST("/password/reset", h.ResetPassword, l.authIP, l.passwordAccount)
	g.Auth.POST("/totp/verify-login", h.VerifyTOTPLogin, l.authIP, l.totpAccount)
	g.Auth.POST("/refresh", h.RefreshSession, l.authIP, l.refreshToken)
	g.Auth.POST("/refresh/logout", h.Logout, l.authIP, l.refreshToken)
	g.Auth.POST("/passkeys/login/start", h.BeginPasskeyLogin, l.authIP)
	g.Auth.POST("/passkeys/login/finish", h.FinishPasskeyLogin)
	g.Auth.GET("/oidc/config", h.GetOIDCConfig)
	g.Auth.POST("/oidc/login/start", h.BeginOIDCLogin, l.authIP)
	g.Auth.GET("/oidc/callback", h.OIDCCallback)
	g.Auth.GET("/oidc/session", h.GetOIDCSession)

	g.Protected.GET("/auth/me", h.Me)

	g.HumanProtected.POST("/auth/logout-all", h.LogoutAll)
	g.HumanProtected.PUT("/auth/password", h.ChangePassword)
	g.HumanProtected.POST("/auth/email/change/send-code", h.SendEmailChangeVerificationCode, l.emailChangeUser)
	g.HumanProtected.POST("/auth/email/change/confirm", h.ConfirmEmailChange)
	g.HumanProtected.POST("/auth/totp/setup", h.SetupTOTP)
	g.HumanProtected.POST("/auth/totp/confirm", h.ConfirmTOTP)
	g.HumanProtected.POST("/auth/totp/disable", h.DisableTOTP)
	g.HumanProtected.GET("/auth/passkeys", h.ListPasskeys)
	g.HumanProtected.POST("/auth/passkeys/register/start", h.BeginPasskeyRegistration)
	g.HumanProtected.POST("/auth/passkeys/register/finish", h.FinishPasskeyRegistration)
	g.HumanProtected.DELETE("/auth/passkeys/:id", h.DeletePasskey)
	g.HumanProtected.GET("/auth/oidc/connections", h.ListOIDCConnections)
	g.HumanProtected.POST("/auth/oidc/connect/start", h.BeginOIDCConnect)
	g.HumanProtected.DELETE("/auth/oidc/connections/:id", h.DeleteOIDCConnection)
}

// RegisterRoutes wires the step-up re-authentication endpoints.
func (h *ReauthHandler) RegisterRoutes(g RouteGroups) {
	l := g.Limiters
	g.Reauth.GET("/methods", h.Methods)
	g.Reauth.POST("/password", h.VerifyPassword, l.reauthIP, l.reauthUser)
	g.Reauth.POST("/passkey/start", h.BeginPasskey, l.reauthIP, l.reauthUser)
	g.Reauth.POST("/passkey/finish", h.FinishPasskey)
	g.Reauth.POST("/oidc/start", h.BeginOIDC, l.reauthIP, l.reauthUser)
	g.Reauth.POST("/oidc/finish", h.FinishOIDC)
}

// RegisterRoutes wires subscriptions, the dashboard summary, action center, and
// analytics report.
func (h *SubscriptionHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/subscriptions", h.List)
	g.Protected.POST("/subscriptions", h.Create)
	g.Protected.GET("/subscriptions/:id/detail", h.GetDetail)
	g.Protected.GET("/subscriptions/:id", h.GetByID)
	g.Protected.PUT("/subscriptions/:id", h.Update)
	g.Protected.DELETE("/subscriptions/:id", h.Delete)
	g.Protected.POST("/subscriptions/:id/mark-renewed", h.MarkRenewed)
	g.Protected.POST("/subscriptions/batch", h.Batch)
	g.Protected.POST("/subscriptions/reconcile", h.Reconcile)
	g.Protected.POST("/subscriptions/:id/icon", h.UploadIcon)
	g.Protected.GET("/dashboard/summary", h.Dashboard)
	g.Protected.GET("/actions", h.ActionCenter)
	g.Protected.POST("/actions/snooze", h.SnoozeAction)
	g.Protected.GET("/reports/analytics", h.AnalyticsReport)
}

// RegisterRoutes wires the dashboard bootstrap aggregate endpoint.
func (h *DashboardBootstrapHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/dashboard/bootstrap", h.Get)
}

// RegisterRoutes wires exchange-rate reads, currency preference, and the
// admin-only rate refresh/status endpoints.
func (h *ExchangeRateHandler) RegisterRoutes(g RouteGroups) {
	g.Admin.GET("/exchange-rates/status", h.GetStatus)
	g.Admin.POST("/exchange-rates/refresh", h.RefreshRates)

	g.Protected.GET("/exchange-rates", h.ListRates)
	g.Protected.GET("/exchange-rates/:base/:target", h.GetRate)
	g.Protected.GET("/preferences/currency", h.GetPreference)
	g.Protected.PUT("/preferences/currency", h.UpdatePreference)
}

// RegisterRoutes wires the user currency catalog.
func (h *CurrencyHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/currencies", h.List)
	g.Protected.POST("/currencies", h.Create)
	g.Protected.PUT("/currencies/reorder", h.Reorder)
	g.Protected.PUT("/currencies/:id", h.Update)
	g.Protected.DELETE("/currencies/:id", h.Delete)
}

// RegisterRoutes wires the category catalog.
func (h *CategoryHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/categories", h.List)
	g.Protected.POST("/categories", h.Create)
	g.Protected.PUT("/categories/reorder", h.Reorder)
	g.Protected.PUT("/categories/:id", h.Update)
	g.Protected.DELETE("/categories/:id", h.Delete)
}

// RegisterRoutes wires the payment-method catalog.
func (h *PaymentMethodHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/payment-methods", h.List)
	g.Protected.POST("/payment-methods", h.Create)
	g.Protected.PUT("/payment-methods/reorder", h.Reorder)
	g.Protected.PUT("/payment-methods/:id", h.Update)
	g.Protected.DELETE("/payment-methods/:id", h.Delete)
	g.Protected.POST("/payment-methods/:id/icon", h.UploadIcon)
}

// RegisterRoutes wires notification channels, policy, and logs.
func (h *NotificationHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/notifications/channels", h.ListChannels)
	g.Protected.POST("/notifications/channels", h.CreateChannel)
	g.Protected.PUT("/notifications/channels/:id", h.UpdateChannel)
	g.Protected.DELETE("/notifications/channels/:id", h.DeleteChannel)
	g.Protected.POST("/notifications/channels/:id/test", h.TestChannel)
	g.Protected.GET("/notifications/policy", h.GetPolicy)
	g.Protected.PUT("/notifications/policy", h.UpdatePolicy)
	g.Protected.GET("/notifications/logs", h.ListLogs)
}

// RegisterRoutes wires notification templates.
func (h *NotificationTemplateHandler) RegisterRoutes(g RouteGroups) {
	g.Protected.GET("/notifications/templates", h.ListTemplates)
	g.Protected.GET("/notifications/templates/:id", h.GetTemplate)
	g.Protected.POST("/notifications/templates", h.CreateTemplate)
	g.Protected.PUT("/notifications/templates/:id", h.UpdateTemplate)
	g.Protected.DELETE("/notifications/templates/:id", h.DeleteTemplate)
	g.Protected.POST("/notifications/templates/preview", h.PreviewTemplate)
}

// RegisterRoutes wires API-key management (human-session only).
func (h *APIKeyHandler) RegisterRoutes(g RouteGroups) {
	g.HumanProtected.GET("/api-keys", h.List)
	g.HumanProtected.POST("/api-keys", h.Create)
	g.HumanProtected.DELETE("/api-keys/:id", h.Delete)
}

// RegisterRoutes wires audit-event reads: admin sees all, users see their own.
func (h *AuditHandler) RegisterRoutes(g RouteGroups) {
	g.Admin.GET("/audit-events", h.ListAdminEvents)
	g.HumanProtected.GET("/audit-events", h.ListUserEvents)
}

// RegisterRoutes wires calendar token management (human-session only) and the
// public ICS feed.
func (h *CalendarHandler) RegisterRoutes(g RouteGroups) {
	g.HumanProtected.GET("/calendar/tokens", h.ListTokens)
	g.HumanProtected.POST("/calendar/tokens", h.CreateToken)
	g.HumanProtected.DELETE("/calendar/tokens/:id", h.DeleteToken)
	g.Public.GET("/calendar/feed", h.GetCalendarFeed)
}

// RegisterRoutes wires the data export endpoint (human-session only).
func (h *ExportHandler) RegisterRoutes(g RouteGroups) {
	g.HumanProtected.GET("/export", h.Export)
}

// RegisterRoutes wires the data import endpoints. The Subdux import is a
// human-only account operation; the Wallos import is allowed for API-key
// principals too. Both raise the body limit for the upload payload.
func (h *ImportHandler) RegisterRoutes(g RouteGroups) {
	g.HumanProtected.POST("/import/subdux", h.ImportSubdux, apimw.RequestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))
	g.Protected.POST("/import/wallos", h.ImportWallos, apimw.RequestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))
}

// RegisterRoutes wires the admin console: user management, settings, backup and
// restore. Restore raises the body limit for the uploaded archive.
func (h *AdminHandler) RegisterRoutes(g RouteGroups) {
	g.Admin.GET("/users", h.ListUsers)
	g.Admin.POST("/users", h.CreateUser)
	g.Admin.PUT("/users/:id/role", h.ChangeUserRole)
	g.Admin.PUT("/users/:id/status", h.ChangeUserStatus)
	g.Admin.POST("/users/:id/disable-totp", h.DisableUserTOTP)
	g.Admin.POST("/users/:id/disable-passkeys", h.DisableUserPasskeys)
	g.Admin.DELETE("/users/:id", h.DeleteUser)
	g.Admin.GET("/background-tasks", h.ListBackgroundTasks)
	g.Admin.GET("/settings", h.GetSettings)
	g.Admin.PUT("/settings", h.UpdateSettings)
	g.Admin.POST("/settings/ssrf/test", h.TestSSRF)
	g.Admin.POST("/settings/smtp/test", h.TestSMTP)
	g.Admin.POST("/backup", h.BackupDB)
	g.Admin.GET("/backup/runs", h.ListBackupRuns)
	g.Admin.GET("/backup/destinations", h.ListBackupDestinations)
	g.Admin.POST("/backup/destinations", h.CreateBackupDestination)
	g.Admin.POST("/backup/destinations/test", h.TestBackupDestinationConfig)
	g.Admin.PUT("/backup/destinations/:id", h.UpdateBackupDestination)
	g.Admin.DELETE("/backup/destinations/:id", h.DeleteBackupDestination)
	g.Admin.POST("/backup/destinations/:id/test", h.TestBackupDestination)
	g.Admin.POST("/backup/destinations/:id/run", h.RunBackupDestination)
	g.Admin.GET("/backup/destinations/:id/backups", h.ListBackupDestinationBackups)
	g.Admin.POST("/backup/destinations/:id/restore", h.RestoreBackupDestination)
	g.Admin.POST("/restore", h.RestoreDB, apimw.RequestBodyLimitMiddleware(32<<20, nil))
}
