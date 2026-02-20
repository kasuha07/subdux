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

func SetupRoutes(e *echo.Echo, db *gorm.DB) {
	authService := service.NewAuthService(db)
	subService := service.NewSubscriptionService(db)
	adminService := service.NewAdminService(db)

	authHandler := NewAuthHandler(authService)
	subHandler := NewSubscriptionHandler(subService)
	adminHandler := NewAdminHandler(adminService)

	api := e.Group("/api")

	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

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
	protected.GET("/dashboard/summary", subHandler.Dashboard)

	protected.GET("/auth/me", authHandler.Me)
	protected.PUT("/auth/password", authHandler.ChangePassword)

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

	seedDefaultSettings(db)
}

func seedDefaultSettings(db *gorm.DB) {
	defaults := []model.SystemSetting{
		{Key: "registration_enabled", Value: "true"},
		{Key: "site_name", Value: "Subdux"},
		{Key: "site_url", Value: ""},
	}

	for _, setting := range defaults {
		var existing model.SystemSetting
		if err := db.Where("key = ?", setting.Key).First(&existing).Error; err != nil {
			db.Create(&setting)
		}
	}
}
