# Infrastructure Layer

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

`internal/pkg` contains backend infrastructure shared across domains: database setup, schema migrations, JWT helpers, logging, settings crypto, timezone helpers, and runtime permission checks. Keep app-specific business rules out of this layer.

## STRUCTURE

```text
pkg/
├── database.go                    # SQLite setup and DB initialization
├── jwt.go                         # JWT claims and signing helpers
├── settings_crypto.go             # Encrypt/decrypt system setting secrets
├── clock.go                       # Time abstraction helpers
├── timezone.go                    # Timezone helpers
├── runtime_permissions_*.go       # Runtime permission adjustments by platform
├── logging/                       # Logger construction, middleware, redaction, context helpers
└── migrations/                    # Schema/data migration registry, runner, steps, tests
```

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| DB bootstrap or connection behavior | `database.go` | Keep SQLite hardening and runtime-path assumptions intact |
| Schema/data migration | `migrations/` | Update registry, runner, steps, and tests together |
| JWT claim shape or signing | `jwt.go` | Coordinate with `internal/api/apimw` and auth/API-key behavior |
| Secret setting encryption | `settings_crypto.go` | Preserve existing decrypt/load error behavior |
| Logging or request context | `logging/` | Keep redaction and middleware behavior stable |
| Timezone/runtime helpers | `timezone.go`, `runtime_permissions_*.go` | Respect OS/TZ-driven model; no per-user timezone layer |

## CONVENTIONS

- `internal/pkg` is infrastructure only. Do not move domain policy or transport behavior here just because multiple packages need it.
- Migrations must be deterministic, ordered, and safe against partially migrated data.
- Prefer additive registry/runner changes over ad hoc startup logic scattered elsewhere.
- Keep JWT/auth primitives shared and transport-agnostic where possible; API-specific interpretation belongs in `internal/api/apimw`.
- Preserve logging redaction and avoid introducing secret-bearing logs.

## VALIDATION

Useful commands after `pkg` changes:

```bash
go test -count=1 ./internal/pkg/...
go test -count=1 ./internal/api ./internal/service/...
go vet ./...
```

For migration work, include migration-runner tests and data-shape regression coverage.
