package api

import (
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func getUserID(c echo.Context) uint {
	token := c.Get("user").(*jwt.Token)
	claims := token.Claims.(*pkg.JWTClaims)
	return claims.UserID
}

func SetupRoutes(e *echo.Echo, db *gorm.DB) {
	authService := service.NewAuthService(db)
	subService := service.NewSubscriptionService(db)

	authHandler := NewAuthHandler(authService)
	subHandler := NewSubscriptionHandler(subService)

	api := e.Group("/api")

	// Public routes
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	// Protected routes
	protected := api.Group("")
	protected.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}))

	protected.GET("/subscriptions", subHandler.List)
	protected.POST("/subscriptions", subHandler.Create)
	protected.GET("/subscriptions/:id", subHandler.GetByID)
	protected.PUT("/subscriptions/:id", subHandler.Update)
	protected.DELETE("/subscriptions/:id", subHandler.Delete)
	protected.GET("/dashboard/summary", subHandler.Dashboard)
}
