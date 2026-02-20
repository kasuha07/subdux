package api

import (
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
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

func SetupRoutes(e *echo.Echo, db *gorm.DB) *service.ExchangeRateService {
	authService := service.NewAuthService(db)
	totpService := service.NewTOTPService(db)
	subService := service.NewSubscriptionService(db)
	adminService := service.NewAdminService(db)
	erService := service.NewExchangeRateService(db)
	currencyService := service.NewCurrencyService(db)

	authHandler := NewAuthHandler(authService, totpService)
	subHandler := NewSubscriptionHandler(subService, erService)
	adminHandler := NewAdminHandler(adminService)
	erHandler := NewExchangeRateHandler(erService)
	currencyHandler := NewCurrencyHandler(currencyService, erService)

	api := e.Group("/api")

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/totp/verify-login", authHandler.VerifyTOTPLogin)
	auth.POST("/passkeys/login/start", authHandler.BeginPasskeyLogin)
	auth.POST("/passkeys/login/finish", authHandler.FinishPasskeyLogin)

	jwtConfig := echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}

	protected := api.Group("")
	protected.Use(echojwt.WithConfig(jwtConfig))

	protected.GET("/subscriptions", subHandler.List)
	protected.POST("/subscriptions", subHandler.Create)
	protected.GET("/subscriptions/:id", subHandler.GetByID)
	protected.PUT("/subscriptions/:id", subHandler.Update)
	protected.DELETE("/subscriptions/:id", subHandler.Delete)
	protected.POST("/subscriptions/:id/icon", subHandler.UploadIcon)
	protected.GET("/dashboard/summary", subHandler.Dashboard)

	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/password", authHandler.ChangePassword)
	protected.GET("/auth/totp/setup", authHandler.SetupTOTP)
	protected.POST("/auth/totp/confirm", authHandler.ConfirmTOTP)
	protected.POST("/auth/totp/disable", authHandler.DisableTOTP)
	protected.GET("/auth/passkeys", authHandler.ListPasskeys)
	protected.POST("/auth/passkeys/register/start", authHandler.BeginPasskeyRegistration)
	protected.POST("/auth/passkeys/register/finish", authHandler.FinishPasskeyRegistration)
	protected.DELETE("/auth/passkeys/:id", authHandler.DeletePasskey)

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

	seedDefaultSettings(db)

	return erService
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "registration_enabled", Value: "true"},
		{Key: "site_name", Value: "Subdux"},
		{Key: "site_url", Value: ""},
		{Key: "currencyapi_key", Value: ""},
		{Key: "exchange_rate_source", Value: "auto"},
		{Key: "max_icon_file_size", Value: "65536"},
	}

	for _, setting := range defaults {
		var existing model.SystemSetting
		if err := db.Where("key = ?", setting.Key).First(&existing).Error; err != nil {
			db.Create(&setting)
		}
	}
}
