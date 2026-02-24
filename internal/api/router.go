package api

import (
	"encoding/json"
	"net/http"
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
	return claims.Role
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

			userID, err := apiKeyService.ValidateKey(key)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
			}

			claims := &pkg.JWTClaims{UserID: userID}
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

	api := e.Group("/api")

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
	auth.GET("/register/config", authHandler.GetRegistrationConfig)
	auth.POST("/register/send-code", authHandler.SendRegisterVerificationCode)
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/password/forgot", authHandler.ForgotPassword)
	auth.POST("/password/reset", authHandler.ResetPassword)
	auth.POST("/totp/verify-login", authHandler.VerifyTOTPLogin)
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
	admin.POST("/restore", adminHandler.RestoreDB)
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

	api.GET("/calendar/feed", calendarHandler.GetCalendarFeed)

	seedDefaultSettings(db)

	return erService, notificationService
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "registration_enabled", Value: "true"},
		{Key: "registration_email_verification_enabled", Value: "false"},
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
