# Go Backend

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

Layered Go backend for Subdux. HTTP entry points live in `api/`, business behavior in `service/`, persistence structs in `model/`, and shared runtime infrastructure in `pkg/`.

The backend now covers subscriptions, lifecycle/action workflows, analytics reports, notifications, imports/exports, audit, API keys, MCP, calendar feeds, admin settings, OIDC, passkeys, TOTP, safe outbound HTTP, and background-task monitoring.

## STRUCTURE

```
internal/
├── api/       # Echo route setup, handlers, middleware, MCP transport boundary
├── service/   # Business logic, validation helpers, outbox/dispatch, imports, reports
├── model/     # Domain-split GORM structs: auth, settings, subscription, notification, audit
└── pkg/       # SQLite setup, migrations, JWT, logging, crypto, timezone, runtime permissions
```

## WHERE TO LOOK

| Task | Start here | Then |
|------|------------|------|
| Add HTTP endpoint | `api/router.go` | Handler file -> service method -> tests |
| Add MCP tool | `api/mcp_tools.go`, `api/mcp_schema.go` | `api/mcp_args.go`, `api/mcp_results.go`, service calls |
| Change auth/session rules | `api/security_middleware.go`, `pkg/jwt.go` | `service/auth*.go`, API tests |
| Change API key behavior | `service/apikey.go` | `api/apikey.go`, MCP/API boundary tests |
| Add or change model field | `model/*_models.go` | `pkg/migration_*.go` if existing data needs migration |
| Change notification delivery | `service/notification*.go` | Channel-specific tests and settings UI |
| Change imports/exports | `service/import_*.go`, `service/export.go` | `api/import.go`, `api/export.go`, payload-limit tests |
| Change admin settings | `service/admin_settings.go`, `service/system_settings.go` | `api/admin.go`, frontend admin settings |
| Change DB/runtime config | `pkg/database.go`, `pkg/schema_migrations.go` | `DATA_PATH`, runtime permission helpers |

## CONVENTIONS

### Layering
- `api/` owns HTTP shape: route registration, request binding, auth middleware, status codes, response DTOs.
- `service/` owns business rules, persistence behavior, domain validation, background work, and reusable helpers.
- `model/` contains GORM structs and JSON tags; keep models domain-split.
- `pkg/` contains infrastructure used across domains. Keep app-specific business rules out of `pkg/`.

### Auth Boundaries
- Ordinary protected REST routes use `JWTOrAPIKeyMiddleware` plus scope checks.
- Human-only routes use `HumanSessionOnlyMiddleware`; keep account credentials, API-key management, audit access, calendar token management, and export behind it.
- Admin routes use JWT plus `AdminMiddleware`. Do not grant admin privileges to API-key principals.
- MCP is a separate `/mcp` entrypoint and should stay API-key based, bounded, audited where appropriate, and narrower than REST.

### Persistence
- Use GORM APIs for request/business logic.
- Use migrations in `pkg/` for non-trivial existing-data changes; do not rely on `AutoMigrate` for destructive or semantic migration work.
- Default data path is `data/`; override with `DATA_PATH`. SQLite DB and uploaded assets live under the data path.
- Respect the existing SQLite hardening helpers and connection settings before adding new concurrency behavior.

### Errors And Responses
- Use the shared handler response style already present in `api/`.
- Avoid leaking internal structs directly when a handler has an existing response mapper.
- Preserve current JSON field names and response shapes unless intentionally changing API contract.

## TESTING

- Backend tests live under `internal/api`, `internal/service`, and `internal/pkg`.
- Prefer focused table-driven tests near the changed package, then run broader checks.

Useful backend validation:

```bash
gofmt -w $(find . -path './web' -prune -o -name '*.go' -print)
go test ./...
go vet ./...
```

For auth, API-key, MCP, import/export, backup/restore, or outbound HTTP changes, add negative controls for rejected principals, missing scopes, unsafe URLs, bad content types, oversized payloads, and privilege-boundary failures.

## ANTI-PATTERNS

- Raw SQL in request/business logic.
- Service-to-service calls that make ownership and testing unclear.
- Middleware setup outside `api/router.go`.
- API-key principals receiving human-only or admin capabilities.
- Expanding MCP into admin, export, credential, notification-CRUD, or account-management surfaces without an explicit trust-boundary review.
