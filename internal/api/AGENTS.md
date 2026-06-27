# API Layer - HTTP Handlers

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

Echo v4 API layer for REST, calendar feed, icon proxy, site/version info, and MCP over Streamable HTTP. Handlers bind and validate request input, derive principals from JWT/API keys, delegate to services, and map service/model results to stable JSON responses.

## STRUCTURE

```
api/
├── router.go                 # SetupRoutes, service wiring, route groups, middleware
├── security_middleware.go    # Human-session, API-key scope, request/body/origin guards
├── auth*.go                  # Registration, login, refresh sessions, TOTP, passkey, OIDC
├── subscription*.go          # CRUD, dashboard, detail, actions, reports response mapping
├── notification*.go          # Channels, policies, logs, templates
├── admin*.go                 # Users, stats, settings, backup/restore, background tasks
├── apikey.go, audit.go       # Human API-key and audit views
├── calendar.go               # Token management and public feed
├── import.go, export.go      # Wallos/Subdux imports and Subdux export
├── mcp*.go                   # MCP transport, schemas, args, results, tools, search helpers
└── *_test.go                 # Handler, middleware, response, MCP, security tests
```

## ROUTE GROUPS

- `/api/auth/*`: public auth, registration, password reset, refresh, passkey login, OIDC login/callback/session.
- `/api/version`, `/api/version/latest`, `/api/site-info`, `/api/icon-proxy/:provider`: public utility endpoints with their own limits where needed.
- Protected REST routes: subscriptions, dashboard, actions, reports, exchange rates, currencies, categories, payment methods, notifications, and imports.
- Human-only routes: account credential changes, passkey/OIDC connection management, API keys, user audit events, calendar token management, and export.
- `/api/admin/*`: admin JWT routes for users, stats, settings, SMTP test, backup/restore, exchange-rate refresh, audit events, and background tasks.
- `/api/calendar/feed`: public calendar feed authenticated by calendar token.
- `/mcp`: MCP POST endpoint; non-POST methods return method-not-allowed style responses.

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Add route | `router.go` | Wire service and handler in `SetupRoutes` |
| Add request validation | Handler file | Bind -> normalize/validate -> service call |
| Add response shape | Handler file or `mcp_results.go` | Prefer explicit response structs for public contracts |
| Change auth boundary | `security_middleware.go`, `router.go` | Add tests for JWT, API-key, human-only, and admin cases |
| Change MCP behavior | `mcp_handler.go`, `mcp_tools.go`, `mcp_schema.go` | Preserve SDK transport assumptions and header/origin checks |
| Change imports/exports | `import.go`, `export.go` | Keep request-size and human/API-key boundaries explicit |
| Change admin API | `admin.go` | Match `service/admin*.go` and frontend `admin` types |

## CONVENTIONS

### Handler Pattern

```go
func (h *XHandler) Create(c echo.Context) error {
    userID := getUserID(c)
    var input service.CreateXInput
    if err := c.Bind(&input); err != nil {
        return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid input"})
    }
    // Validate HTTP-level fields here.
    result, err := h.service.Create(userID, input)
    if err != nil {
        return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
    }
    return c.JSON(http.StatusOK, result)
}
```

- Handlers own HTTP input validation, status codes, and response mapping.
- Services own behavior and persistence decisions.
- Use `getUserID`, `getUserRole`, `getAuthType`, `hasAPIKeyScope`, and `getAPIKeyKind` instead of reparsing claims.
- Keep handler DTOs private unless there is an established shared type.

### Auth And API Keys

- `JWTOrAPIKeyMiddleware` accepts Bearer JWT first, then `X-API-Key`.
- API-key principals carry `AuthTypeAPIKey`, key ID, key kind, and scopes.
- API keys are machine principals. `getUserRole` intentionally does not grant human role privileges to API keys.
- Human-only routes must return the established human-session error rather than falling through to admin or generic auth behavior.

### MCP

- MCP is gated by the system setting plus `/mcp` middleware.
- Keep `X-API-Key`, `Origin`, `Content-Type`, `Accept`, and protocol-version checks in front of the SDK handler.
- Use explicit schemas/args/result helpers; do not let MCP expose broad internal structs by accident.
- Current MCP surface is intentionally subscription/settings-reference oriented; do not add admin, export, account, calendar-token, or notification-CRUD tools casually.

## TESTING

- Add handler tests in `internal/api/*_test.go` for response contracts and boundary behavior.
- For MCP changes, test the real `/mcp` request path and SDK-backed behavior rather than preserving old manual-dispatch semantics.
- For auth/security changes, include negative tests for API keys, missing scopes, cross-origin requests, bad content types, and human-only/admin-only routes.

## ANTI-PATTERNS

- Direct DB access from handlers.
- Business validation in handlers when it belongs in service logic.
- New middleware outside `router.go`/existing middleware files.
- Returning raw models that expose secrets or internal-only fields.
- Treating API keys as equivalent to human sessions.
