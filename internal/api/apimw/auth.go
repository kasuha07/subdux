package apimw

import (
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"github.com/kasuha07/subdux/internal/service/userstatus"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// CurrentUserMiddleware refreshes authorization on every human request so
// disabling, deleting or demoting an account takes effect for existing JWTs.
func CurrentUserMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			p := From(c)
			if p.IsAPIKey() {
				return next(c)
			}
			if p.UserID == 0 {
				return httpx.WriteError(c, http.StatusUnauthorized, "unauthorized")
			}
			user, err := userstatus.Load(db.WithContext(c.Request().Context()), p.UserID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return httpx.WriteError(c, http.StatusUnauthorized, "unauthorized")
				}
				return err
			}
			if !user.IsActive() {
				return httpx.WriteError(c, http.StatusUnauthorized, "unauthorized")
			}
			p.Role = user.Role
			c.Set(principalContextKey, p)
			return next(c)
		}
	}
}

// AdminMiddleware rejects any request whose principal is not an admin.
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if From(c).Role != "admin" {
			return httpx.WriteError(c, http.StatusForbidden, "admin_access_required")
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
				return httpx.WriteError(c, http.StatusUnauthorized, "authorization_required")
			}

			principal, err := apiKeyService.WithContext(c.Request().Context()).ValidateKey(key)
			if err != nil {
				return httpx.WriteErrorFrom(c, http.StatusUnauthorized, err)
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
				return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_read_mcp_settings")
			}
			if !enabled {
				return httpx.WriteError(c, http.StatusNotFound, "mcp_is_not_enabled")
			}
			return next(c)
		}
	}
}
