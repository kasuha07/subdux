# API Layer

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

Echo v4 API layer for REST endpoints, calendar feed, icon proxy, site/version info, and MCP over Streamable HTTP. Handlers bind and validate HTTP input, derive principals through `apimw`, call service packages, and map results to stable JSON contracts.

## LOCAL AGENTS INDEX

| Scope | File | Use when |
|------|------|----------|
| API layer | `internal/api/AGENTS.md` | Route wiring, handlers, response mapping, HTTP error handling |
| Middleware and principals | `internal/api/apimw/AGENTS.md` | JWT/API-key auth, rate limits, human/admin gates, request guards |
| MCP transport | `internal/api/mcp/AGENTS.md` | Tool schemas, SDK transport, MCP write idempotency |

## STRUCTURE

```text
api/
├── router.go                 # SetupRoutes, service wiring, route groups, middleware order
├── apimw/                    # JWT/API-key auth, principals, rate limits, body/origin/security middleware
├── contract/                 # Shared request/response contract helpers such as icon validation
├── httpx/                    # Common HTTP error/JSON helpers and transient SQLite helpers
├── error_handler.go          # Central Echo error handler for typed service errors
├── version.go                # /api/version and /api/version/latest
├── auth*.go                  # Registration, login, refresh, TOTP, passkey, OIDC, account session flows
├── reauth.go                 # Shared /api/reauth step-up endpoints
├── subscription*.go          # CRUD, detail, dashboard, actions, reports, icon upload
├── notification*.go          # Channels and templates
├── admin*.go                 # Users, settings, SMTP, backup/restore, background tasks
├── apikey.go, audit.go       # Human API-key and audit surfaces
├── calendar.go               # Calendar token management and public feed
├── import.go, export.go      # Subdux/Wallos import and Subdux export
├── mcp/                      # MCP transport, schema, tool implementations, search, idempotency
└── *_test.go                 # Handler, contract, boundary, security, and route tests
```

## ROUTE OWNERSHIP

- Public auth routes live under `/api/auth/*`.
- Shared step-up routes live under `/api/reauth/*`.
- Public utility routes include `/api/version`, `/api/version/latest`, `/api/site-info`, and `/api/icon-proxy/:provider`.
- Protected REST routes cover subscriptions, dashboard, actions, reports, exchange rates, currencies, categories, payment methods, notifications, and imports.
- Human-only routes cover account changes, passkeys, OIDC connections, API keys, audit events, calendar token management, and export.
- Admin JWT routes live under `/api/admin/*`.
- Public calendar feed is `/api/calendar/feed`.
- MCP lives at `/mcp` and is handled separately from REST.

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| Add or move a route | `router.go` | Keep service construction and middleware order in `SetupRoutes` |
| Change handler request validation | Matching handler file | Bind -> normalize/validate -> service call |
| Change principal or auth boundary | `apimw/` + `router.go` | Add JWT, API-key, human-only, and admin negative tests |
| Change API error/status mapping | `error_handler.go`, `internal/service/serviceerr/` | Preserve the frozen `{ "error": "..." }` envelope |
| Change shared JSON/HTTP helpers | `httpx/`, `contract/` | Keep API contract wording stable unless intentional |
| Change MCP behavior | `mcp/` | Preserve SDK transport assumptions and MCP-specific checks |
| Change import/export API | `import.go`, `export.go` | Keep request-size and human/API-key boundaries explicit |
| Change admin API | `admin.go` | Match frontend admin DTOs and corresponding service package behavior |

## CONVENTIONS

- Handlers own HTTP input validation, status codes, and response mapping.
- Services own business rules, persistence, and semantic validation.
- Use `apimw.From(c)` for request identity instead of reparsing JWT claims or headers.
- Return typed `serviceerr.Error` values from service calls and let `APIErrorHandler` decide HTTP status.
- Use shared helpers in `httpx` and `contract` for stable error envelopes and validation logic.
- Keep handler DTOs private unless there is already an explicit cross-file contract type.
- Do not expose raw models or secrets when a response mapper already exists.

## TESTING

- Add focused handler tests in `internal/api/*_test.go`.
- For auth/security changes, include negative tests for API keys, missing scopes, cross-origin requests, bad content types, and human-only/admin-only routes.
- For MCP changes, test the real `/mcp` path and SDK-backed behavior instead of reviving manual dispatch shortcuts.
