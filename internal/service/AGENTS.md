# Service Layer - Business Logic

**Generated:** 2026-06-27 11:32 UTC
**Commit:** 0967e52
**Branch:** main

## OVERVIEW

Business logic layer for auth, subscriptions, lifecycle actions, reports, imports/exports, notifications, audit, API keys, calendar feeds, admin operations, system settings, safe outbound HTTP, OIDC, passkeys, TOTP, and background tasks.

Services generally receive `*gorm.DB`, expose small input structs, and return models, DTO-like structs, or errors. Keep this layer independent enough that handlers compose services instead of services calling unrelated services.

## STRUCTURE

```
service/
├── auth*.go                         # Registration, login, refresh sessions, email, bootstrap, cleanup
├── passkey.go, totp.go, oidc.go     # Additional auth methods and OIDC callback behavior
├── apikey.go, audit.go              # API-key principals and audit logging
├── subscription*.go                 # CRUD, billing, lifecycle, rollover, detail, reports, actions
├── import_*.go, export.go           # Subdux/Wallos import and Subdux export
├── notification*.go                 # Channels, validation, templates, rendering, outbox, dispatch
├── admin*.go                        # Admin users, stats, settings, backup, SMTP
├── category.go, currency.go         # User-scoped settings/reference data
├── payment_method.go                # Payment methods and icons
├── calendar.go                      # Calendar feed tokens and event generation
├── outbound_http.go                 # Safe outbound HTTP policy/client
├── system_settings.go               # Global runtime settings
└── *_test.go                        # Focused service behavior and security tests
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| Auth/password/session behavior | `auth*.go` | Keep refresh-token and human-session expectations intact |
| TOTP/passkey/OIDC | `totp.go`, `passkey.go`, `oidc.go` | Cover login, linking, metadata, and security cases |
| API keys | `apikey.go` | Preserve kind/scope separation for REST vs MCP |
| Subscription CRUD | `subscription_crud.go`, `subscription.go` | Keep user scoping and URL safety helpers |
| Billing/lifecycle | `subscription_billing.go`, `subscription_lifecycle.go`, `subscription_rollover.go` | Date math must handle month end and leap years |
| Detail/actions/reports | `subscription_detail.go`, `subscription_actions*.go`, `subscription_report.go` | Avoid N+1 regressions and stale-action semantics |
| Notifications | `notification*.go` | Validate config, render templates, enqueue, dispatch, log policy outcomes |
| Imports/exports | `import_subdux.go`, `import_wallos.go`, `export.go` | Keep dedup, mapping, and size/security behavior tested |
| Admin settings | `admin_settings.go`, `system_settings.go`, `security_settings.go` | Handle configured-secret flags carefully |
| Outbound network | `outbound_http.go` | Use safe client rules for OIDC, webhooks, icon proxy, release checks |

## CONVENTIONS

### Constructors

```go
func NewXService(db *gorm.DB) *XService {
    return &XService{db: db}
}
```

- Keep constructors simple and follow existing local exceptions where a service already needs helper dependencies.
- Avoid service-to-service calls. Prefer shared helpers or handler composition when two domains need coordination.

### Input And Update Structs

- `CreateXInput` generally uses value fields for required input.
- `UpdateXInput` generally uses pointer fields so nil means "unchanged".
- Keep HTTP-specific parsing in handlers; keep semantic normalization and persistence rules in services.

### Data Ownership

- All user-owned queries must be scoped by `userID`.
- Treat URL fields as sink/source data: subscription CRUD, imports, and MCP writes can feed UI links.
- Secret settings often use paired configured flags; avoid clearing existing secrets unless the input explicitly requests replacement/removal.
- Calendar tokens, API keys, audit views, and export data are human-session surfaces at the API layer; service tests should still assume hostile IDs/input.

### Dates And Money

- Preserve existing recurring-only billing assumptions.
- Date math should stay deterministic around month end and leap years.
- Dashboard/report conversions should use existing exchange-rate and monthly-cost helpers rather than reimplementing formulas ad hoc.

### Notifications

- Keep validation and delivery separated.
- Outbox/dispatch logic should remain lease-aware and idempotent enough for retries.
- Template rendering must keep unsafe template behavior covered by tests.
- New channels need validation, rendering/delivery, policy/log behavior, settings UI, and tests.

## TESTING

- Service tests are table-driven and use temporary/in-memory SQLite helpers.
- Prefer plain `if` + `t.Fatalf` style unless the nearby tests use a helper.
- Add targeted regression tests for semantic changes before running full package tests.

Useful commands:

```bash
go test ./internal/service
go test ./...
go test -race ./...
```

Run race tests for notification/background/lease/concurrency changes when feasible. For outbound/security changes, include negative controls for private IPs, unsafe redirects, invalid templates, rejected domains, and privilege boundaries.

## ANTI-PATTERNS

- Raw SQL in service logic.
- Cross-service orchestration hidden inside services.
- Trusting handler validation for security-critical service behavior.
- Losing user scoping on list/detail/update/delete queries.
- Clearing configured secrets because a form sends an empty placeholder.
- Adding notification or MCP behavior without corresponding validation and tests.
