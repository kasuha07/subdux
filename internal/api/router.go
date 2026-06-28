package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/pkg/logging"
	"github.com/shiroha/subdux/internal/service"
	"github.com/shiroha/subdux/internal/version"
	"gorm.io/gorm"
)

func getUserID(c echo.Context) uint {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	return claims.UserID
}

func getUserRole(c echo.Context) string {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	if getAuthType(c) == pkg.AuthTypeAPIKey {
		// API keys are machine principals and should not be granted human role
		// privileges. Treat as least-privilege "user" for legacy role checks.
		return "user"
	}
	if strings.TrimSpace(claims.Role) == "" {
		return "user"
	}
	return claims.Role
}

func getAuthType(c echo.Context) string {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	if claims.AuthType != "" {
		return claims.AuthType
	}
	if len(claims.Scopes) > 0 {
		return pkg.AuthTypeAPIKey
	}
	return pkg.AuthTypeUser
}

func hasAPIKeyScope(c echo.Context, scope string) bool {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	for _, candidate := range claims.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

func getAPIKeyKind(c echo.Context) string {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	return service.NormalizePersistedAPIKeyKind(claims.KeyKind)
}

func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if getUserRole(c) != "admin" {
			return c.JSON(403, echo.Map{"error": "admin access required"})
		}
		return next(c)
	}
}

// JWTOrAPIKeyMiddleware accepts either a Bearer JWT token or an X-API-Key header.
// JWT is tried first; if no Authorization header is present, it falls back to API key.
func JWTOrAPIKeyMiddleware(jwtConfig echojwt.Config, apiKeyService *service.APIKeyService) echo.MiddlewareFunc {
	jwtMiddleware := echojwt.WithConfig(jwtConfig)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// If the request has an Authorization header, use JWT auth
			if c.Request().Header.Get("Authorization") != "" {
				return jwtMiddleware(next)(c)
			}

			// Otherwise, try API key
			key := c.Request().Header.Get("X-API-Key")
			if key == "" {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authorization required"})
			}

			principal, err := apiKeyService.WithContext(c.Request().Context()).ValidateKey(key)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
			}

			claims := &pkg.JWTClaims{
				UserID:   principal.UserID,
				AuthType: pkg.AuthTypeAPIKey,
				KeyID:    principal.KeyID,
				KeyKind:  principal.KeyKind,
				Scopes:   principal.Scopes,
			}
			token := &jwt.Token{Claims: claims}
			c.Set("user", token)

			return next(c)
		}
	}
}

