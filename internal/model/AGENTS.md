# Model Layer

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

`internal/model` contains the persistent GORM structs shared across handlers, services, migrations, and tests. Keep this layer schema-focused: field shape, tags, associations, and model-level invariants only.

## STRUCTURE

```text
model/
├── auth_models.go           # Users, sessions, passkeys, OIDC links, TOTP, auth-related state
├── settings_models.go       # System settings, API keys, calendar tokens, exchange prefs
├── subscription_models.go   # Subscriptions, lifecycle events, categories, payment methods
├── notification_models.go   # Channels, templates, policies, outbox, logs
├── audit_models.go          # Audit events and related enums
└── idempotency_models.go    # MCP/API idempotency records
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Human auth/session schema | `auth_models.go` | Coordinate with `internal/service/auth` and auth API mapping |
| System/security settings | `settings_models.go` | Changes often affect admin/settings UI plus migrations |
| Subscription domain schema | `subscription_models.go` | Coordinate with `internal/service/subscription`, import/export, reports |
| Notification persistence | `notification_models.go` | Coordinate with `internal/service/notification` and settings UI |
| Audit storage | `audit_models.go` | Coordinate with `internal/service/audit` and admin/user audit views |
| Idempotency records | `idempotency_models.go` | Coordinate with `internal/service/idempotency` and `internal/api/mcp` |

## CONVENTIONS

- Keep models domain-split. Prefer editing the matching domain file instead of creating a generic catch-all file.
- Preserve JSON tags and serialized field names unless the API contract is intentionally changing.
- Keep business rules out of model methods. Services own behavior.
- Use explicit migrations under `internal/pkg/migrations/` for non-trivial schema/data changes; do not rely on `AutoMigrate` for semantic migrations.
- Be careful with zero values, nullable pointers, defaults, and uniqueness/index tags. They affect both persistence and transport behavior.

## VALIDATION

Useful commands after model changes:

```bash
go test -count=1 ./internal/pkg/migrations ./internal/service/... ./internal/api
go vet ./...
```

If the model change affects existing rows, verify the migration path and add or update migration tests.
