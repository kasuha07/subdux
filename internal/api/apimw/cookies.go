package apimw

import (
	"net/http"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/labstack/echo/v4"
)

const (
	RefreshTokenCookieName = "refresh_token"
	AuthRefreshPath        = "/api/auth/refresh"

	OIDCSessionCookieName = "oidc_session"
	oidcSessionCookiePath = "/api/auth/oidc/session"
	oidcSessionCookieTTL  = 3 * time.Minute

	// The reauth ("step-up") OIDC session cookie is scoped to the reauth OIDC
	// finish endpoint so it is never sent on the ordinary login/connect session
	// path, and vice versa. It carries the result-session id minted by the OIDC
	// callback for a step-up flow.
	OIDCReauthSessionCookieName = "oidc_reauth_session"
	oidcReauthSessionCookiePath = "/api/reauth/oidc"
)

func SetRefreshTokenCookie(c echo.Context, token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}

	ttl := pkg.GetRefreshTokenTTL()
	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
	c.SetCookie(&http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     AuthRefreshPath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(ttl),
		MaxAge:   int(ttl.Seconds()),
	})
}

func ClearRefreshTokenCookie(c echo.Context) {
	clearCookie(c, RefreshTokenCookieName, AuthRefreshPath)
}

func SetOIDCSessionCookie(c echo.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
	c.SetCookie(&http.Cookie{
		Name:     OIDCSessionCookieName,
		Value:    sessionID,
		Path:     oidcSessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(oidcSessionCookieTTL),
		MaxAge:   int(oidcSessionCookieTTL.Seconds()),
	})
}

func ClearOIDCSessionCookie(c echo.Context) {
	clearCookie(c, OIDCSessionCookieName, oidcSessionCookiePath)
}

func SetOIDCReauthSessionCookie(c echo.Context, sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	// #nosec G124 -- Secure is set for HTTPS/TLS and intentionally remains false on local HTTP development.
	c.SetCookie(&http.Cookie{
		Name:     OIDCReauthSessionCookieName,
		Value:    sessionID,
		Path:     oidcReauthSessionCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   shouldUseSecureCookies(c),
		Expires:  pkg.NowUTC().Add(oidcSessionCookieTTL),
		MaxAge:   int(oidcSessionCookieTTL.Seconds()),
	})
}

func ClearOIDCReauthSessionCookie(c echo.Context) {
	clearCookie(c, OIDCReauthSessionCookieName, oidcReauthSessionCookiePath)
}

func GetCookieValue(c echo.Context, name string) string {
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
