package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
)

const (
	refreshTokenCookieName = "refresh_token"
	refreshTokenCookiePath = "/api/auth/refresh"

	oidcSessionCookieName = "oidc_session"
	oidcSessionCookiePath = "/api/auth/oidc/session"
	oidcSessionCookieTTL  = 3 * time.Minute
)

func setRefreshTokenCookie(c echo.Context, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	ttl := pkg.GetRefreshTokenTTL()
	c.SetCookie(&http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     refreshTokenCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearRefreshTokenCookie(c echo.Context) {
	clearCookie(c, refreshTokenCookieName, refreshTokenCookiePath)
}

func setOIDCSessionCookie(c echo.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	c.SetCookie(&http.Cookie{
		Name:     oidcSessionCookieName,
		Value:    sessionID,
		Path:     oidcSessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(oidcSessionCookieTTL),
		MaxAge:   int(oidcSessionCookieTTL.Seconds()),
	})
}

func clearOIDCSessionCookie(c echo.Context) {
	clearCookie(c, oidcSessionCookieName, oidcSessionCookiePath)
}

func getCookieValue(c echo.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func clearCookie(c echo.Context, name string, path string) {
	c.SetCookie(&http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
}

func shouldUseSecureCookies(c echo.Context) bool {
	if c.Request().TLS != nil {
		return true
	}

	forwardedProto := c.Request().Header.Get(echo.HeaderXForwardedProto)
	if forwardedProto != "" {
		first := strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
		if strings.EqualFold(first, "https") {
			return true
		}
	}

	return strings.EqualFold(strings.TrimSpace(c.Scheme()), "https")
}
