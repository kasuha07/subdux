# Go Backend

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

Layered Go backend for Subdux. `internal/` is split into HTTP transport (`api/`), business logic (`service/`), GORM models (`model/`), and shared infrastructure (`pkg/`). Read the nearest child `AGENTS.md` before changing a specific backend area.

## BACKEND AGENTS INDEX

| Scope | File | Use when |
|------|------|----------|
| Backend index | `internal/AGENTS.md` | Cross-cutting backend work or choosing the right child guide |
| HTTP API | `internal/api/AGENTS.md` | Route wiring, handlers, response mapping, HTTP error handling |
| API middleware | `internal/api/apimw/AGENTS.md` | JWT/API-key auth, request principals, rate limits, body/origin guards |
| MCP transport | `internal/api/mcp/AGENTS.md` | MCP schemas, tools, SDK transport, idempotent writes |
| Business logic | `internal/service/AGENTS.md` | Domain services, package boundaries, validation, concurrency |
| Persistence structs | `internal/model/AGENTS.md` | GORM model changes and schema shape |
| Shared infra | `internal/pkg/AGENTS.md` | DB setup, migrations, JWT, logging, crypto, timezone, runtime permissions |

## STRUCTURE

```text
internal/
├── api/       # Echo handlers, route wiring, middleware, HTTP helpers, MCP transport
├── service/   # Package-oriented business logic and shared service helpers
├── model/     # Domain-split GORM structs
├── pkg/       # DB, migrations, JWT, logging, crypto, timezone, permissions
└── version/   # ldflags-injected build metadata
```

## BACKEND-WIDE RULES

- Keep the flow explicit: `api` request parsing and auth boundary checks -> `service/*` business logic -> `model` + `pkg`.
- Prefer focused service subpackages over adding more business logic to the parent `internal/service` package.
- Use GORM APIs in request/business code. Raw SQL belongs only in narrowly justified migration/helper code.
- API keys are machine principals. Human-only account, audit, export, calendar-token, and credential flows must stay behind human-session-only middleware.
- MCP stays API-key based and narrower than REST. New MCP surfaces need schema, request limits, audit review, and trust-boundary review.
- Model shape changes usually imply matching service logic, API mappers, and often a migration under `internal/pkg/migrations/`.
- Use typed service errors from `internal/service/serviceerr/` and let `internal/api/error_handler.go` own HTTP status mapping.

## VALIDATION

Useful backend commands:

```bash
gofmt -w $(find . -path './web' -prune -o -name '*.go' -print)
go list ./internal/service/...
go test -count=1 ./internal/...
go vet ./...
```

For auth, API-key, MCP, import/export, backup/restore, outbound HTTP, or migration changes, add negative-path tests for rejected principals, unsafe URLs, malformed payloads, stale reauth tickets, or bad migration state.