func SetupRoutes(
	ctx context.Context,
	e *echo.Echo,
	db *gorm.DB,
	taskMonitor *service.BackgroundTaskMonitor,
) (*service.ExchangeRateService, *service.NotificationService) {
	authService := service.NewAuthService(db)
	authService.StartSessionCleanupLoop(ctx)
	totpService := service.NewTOTPService(db)
	subService := service.NewSubscriptionService(db)
	adminService := service.NewAdminService(db)
	systemSettingsService := service.NewSystemSettingsService(db)
	iconProxyService := service.NewIconProxyService(db)
	erService := service.NewExchangeRateService(db)
	currencyService := service.NewCurrencyService(db)
	categoryService := service.NewCategoryService(db)
	paymentMethodService := service.NewPaymentMethodService(db)
	validator := service.NewTemplateValidator()
	renderer := service.NewTemplateRenderer(validator)
	templateService := service.NewNotificationTemplateService(db, validator)
	notificationService := service.NewNotificationService(db, templateService, renderer)
	apiKeyService := service.NewAPIKeyService(db)
	auditService := service.NewAuditService(db)
	calendarService := service.NewCalendarService(db)
	exportService := service.NewExportService(db)
	importService := service.NewImportService(db)
	if err := systemSettingsService.SeedDefaults(); err != nil {
		logging.Error("failed to seed default system settings", slog.Any("error", err))
	}

	authHandler := NewAuthHandler(authService, totpService)
	subHandler := NewSubscriptionHandler(subService, erService)
	adminHandler := NewAdminHandler(adminService, taskMonitor)
	siteInfoHandler := NewSiteInfoHandler(systemSettingsService)
	iconProxyHandler := NewIconProxyHandler(iconProxyService)
	erHandler := NewExchangeRateHandler(erService)
	currencyHandler := NewCurrencyHandler(currencyService, erService)
	categoryHandler := NewCategoryHandler(categoryService)
	paymentMethodHandler := NewPaymentMethodHandler(paymentMethodService)
	dashboardBootstrapHandler := NewDashboardBootstrapHandler(subService, erService, currencyService, categoryService, paymentMethodService)
	notificationHandler := NewNotificationHandler(notificationService)
	templateHandler := NewNotificationTemplateHandler(templateService)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	auditHandler := NewAuditHandler(auditService)
	calendarHandler := NewCalendarHandler(calendarService)
	exportHandler := NewExportHandler(exportService)
	importHandler := NewImportHandler(importService)
	mcpHandler := NewMCPHandler(apiKeyService, auditService, subService, erService, currencyService, categoryService, paymentMethodService)

	requireMCPEnabled := mcpEnabledMiddleware(systemSettingsService)
	e.POST("/mcp", mcpHandler.HandlePost, requireMCPEnabled, requestBodyLimitMiddleware(1<<20, nil))
	e.GET("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PUT("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PATCH("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.DELETE("/mcp", mcpHandler.MethodNotAllowed, requireMCPEnabled)

	api := e.Group("/api")
	api.Use(requestBodyLimitMiddleware(1<<20, func(c echo.Context) bool {
		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}
		return path == "/api/admin/restore" || path == "/api/import/wallos" || path == "/api/import/subdux"
	}))

	authIPLimiter := authIPRateLimit(30, time.Minute)
	loginAccountLimiter := authAccountRateLimit(10, time.Minute, loginAccountKey)
	registerAccountLimiter := authAccountRateLimit(6, 10*time.Minute, registerAccountKey)
	passwordAccountLimiter := authAccountRateLimit(6, 10*time.Minute, emailAccountKey)
	totpAccountLimiter := authAccountRateLimit(8, 5*time.Minute, totpAccountKey)
	refreshTokenLimiter := authAccountRateLimit(20, time.Minute, refreshTokenAccountKey)
	iconProxyLimiter := authIPRateLimit(600, time.Minute)

	api.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, version.Get())
	})

	api.GET("/version/latest", func(c echo.Context) error {
		client := service.NewSafeOutboundHTTPClient(db, 10*time.Second)
		req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet,
			"https://api.github.com/repos/kasuha07/subdux/releases/latest", nil)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create request"})
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := client.Do(req)
		if err != nil {
			return c.JSON(http.StatusBadGateway, echo.Map{"error": "failed to fetch latest release"})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return c.JSON(http.StatusBadGateway, echo.Map{"error": "github api returned non-200"})
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to parse response"})
		}

		return c.JSON(http.StatusOK, echo.Map{"tag_name": release.TagName})
	})

	api.GET("/icon-proxy/:provider", iconProxyHandler.Get, iconProxyLimiter)

	auth := api.Group("/auth")
	auth.Use(requestBodyLimitMiddleware(maxAuthRequestBodyBytes, nil))
	auth.GET("/register/config", authHandler.GetRegistrationConfig)
	auth.POST("/register/send-code", authHandler.SendRegisterVerificationCode, authIPLimiter, registerAccountLimiter)
	auth.POST("/register", authHandler.Register, authIPLimiter, registerAccountLimiter)
	auth.POST("/login", authHandler.Login, authIPLimiter, loginAccountLimiter)
	auth.POST("/password/forgot", authHandler.ForgotPassword, authIPLimiter, passwordAccountLimiter)
	auth.POST("/password/reset", authHandler.ResetPassword, authIPLimiter, passwordAccountLimiter)
	auth.POST("/totp/verify-login", authHandler.VerifyTOTPLogin, authIPLimiter, totpAccountLimiter)
	auth.POST("/refresh", authHandler.RefreshSession, authIPLimiter, refreshTokenLimiter)
	auth.POST("/refresh/logout", authHandler.Logout, authIPLimiter, refreshTokenLimiter)
	auth.POST("/passkeys/login/start", authHandler.BeginPasskeyLogin)
	auth.POST("/passkeys/login/finish", authHandler.FinishPasskeyLogin)
	auth.GET("/oidc/config", authHandler.GetOIDCConfig)
	auth.POST("/oidc/login/start", authHandler.BeginOIDCLogin)
	auth.GET("/oidc/callback", authHandler.OIDCCallback)
	auth.GET("/oidc/session", authHandler.GetOIDCSession)

	jwtConfig := echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}

	protected := api.Group("")
	protected.Use(JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	protected.Use(APIKeyScopeMiddleware)

	humanProtected := api.Group("")
	humanProtected.Use(JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	humanProtected.Use(HumanSessionOnlyMiddleware)
	humanProtected.Use(APIKeyScopeMiddleware)

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
	humanProtected.PUT("/auth/password", authHandler.ChangePassword)
	humanProtected.POST("/auth/email/change/send-code", authHandler.SendEmailChangeVerificationCode)
	humanProtected.POST("/auth/email/change/confirm", authHandler.ConfirmEmailChange)
	humanProtected.GET("/auth/totp/setup", authHandler.SetupTOTP)
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
	admin.Use(AdminMiddleware)

	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)
	admin.PUT("/users/:id/role", adminHandler.ChangeUserRole)
	admin.PUT("/users/:id/status", adminHandler.ChangeUserStatus)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.GET("/stats", adminHandler.GetStats)
	admin.GET("/background-tasks", adminHandler.ListBackgroundTasks)
	admin.GET("/audit-events", auditHandler.ListAdminEvents)
	admin.GET("/settings", adminHandler.GetSettings)
	admin.PUT("/settings", adminHandler.UpdateSettings)
	admin.POST("/settings/smtp/test", adminHandler.TestSMTP)
	admin.GET("/backup", adminHandler.BackupDB)
	admin.POST("/restore", adminHandler.RestoreDB, requestBodyLimitMiddleware(32<<20, nil))
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
	protected.POST("/import/wallos", importHandler.ImportWallos, requestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))
	protected.POST("/import/subdux", importHandler.ImportSubdux, requestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))

	api.GET("/calendar/feed", calendarHandler.GetCalendarFeed)

	api.GET("/site-info", siteInfoHandler.Get)

	return erService, notificationService
}

func mcpEnabledMiddleware(settingsService *service.SystemSettingsService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			enabled, err := settingsService.IsMCPEnabled()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read mcp settings"})
			}
			if !enabled {
				return c.JSON(http.StatusNotFound, echo.Map{"error": "mcp is not enabled"})
			}
			return next(c)
		}
	}
}
