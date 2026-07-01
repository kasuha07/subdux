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
	authRefreshPath        = "/api/auth/refresh"

	oidcSessionCookieName = "oidc_session"
	oidcSessionCookiePath = "/api/auth/oidc/session"
	oidcSessionCookieTTL  = 3 * time.Minute

	// The reauth ("step-up") OIDC session cookie is scoped to the admin reauth
	// finish endpoint so it is never sent on the ordinary login/connect session
	// path, and vice versa. It carries the result-session id minted by the OIDC
	// callback for a step-up flow.
	oidcReauthSessionCookieName = "oidc_reauth_session"
	oidcReauthSessionCookiePath = "/api/admin/reauth/oidc"
)

func setRefreshTokenCookie(c echo.Context, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	ttl := pkg.GetRefreshTokenTTL()
	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
	c.SetCookie(&http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    token,
		Path:     authRefreshPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearRefreshTokenCookie(c echo.Context) {
	clearCookie(c, refreshTokenCookieName, authRefreshPath)
}

func setOIDCSessionCookie(c echo.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
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

func setOIDCReauthSessionCookie(c echo.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
	c.SetCookie(&http.Cookie{
		Name:     oidcReauthSessionCookieName,
		Value:    sessionID,
		Path:     oidcReauthSessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(oidcSessionCookieTTL),
		MaxAge:   int(oidcSessionCookieTTL.Seconds()),
	})
}

func clearOIDCReauthSessionCookie(c echo.Context) {
	clearCookie(c, oidcReauthSessionCookieName, oidcReauthSessionCookiePath)
}

func getCookieValue(c echo.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func clearCookie(c echo.Context, name string, path string) {
	// #nosec G124 -- Expiry cookie mirrors the original cookie security attributes for reliable deletion.
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
