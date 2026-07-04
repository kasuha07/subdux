package apimw

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/pkg"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"github.com/labstack/echo/v4"
)

// principalContextKey caches the derived Principal on the request context so the
// JWT claims are interpreted exactly once per request rather than re-asserted at
// every accessor call.
const principalContextKey = "principal"

// Principal is the typed, request-scoped identity derived from the validated JWT
// or API-key claims. It replaces the scattered token/claims type assertions and
// centralizes the API-key role-downgrade and auth-type derivation rules in one
// place (buildPrincipal).
type Principal struct {
	UserID uint
	// Role is the effective human role. API-key principals are always "user"
	// (machine principals never inherit human role privileges), and a missing
	// role defaults to "user".
	Role     string
	AuthType string
	KeyKind  string
	Scopes   []string
}

// IsAPIKey reports whether the principal authenticated with an API key rather
// than a human session.
func (p Principal) IsAPIKey() bool {
	return p.AuthType == pkg.AuthTypeAPIKey
}

// HasScope reports whether the principal's API-key scopes include scope.
func (p Principal) HasScope(scope string) bool {
	for _, candidate := range p.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

// From returns the request's Principal, deriving and caching it on
// first use. It is safe to call on any authenticated route; on an unauthenticated
// context it returns the zero Principal rather than panicking.
func From(c echo.Context) Principal {
	if cached, ok := c.Get(principalContextKey).(Principal); ok {
		return cached
	}
	p := buildPrincipal(c)
	c.Set(principalContextKey, p)
	return p
}

func buildPrincipal(c echo.Context) Principal {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok || token == nil {
		return Principal{}
	}
	claims, ok := token.Claims.(*pkg.JWTClaims)
	if !ok || claims == nil {
		return Principal{}
	}

	authType := claims.AuthType
	if authType == "" {
		if len(claims.Scopes) > 0 {
			authType = pkg.AuthTypeAPIKey
		} else {
			authType = pkg.AuthTypeUser
		}
	}

	role := claims.Role
	if authType == pkg.AuthTypeAPIKey {
		// API keys are machine principals and should not be granted human role
		// privileges. Treat as least-privilege "user" for legacy role checks.
		role = "user"
	} else if strings.TrimSpace(role) == "" {
		role = "user"
	}

	return Principal{
		UserID:   claims.UserID,
		Role:     role,
		AuthType: authType,
		KeyKind:  apikeyservice.NormalizePersistedAPIKeyKind(claims.KeyKind),
		Scopes:   claims.Scopes,
	}
}
