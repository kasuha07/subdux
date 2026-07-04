package api

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/api/apimw"
	mcpapi "github.com/kasuha07/subdux/internal/api/mcp"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/logging"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	auditservice "github.com/kasuha07/subdux/internal/service/audit"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	authreauth "github.com/kasuha07/subdux/internal/service/authreauth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	calendarservice "github.com/kasuha07/subdux/internal/service/calendar"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	exporter "github.com/kasuha07/subdux/internal/service/exporter"
	iconproxy "github.com/kasuha07/subdux/internal/service/iconproxy"
	importer "github.com/kasuha07/subdux/internal/service/importer"
	notificationservice "github.com/kasuha07/subdux/internal/service/notification"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func SetupRoutes(
	ctx context.Context,
	e *echo.Echo,
	db *gorm.DB,
	taskMonitor *serviceutil.BackgroundTaskMonitor,
) (*exchangerate.Service, *notificationservice.Service) {
	// Route all handler-returned errors through the single typed-error handler.
	// Handlers signal failures by returning *serviceerr.Error (or a wrapped
	// cause); the handler renders the frozen {"error": message} envelope with a
	// Kind-derived status. Echo's own HTTPErrors (jwt 401, 404, 405) are
	// delegated to the previous default so their behavior is unchanged.
	e.HTTPErrorHandler = APIErrorHandler(e.HTTPErrorHandler)

	authService := serviceauth.NewService(db)
	authService.StartSessionCleanupLoop(ctx)
	totpService := serviceauth.NewTOTPService(db)
	subService := subscriptionservice.NewService(db)
	adminService := adminservice.NewService(db)
	reauthService := servicereauth.NewService(db, authreauth.Adapt(authService))
	systemSettingsService := systemsettings.NewService(db)
	iconProxyService := iconproxy.NewService(db)
	erService := exchangerate.NewService(db)
	currencyService := catalogservice.NewCurrencyService(db)
	categoryService := catalogservice.NewCategoryService(db)
	paymentMethodService := catalogservice.NewPaymentMethodService(db)
	validator := notificationservice.NewTemplateValidator()
	renderer := notificationservice.NewTemplateRenderer(validator)
	templateService := notificationservice.NewNotificationTemplateService(db, validator)
	notificationService := notificationservice.NewService(db, templateService, renderer)
	apiKeyService := apikeyservice.NewService(db)
	auditService := auditservice.NewService(db)
	calendarService := calendarservice.NewService(db)
	exportService := exporter.NewService(db)
	importService := importer.NewService(db)
	if err := systemSettingsService.SeedDefaults(); err != nil {
		logging.Error("failed to seed default system settings", slog.Any("error", err))
	}

	authHandler := NewAuthHandler(authService, totpService, reauthService)
	subHandler := NewSubscriptionHandler(subService, erService)
	adminHandler := NewAdminHandler(adminService, taskMonitor, reauthService, servicebackup.NewService(db))
	reauthHandler := NewReauthHandler(reauthService)
	siteInfoHandler := NewSiteInfoHandler(systemSettingsService)
	iconProxyHandler := NewIconProxyHandler(iconProxyService)
	erHandler := NewExchangeRateHandler(erService)
	currencyHandler := NewCurrencyHandler(currencyService, erService)
	categoryHandler := NewCategoryHandler(categoryService)
	paymentMethodHandler := NewPaymentMethodHandler(paymentMethodService)
	dashboardBootstrapHandler := NewDashboardBootstrapHandler(subService, erService, currencyService, categoryService, paymentMethodService)
	notificationHandler := NewNotificationHandler(notificationService)
	templateHandler := NewNotificationTemplateHandler(templateService)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService, reauthService)
	auditHandler := NewAuditHandler(auditService)
	calendarHandler := NewCalendarHandler(calendarService)
	exportHandler := NewExportHandler(exportService, reauthService)
	importHandler := NewImportHandler(importService, reauthService)
	versionHandler := NewVersionHandler(db)
	mcpHandler := mcpapi.NewMCPHandler(apiKeyService, auditService, subService, erService, currencyService, categoryService, paymentMethodService)

	requireMCPEnabled := apimw.MCPEnabledMiddleware(systemSettingsService)
	e.POST("/mcp", mcpHandler.HandlePost, requireMCPEnabled, apimw.RequestBodyLimitMiddleware(1<<20, nil))
	e.GET("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PUT("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PATCH("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.DELETE("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)

	api := e.Group("/api")
	api.Use(apimw.RequestBodyLimitMiddleware(1<<20, func(c echo.Context) bool {
		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}
		return path == "/api/admin/restore" || path == "/api/import/wallos" || path == "/api/import/subdux"
	}))

	authIPLimiter := apimw.AuthIPRateLimit(30, time.Minute)
	loginAccountLimiter := apimw.AuthAccountRateLimit(10, time.Minute, apimw.LoginAccountKey)
	registerAccountLimiter := apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.RegisterAccountKey)
	passwordAccountLimiter := apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.EmailAccountKey)
	totpAccountLimiter := apimw.AuthAccountRateLimit(8, 5*time.Minute, apimw.TOTPAccountKey)
	refreshTokenLimiter := apimw.AuthAccountRateLimit(20, time.Minute, apimw.RefreshTokenAccountKey)
	iconProxyLimiter := apimw.AuthIPRateLimit(600, time.Minute)
	reauthIPLimiter := apimw.AuthIPRateLimit(30, time.Minute)
	reauthUserLimiter := apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.AuthenticatedUserAccountKey)
	emailChangeUserLimiter := apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.AuthenticatedUserAccountKey)

	api.GET("/version", versionHandler.Get)

	api.GET("/version/latest", versionHandler.GetLatest)

	api.GET("/icon-proxy/:provider", iconProxyHandler.Get, iconProxyLimiter)

	auth := api.Group("/auth")
	auth.Use(apimw.RequestBodyLimitMiddleware(apimw.MaxAuthRequestBodyBytes, nil))
	auth.GET("/register/config", authHandler.GetRegistrationConfig)
	auth.POST("/register/send-code", authHandler.SendRegisterVerificationCode, authIPLimiter, registerAccountLimiter)
	auth.POST("/register", authHandler.Register, authIPLimiter, registerAccountLimiter)
	auth.POST("/login", authHandler.Login, authIPLimiter, loginAccountLimiter)
	auth.POST("/password/forgot", authHandler.ForgotPassword, authIPLimiter, passwordAccountLimiter)
	auth.POST("/password/reset", authHandler.ResetPassword, authIPLimiter, passwordAccountLimiter)
	auth.POST("/totp/verify-login", authHandler.VerifyTOTPLogin, authIPLimiter, totpAccountLimiter)
	auth.POST("/refresh", authHandler.RefreshSession, authIPLimiter, refreshTokenLimiter)
	auth.POST("/refresh/logout", authHandler.Logout, authIPLimiter, refreshTokenLimiter)
	auth.POST("/passkeys/login/start", authHandler.BeginPasskeyLogin, authIPLimiter)
	auth.POST("/passkeys/login/finish", authHandler.FinishPasskeyLogin)
	auth.GET("/oidc/config", authHandler.GetOIDCConfig)
	auth.POST("/oidc/login/start", authHandler.BeginOIDCLogin, authIPLimiter)
	auth.GET("/oidc/callback", authHandler.OIDCCallback)
	auth.GET("/oidc/session", authHandler.GetOIDCSession)

	jwtConfig := echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}

	protected := api.Group("")
	protected.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	protected.Use(apimw.APIKeyScopeMiddleware)

	humanProtected := api.Group("")
	humanProtected.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	humanProtected.Use(apimw.HumanSessionOnlyMiddleware)
	humanProtected.Use(apimw.APIKeyScopeMiddleware)

	// Reauth ("step-up") is a generic capability, not admin-only: any human
	// session can prove presence and mint an operation-scoped ticket. Which
	// operations exist and which endpoints consume tickets is decided elsewhere.
	// It therefore lives on a human-session-only group mirroring humanProtected,
	// not under /admin.
	reauth := api.Group("/reauth")
	reauth.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	reauth.Use(apimw.HumanSessionOnlyMiddleware)
	reauth.Use(apimw.APIKeyScopeMiddleware)
	reauth.GET("/methods", reauthHandler.Methods)
	reauth.POST("/password", reauthHandler.VerifyPassword, reauthIPLimiter, reauthUserLimiter)
	reauth.POST("/passkey/start", reauthHandler.BeginPasskey, reauthIPLimiter, reauthUserLimiter)
	reauth.POST("/passkey/finish", reauthHandler.FinishPasskey)
	reauth.POST("/oidc/start", reauthHandler.BeginOIDC, reauthIPLimiter, reauthUserLimiter)
	reauth.POST("/oidc/finish", reauthHandler.FinishOIDC)

	protected.GET("/subscriptions", subHandler.List)
	protected.POST("/subscriptions", subHandler.Create)
	protected.GET("/subscriptions/:id/detail", subHandler.GetDetail)
	protected.GET("/subscriptions/:id", subHandler.GetByID)
	protected.PUT("/subscriptions/:id", subHandler.Update)
	protected.DELETE("/subscriptions/:id", subHandler.Delete)
	protected.POST("/subscriptions/:id/mark-renewed", subHandler.MarkRenewed)
	protected.POST("/subscriptions/reconcile", subHandler.Reconcile)
	protected.POST("/subscriptions/:id/icon", subHandler.UploadIcon)
	protected.GET("/dashboard/summary", subHandler.Dashboard)
	protected.GET("/dashboard/bootstrap", dashboardBootstrapHandler.Get)
	protected.GET("/actions", subHandler.ActionCenter)
	protected.POST("/actions/snooze", subHandler.SnoozeAction)
	protected.GET("/reports/analytics", subHandler.AnalyticsReport)

	protected.GET("/auth/me", authHandler.Me)
	humanProtected.POST("/auth/logout-all", authHandler.LogoutAll)
	humanProtected.PUT("/auth/password", authHandler.ChangePassword)
	humanProtected.POST("/auth/email/change/send-code", authHandler.SendEmailChangeVerificationCode, emailChangeUserLimiter)
	humanProtected.POST("/auth/email/change/confirm", authHandler.ConfirmEmailChange)
	humanProtected.POST("/auth/totp/setup", authHandler.SetupTOTP)
	humanProtected.POST("/auth/totp/confirm", authHandler.ConfirmTOTP)
	humanProtected.POST("/auth/totp/disable", authHandler.DisableTOTP)
	humanProtected.GET("/auth/passkeys", authHandler.ListPasskeys)
	humanProtected.POST("/auth/passkeys/register/start", authHandler.BeginPasskeyRegistration)
	humanProtected.POST("/auth/passkeys/register/finish", authHandler.FinishPasskeyRegistration)
	humanProtected.DELETE("/auth/passkeys/:id", authHandler.DeletePasskey)
	humanProtected.GET("/auth/oidc/connections", authHandler.ListOIDCConnections)
	humanProtected.POST("/auth/oidc/connect/start", authHandler.BeginOIDCConnect)
	humanProtected.DELETE("/auth/oidc/connections/:id", authHandler.DeleteOIDCConnection)
	admin := api.Group("/admin")

	admin.Use(echojwt.WithConfig(jwtConfig))
	admin.Use(apimw.AdminMiddleware)

	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)
	admin.PUT("/users/:id/role", adminHandler.ChangeUserRole)
	admin.PUT("/users/:id/status", adminHandler.ChangeUserStatus)
	admin.POST("/users/:id/disable-totp", adminHandler.DisableUserTOTP)
	admin.POST("/users/:id/disable-passkeys", adminHandler.DisableUserPasskeys)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.GET("/background-tasks", adminHandler.ListBackgroundTasks)
	admin.GET("/audit-events", auditHandler.ListAdminEvents)
	admin.GET("/settings", adminHandler.GetSettings)
	admin.PUT("/settings", adminHandler.UpdateSettings)
	admin.POST("/settings/ssrf/test", adminHandler.TestSSRF)
	admin.POST("/settings/smtp/test", adminHandler.TestSMTP)
	admin.POST("/backup", adminHandler.BackupDB)
	admin.POST("/backup/run", adminHandler.RunBackupNow)
	admin.GET("/backup/local", adminHandler.ListLocalBackups)
	admin.POST("/restore", adminHandler.RestoreDB, apimw.RequestBodyLimitMiddleware(32<<20, nil))
	admin.GET("/exchange-rates/status", erHandler.GetStatus)
	admin.POST("/exchange-rates/refresh", erHandler.RefreshRates)

	protected.GET("/exchange-rates", erHandler.ListRates)
	protected.GET("/exchange-rates/:base/:target", erHandler.GetRate)
	protected.GET("/preferences/currency", erHandler.GetPreference)
	protected.PUT("/preferences/currency", erHandler.UpdatePreference)

	protected.GET("/currencies", currencyHandler.List)
	protected.POST("/currencies", currencyHandler.Create)
	protected.PUT("/currencies/reorder", currencyHandler.Reorder)
	protected.PUT("/currencies/:id", currencyHandler.Update)
	protected.DELETE("/currencies/:id", currencyHandler.Delete)

	protected.GET("/categories", categoryHandler.List)
	protected.POST("/categories", categoryHandler.Create)
	protected.PUT("/categories/reorder", categoryHandler.Reorder)
	protected.PUT("/categories/:id", categoryHandler.Update)
	protected.DELETE("/categories/:id", categoryHandler.Delete)

	protected.GET("/payment-methods", paymentMethodHandler.List)
	protected.POST("/payment-methods", paymentMethodHandler.Create)
	protected.PUT("/payment-methods/reorder", paymentMethodHandler.Reorder)
	protected.PUT("/payment-methods/:id", paymentMethodHandler.Update)
	protected.DELETE("/payment-methods/:id", paymentMethodHandler.Delete)
	protected.POST("/payment-methods/:id/icon", paymentMethodHandler.UploadIcon)

	protected.GET("/notifications/channels", notificationHandler.ListChannels)
	protected.POST("/notifications/channels", notificationHandler.CreateChannel)
	protected.PUT("/notifications/channels/:id", notificationHandler.UpdateChannel)
	protected.DELETE("/notifications/channels/:id", notificationHandler.DeleteChannel)
	protected.POST("/notifications/channels/:id/test", notificationHandler.TestChannel)
	protected.GET("/notifications/policy", notificationHandler.GetPolicy)
	protected.PUT("/notifications/policy", notificationHandler.UpdatePolicy)
	protected.GET("/notifications/logs", notificationHandler.ListLogs)
	protected.GET("/notifications/templates", templateHandler.ListTemplates)
	protected.GET("/notifications/templates/:id", templateHandler.GetTemplate)
	protected.POST("/notifications/templates", templateHandler.CreateTemplate)
	protected.PUT("/notifications/templates/:id", templateHandler.UpdateTemplate)
	protected.DELETE("/notifications/templates/:id", templateHandler.DeleteTemplate)
	protected.POST("/notifications/templates/preview", templateHandler.PreviewTemplate)

	humanProtected.GET("/api-keys", apiKeyHandler.List)
	humanProtected.POST("/api-keys", apiKeyHandler.Create)
	humanProtected.DELETE("/api-keys/:id", apiKeyHandler.Delete)
	humanProtected.GET("/audit-events", auditHandler.ListUserEvents)

	humanProtected.GET("/calendar/tokens", calendarHandler.ListTokens)
	humanProtected.POST("/calendar/tokens", calendarHandler.CreateToken)
	humanProtected.DELETE("/calendar/tokens/:id", calendarHandler.DeleteToken)

	humanProtected.GET("/export", exportHandler.Export)
	humanProtected.POST("/import/subdux", importHandler.ImportSubdux, apimw.RequestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))
	protected.POST("/import/wallos", importHandler.ImportWallos, apimw.RequestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))

	api.GET("/calendar/feed", calendarHandler.GetCalendarFeed)

	api.GET("/site-info", siteInfoHandler.Get)

	return erService, notificationService
}
