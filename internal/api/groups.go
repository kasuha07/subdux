package api

import (
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/labstack/echo/v4"
)

// Registrar is implemented by each domain handler to register its own routes
// onto the shared, security-scoped route groups. Keeping registration next to
// the handler—while the groups themselves are defined centrally—preserves the
// single-glance security audit (which group a route lands on) without a
// monolithic route table.
type Registrar interface {
	RegisterRoutes(g RouteGroups)
}

// RouteGroups are the security-scoped Echo groups a handler may attach to.
// Choosing the right group is the security decision; the group's middleware
// stack enforces it.
//
//   - Public: /api with body-limit only (no auth).
//   - Auth: /api/auth with the stricter auth-body limit (unauthenticated login/register flows).
//   - Protected: JWT-or-API-key, API-key scope enforced.
//   - HumanProtected: Protected plus human-session-only (no API-key principals).
//   - Reauth: /api/reauth, human-session-only step-up endpoints.
//   - Admin: /api/admin, JWT plus admin-role required.
type RouteGroups struct {
	Public         *echo.Group
	Auth           *echo.Group
	Protected      *echo.Group
	HumanProtected *echo.Group
	Reauth         *echo.Group
	Admin          *echo.Group
	Limiters       rateLimiters
}

// rateLimiters holds the shared per-route limiter middleware. They are built
// once per app so all routes share a limiter's window state.
type rateLimiters struct {
	authIP          echo.MiddlewareFunc
	loginAccount    echo.MiddlewareFunc
	registerAccount echo.MiddlewareFunc
	passwordAccount echo.MiddlewareFunc
	totpAccount     echo.MiddlewareFunc
	refreshToken    echo.MiddlewareFunc
	iconProxy       echo.MiddlewareFunc
	reauthIP        echo.MiddlewareFunc
	reauthUser      echo.MiddlewareFunc
	emailChangeUser echo.MiddlewareFunc
}

func newRateLimiters() rateLimiters {
	return rateLimiters{
		authIP:          apimw.AuthIPRateLimit(30, time.Minute),
		loginAccount:    apimw.AuthAccountRateLimit(10, time.Minute, apimw.LoginAccountKey),
		registerAccount: apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.RegisterAccountKey),
		passwordAccount: apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.EmailAccountKey),
		totpAccount:     apimw.AuthAccountRateLimit(8, 5*time.Minute, apimw.TOTPAccountKey),
		refreshToken:    apimw.AuthAccountRateLimit(20, time.Minute, apimw.RefreshTokenAccountKey),
		iconProxy:       apimw.AuthIPRateLimit(600, time.Minute),
		reauthIP:        apimw.AuthIPRateLimit(30, time.Minute),
		reauthUser:      apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.AuthenticatedUserAccountKey),
		emailChangeUser: apimw.AuthAccountRateLimit(6, 10*time.Minute, apimw.AuthenticatedUserAccountKey),
	}
}
