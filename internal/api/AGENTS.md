# API Layer — HTTP Handlers

**Generated:** 2026-02-22 13:12 UTC  
**Commit:** fdfaf8c  
**Branch:** main

## OVERVIEW

12 Go files implementing Echo v4 handlers for auth, subscriptions, notifications, admin, TOTP/passkey/OIDC. Handlers extract JWT claims, validate input, delegate to services, return JSON.

## STRUCTURE

```
api/
├── router.go                  # SetupRoutes() — route groups, JWT middleware
├── auth.go                    # Register, Login (public)
├── subscription.go            # CRUD + Dashboard (protected)
├── notification*.go           # Channel/template/log management
├── admin*.go                  # User management, stats, settings
├── totp.go, passkey.go        # 2FA endpoints
├── oidc.go                    # OAuth callback handler
└── *_test.go                  # Validation tests
```

## WHERE TO LOOK

| Task | File | Pattern |
|------|------|---------|
| Add new endpoint | `router.go` | Add route → create handler method → wire service |
| Auth endpoints | `auth.go` | Public routes (no JWT) |
| Protected endpoints | `subscription.go`, `notification*.go` | Extract `userID` via `getUserID(c)` |
| Admin endpoints | `admin*.go` | Require `role == "admin"` check |
| Input validation | Any handler | Bind → validate → call service |
| Error responses | All handlers | `echo.Map{"error": "message"}` |

## CONVENTIONS

### Handler Pattern
```go
func (h *XHandler) Create(c echo.Context) error {
    userID := getUserID(c)          // Extract from JWT claims
    var input service.CreateXInput
    if err := c.Bind(&input); err != nil {
        return c.JSON(400, echo.Map{"error": "invalid input"})
    }
    // Validate here (not in service)
    result, err := h.service.Create(userID, input)
    if err != nil {
        return c.JSON(500, echo.Map{"error": err.Error()})
    }
    return c.JSON(200, result)
}
```

### Route Groups (router.go)
- `/api/auth` — Public (Register, Login)
- `/api/subscriptions` — Protected (JWT required)
- `/api/notifications` — Protected
- `/api/admin` — Protected + admin role check
- `/api/oidc` — Public (OAuth callback)

### JWT Middleware
- Applied to all `/api/*` except `/api/auth` and `/api/oidc`
- Claims stored in `c.Get("user")` as `*jwt.Token`
- Helper: `getUserID(c echo.Context) uint` extracts user ID

### Response Mapping
Private `xResponse` structs in handler files map models to JSON (hide internal fields)

## ANTI-PATTERNS

- **No validation in services** — handlers validate, services compute
- **No middleware beyond router.go** — all middleware configured in `SetupRoutes()`
- **No direct DB access** — handlers call services only
