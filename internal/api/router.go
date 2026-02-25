package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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

			principal, err := apiKeyService.ValidateKey(key)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
			}

			claims := &pkg.JWTClaims{
				UserID:   principal.UserID,
				AuthType: pkg.AuthTypeAPIKey,
				Scopes:   principal.Scopes,
			}
			token := &jwt.Token{Claims: claims}
			c.Set("user", token)

			return next(c)
		}
	}
}

func SetupRoutes(e *echo.Echo, db *gorm.DB) (*service.ExchangeRateService, *service.NotificationService) {
	authService := service.NewAuthService(db)
	totpService := service.NewTOTPService(db)
	subService := service.NewSubscriptionService(db)
	adminService := service.NewAdminService(db)
	erService := service.NewExchangeRateService(db)
	currencyService := service.NewCurrencyService(db)
	categoryService := service.NewCategoryService(db)
	paymentMethodService := service.NewPaymentMethodService(db)
	validator := service.NewTemplateValidator()
	renderer := service.NewTemplateRenderer(validator)
	templateService := service.NewNotificationTemplateService(db, validator)
	notificationService := service.NewNotificationService(db, templateService, renderer)
	apiKeyService := service.NewAPIKeyService(db)
	calendarService := service.NewCalendarService(db)
	exportService := service.NewExportService(db)
	importService := service.NewImportService(db)

	authHandler := NewAuthHandler(authService, totpService)
	subHandler := NewSubscriptionHandler(subService, erService)
	adminHandler := NewAdminHandler(adminService)
	erHandler := NewExchangeRateHandler(erService)
	currencyHandler := NewCurrencyHandler(currencyService, erService)
	categoryHandler := NewCategoryHandler(categoryService)
	paymentMethodHandler := NewPaymentMethodHandler(paymentMethodService)
	notificationHandler := NewNotificationHandler(notificationService)
	templateHandler := NewNotificationTemplateHandler(templateService)
	apiKeyHandler := NewAPIKeyHandler(apiKeyService)
	calendarHandler := NewCalendarHandler(calendarService)
	exportHandler := NewExportHandler(exportService)
	importHandler := NewImportHandler(importService)

	api := e.Group("/api")
	api.Use(securityHeadersMiddleware)
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

	api.GET("/version", func(c echo.Context) error {
		return c.JSON(http.StatusOK, version.Get())
	})

	api.GET("/version/latest", func(c echo.Context) error {
		client := &http.Client{Timeout: 10 * time.Second}
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
	auth.POST("/passkeys/login/start", authHandler.BeginPasskeyLogin)
	auth.POST("/passkeys/login/finish", authHandler.FinishPasskeyLogin)
	auth.GET("/oidc/config", authHandler.GetOIDCConfig)
	auth.POST("/oidc/login/start", authHandler.BeginOIDCLogin)
	auth.GET("/oidc/callback", authHandler.OIDCCallback)
	auth.GET("/oidc/session/:id", authHandler.GetOIDCSession)

	jwtConfig := echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}

	protected := api.Group("")
	protected.Use(JWTOrAPIKeyMiddleware(jwtConfig, apiKeyService))
	protected.Use(APIKeyScopeMiddleware)

	protected.GET("/subscriptions", subHandler.List)
	protected.POST("/subscriptions", subHandler.Create)
	protected.GET("/subscriptions/:id", subHandler.GetByID)
	protected.PUT("/subscriptions/:id", subHandler.Update)
	protected.DELETE("/subscriptions/:id", subHandler.Delete)
	protected.POST("/subscriptions/:id/icon", subHandler.UploadIcon)
	protected.GET("/dashboard/summary", subHandler.Dashboard)

	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/password", authHandler.ChangePassword)
	protected.POST("/auth/email/change/send-code", authHandler.SendEmailChangeVerificationCode)
	protected.POST("/auth/email/change/confirm", authHandler.ConfirmEmailChange)
	protected.GET("/auth/totp/setup", authHandler.SetupTOTP)
	protected.POST("/auth/totp/confirm", authHandler.ConfirmTOTP)
	protected.POST("/auth/totp/disable", authHandler.DisableTOTP)
	protected.GET("/auth/passkeys", authHandler.ListPasskeys)
	protected.POST("/auth/passkeys/register/start", authHandler.BeginPasskeyRegistration)
	protected.POST("/auth/passkeys/register/finish", authHandler.FinishPasskeyRegistration)
	protected.DELETE("/auth/passkeys/:id", authHandler.DeletePasskey)
	protected.GET("/auth/oidc/connections", authHandler.ListOIDCConnections)
	protected.POST("/auth/oidc/connect/start", authHandler.BeginOIDCConnect)
	protected.DELETE("/auth/oidc/connections/:id", authHandler.DeleteOIDCConnection)
	admin := api.Group("/admin")

	admin.Use(echojwt.WithConfig(jwtConfig))
	admin.Use(AdminMiddleware)

	admin.GET("/users", adminHandler.ListUsers)
	admin.POST("/users", adminHandler.CreateUser)
	admin.PUT("/users/:id/role", adminHandler.ChangeUserRole)
	admin.PUT("/users/:id/status", adminHandler.ChangeUserStatus)
	admin.DELETE("/users/:id", adminHandler.DeleteUser)
	admin.GET("/stats", adminHandler.GetStats)
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

	protected.GET("/api-keys", apiKeyHandler.List)
	protected.POST("/api-keys", apiKeyHandler.Create)
	protected.DELETE("/api-keys/:id", apiKeyHandler.Delete)

	protected.GET("/calendar/tokens", calendarHandler.ListTokens)
	protected.POST("/calendar/tokens", calendarHandler.CreateToken)
	protected.DELETE("/calendar/tokens/:id", calendarHandler.DeleteToken)

	protected.GET("/export", exportHandler.Export)
	protected.POST("/import/wallos", importHandler.ImportWallos, requestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))
	protected.POST("/import/subdux", importHandler.ImportSubdux, requestBodyLimitMiddleware(maxImportRequestBodyBytes, nil))

	api.GET("/calendar/feed", calendarHandler.GetCalendarFeed)

	api.GET("/site-info", func(c echo.Context) error {
		var setting model.SystemSetting
		siteName := "Subdux"
		if err := db.Where("key = ?", "site_name").First(&setting).Error; err == nil && setting.Value != "" {
			siteName = setting.Value
		}
		return c.JSON(http.StatusOK, echo.Map{"site_name": siteName})
	})

	seedDefaultSettings(db)

	return erService, notificationService
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "registration_enabled", Value: "true"},
		{Key: "registration_email_verification_enabled", Value: "false"},
		{Key: "email_domain_whitelist", Value: ""},
		{Key: "site_name", Value: "Subdux"},
		{Key: "site_url", Value: ""},
		{Key: "currencyapi_key", Value: ""},
		{Key: "exchange_rate_source", Value: "auto"},
		{Key: "allow_image_upload", Value: "true"},
		{Key: "max_icon_file_size", Value: "65536"},
		{Key: "smtp_enabled", Value: "false"},
		{Key: "smtp_host", Value: ""},
		{Key: "smtp_port", Value: "587"},
		{Key: "smtp_username", Value: ""},
		{Key: "smtp_password", Value: ""},
		{Key: "smtp_from_email", Value: ""},
		{Key: "smtp_from_name", Value: ""},
		{Key: "smtp_encryption", Value: "starttls"},
		{Key: "smtp_auth_method", Value: "auto"},
		{Key: "smtp_helo_name", Value: ""},
		{Key: "smtp_timeout_seconds", Value: "10"},
		{Key: "smtp_skip_tls_verify", Value: "false"},
		{Key: "oidc_enabled", Value: "false"},
		{Key: "oidc_provider_name", Value: "OIDC"},
		{Key: "oidc_issuer_url", Value: ""},
		{Key: "oidc_client_id", Value: ""},
		{Key: "oidc_client_secret", Value: ""},
		{Key: "oidc_redirect_url", Value: ""},
		{Key: "oidc_scopes", Value: "openid profile email"},
		{Key: "oidc_auto_create_user", Value: "false"},
		{Key: "oidc_authorization_endpoint", Value: ""},
		{Key: "oidc_token_endpoint", Value: ""},
		{Key: "oidc_userinfo_endpoint", Value: ""},
		{Key: "oidc_audience", Value: ""},
		{Key: "oidc_resource", Value: ""},
		{Key: "oidc_extra_auth_params", Value: ""},
	}

	for _, setting := range defaults {
		var existing model.SystemSetting
		if err := db.Where("key = ?", setting.Key).First(&existing).Error; err != nil {
			db.Create(&setting)
		}
	}
}
