package apimw

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

// AdminMiddleware rejects any request whose principal is not an admin.
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if From(c).Role != "admin" {
			return httpx.WriteError(c, http.StatusForbidden, "admin access required")
		}
		return next(c)
	}
}

// JWTOrAPIKeyMiddleware accepts either a Bearer JWT token or an X-API-Key header.
// JWT is tried first; if no Authorization header is present, it falls back to API key.
func JWTOrAPIKeyMiddleware(jwtConfig echojwt.Config, apiKeyService *apikeyservice.Service) echo.MiddlewareFunc {
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
				return httpx.WriteError(c, http.StatusUnauthorized, "authorization required")
			}

			principal, err := apiKeyService.WithContext(c.Request().Context()).ValidateKey(key)
			if err != nil {
				return httpx.WriteError(c, http.StatusUnauthorized, err.Error())
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

// MCPEnabledMiddleware rejects requests to the MCP endpoint when MCP is disabled
// in system settings.
func MCPEnabledMiddleware(settingsService *systemsettings.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			enabled, err := settingsService.IsMCPEnabled()
			if err != nil {
				return httpx.WriteError(c, http.StatusInternalServerError, "failed to read mcp settings")
			}
			if !enabled {
				return httpx.WriteError(c, http.StatusNotFound, "mcp is not enabled")
			}
			return next(c)
		}
	}
}
