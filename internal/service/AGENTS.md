# Service Layer - Business Logic

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

Business logic layer for auth, reauthentication, subscriptions, lifecycle actions, reports, imports/exports, notifications, audit, API keys, calendar feeds, admin operations, system settings, outbound HTTP policy, icon proxying, SMTP, multi-destination backup/restore, and background tasks.

The service layer is now package-oriented: concrete domain logic lives in subpackages under `internal/service/*`. Do not add new Go files directly in `internal/service/` unless you are deliberately reintroducing a narrowly scoped compatibility facade with a clear reason.

Services generally receive `*gorm.DB`, expose small input structs, and return models, DTO-like structs, or domain errors. Keep HTTP parsing/status-code decisions in `internal/api`; keep persistence rules, semantic validation, and domain invariants here.

## STRUCTURE

```
service/
├── admin/           # Admin users, settings aggregation, SSRF test surface, SMTP/backup settings
├── apikey/          # API key lifecycle, kind/scope policy, active-user checks
├── audit/           # Audit event creation/listing and audit-enabled setting checks
├── auth/            # Registration, login, sessions, password, email policy, OIDC, passkeys, TOTP
├── authreauth/      # Bridge between auth factor verification and reauth service contracts
├── backup/          # Multi-destination backup (local/S3/WebDAV) fan-out, per-destination retention, durable run state, plus restore/export archive behavior
├── calendar/        # Calendar feed tokens and event generation
├── catalog/         # Categories, currencies, payment methods
├── exchangerate/    # Exchange-rate preferences, fetching, conversion helpers
├── exporter/        # Subdux export payload assembly
├── iconproxy/       # Icon proxy, domain policy, and icon-proxy settings
├── idempotency/     # MCP/API idempotency key claim/finalization
├── importer/        # Subdux and Wallos import parsing, mapping, and dedup
├── notification/    # Channels, validation, templates, rendering, outbox, scheduler, dispatch
├── outbound/        # Outbound HTTP clients, proxy dialers, SSRF policy/settings/validators
├── reauth/          # Operation-scoped step-up tickets, factor policy, OIDC grade requirements
├── servicetest/     # Shared service test DB/user helpers
├── serviceutil/     # Shared tokens, background leases/monitoring, icon validation, user defaults
├── settings/        # Global runtime/system/security settings helpers
├── smtp/            # SMTP runtime config and sender
├── subscription/    # CRUD, billing, lifecycle, events, actions, dashboard, reports, reminders
├── serviceerr/      # Typed service error vocabulary mapped to HTTP in api/error_handler.go
└── userstatus/      # Active-user checks shared by auth-sensitive services
```

## WHERE TO LOOK

| Task | Start here | Notes |
|------|------------|-------|
| Auth/password/session behavior | `auth/` | Keep refresh-token, session limits, and human-session expectations intact |
| OIDC/passkey/TOTP behavior | `auth/oidc.go`, `auth/passkey.go`, `auth/totp.go` | Cover login, linking, metadata, freshness, and downgrade cases |
| Step-up reauth policy | `reauth/`, `authreauth/` | Tickets are user-scoped, operation-scoped, single-use, TTL-limited, and stateful |
| API keys | `apikey/` | Preserve API-key kind/scope separation for REST vs MCP and active-user checks |
| Admin user/settings behavior | `admin/` | Keep sensitive admin mutations operation-scoped and reauth-gated at the API boundary |
| Subscription CRUD/lifecycle/actions | `subscription/` | Keep user scoping, URL safety, date math, stale-action semantics, and report conversions tested |
| Categories/currencies/payment methods | `catalog/` | Preserve per-user ownership and delete guards |
| Exchange rates | `exchangerate/` | Use outbound clients/settings and existing monthly-cost helpers |
| Notifications | `notification/` | Validate config, render templates safely, enqueue/lease/dispatch, and log policy outcomes |
| Imports/exports | `importer/`, `exporter/` | Keep payload limits, dedup, URL handling, and human/API-key boundaries explicit in API tests |
| Calendar feeds | `calendar/` | Treat feed tokens as bearer credentials; keep hostile-ID and lifecycle tests |
| Outbound network | `outbound/` | Use central client/dialer/policy helpers for OIDC, webhooks, icon proxy, exchange rates, and release checks |
| Icon proxy/upload helpers | `iconproxy/`, `serviceutil/` | Keep domain policy and upload validation centralized |
| System/security settings | `settings/`, `admin/` | Handle configured-secret flags carefully; do not clear secrets on placeholder input |
| Backup/restore | `backup/` | Pluggable `BackupTarget` transports (local/S3/WebDAV) fan out one archive per run; keep per-destination retention, durable/resumable run state, archive extraction, path safety, and restore semantics tested |
| Typed service errors | `serviceerr/` | Add or reuse `Kind`-based errors instead of transport-specific status logic |
| Shared service tests | `servicetest/` | Reuse `NewDB` and user helpers instead of copying test setup |

## CONVENTIONS

### Package Boundaries

- Prefer domain subpackages over a broad parent `service` package.
- Do not import the parent `internal/service` package from a service subpackage.
- Keep compatibility shims thin when they exist: aliases, constructors, or direct forwarding only.
- Shared helpers belong in focused packages such as `serviceutil`, `settings`, `outbound`, `smtp`, `servicetest`, or `userstatus`.
- Service-to-service imports are acceptable only when they reflect an established boundary, such as notification scheduling reading subscription reminder candidates or admin settings composing settings/backup/smtp/outbound services. Avoid hidden orchestration that would be clearer in `api/`.
- When splitting more services, move tests with the logic and replace parent-helper coupling with small local interfaces or lower-level helper packages.

### Constructors

```go
func NewService(db *gorm.DB) *Service {
    return &Service{db: db}
}
```

- Keep constructors simple and follow existing local exceptions where a service already needs helper dependencies.
- Use explicit constructor names only when the package exposes multiple service types, such as `catalog.NewCategoryService` or `notification.NewNotificationTemplateService`.

### Input And Update Structs

- `CreateXInput` generally uses value fields for required input.
- `UpdateXInput` generally uses pointer fields so nil means "unchanged".
- Keep HTTP-specific binding and content-type decisions in handlers.
- Keep semantic normalization, deduplication, policy checks, and persistence rules in services.

### Errors

- Use `serviceerr.New` / `serviceerr.Wrap` for client-facing service errors that need stable transport mapping.
- Preserve existing external error text unless the contract is intentionally changing.
- Keep HTTP status selection out of services; `internal/api/error_handler.go` is the transport boundary.

### Security And Data Ownership

- Scope all user-owned queries by `userID`; list/detail/update/delete paths should all prove ownership.
- Treat URL fields as sink/source data: subscription CRUD, imports, MCP writes, icon proxying, OIDC, webhooks, SMTP, and exchange-rate fetches must use the central safety helpers appropriate to their trust boundary.
- API keys are machine principals. Service methods should not assume API keys can access human-only surfaces; enforce that at API boundaries and keep service tests hostile to bad IDs/input.
- Calendar tokens, API keys, audit views, export data, credential management, and reauth-protected operations are human-session surfaces at the API layer.
- Secret settings often use paired configured flags; avoid clearing existing secrets unless the input explicitly requests replacement or removal.

### Reauth

- Add new sensitive operations in `reauth/operations.go` and keep `IsValidReauthOperation` in sync.
- Keep factor policy in `reauth/policy.go`; do not duplicate factor-strength rules in feature services.
- OIDC reauth must stay fresh-login enforced: request-side `prompt=login`/`max_age=0` and callback-side `auth_time` validation before minting a reauth result.
- Consume `X-Reauth-Ticket` at the API boundary, then keep feature services focused on the mutation itself.
- Treat recovery workflows separately from weakening normal sensitive-operation policy.

### Dates And Money

- Preserve existing recurring-only billing assumptions.
- Date math should stay deterministic around month end and leap years.
- Dashboard/report conversions should use existing exchange-rate and monthly-cost helpers rather than reimplementing formulas ad hoc.

### Notifications

- Keep validation, rendering, enqueueing, and delivery separated.
- Outbox/dispatch logic should remain lease-aware and idempotent enough for retries.
- Template rendering must keep unsafe template behavior covered by tests.
- New channels need validation, rendering/delivery, policy/log behavior, settings UI/API wiring, and tests.

## TESTING

- Service tests are table-driven and usually use temporary/in-memory SQLite helpers.
- Prefer plain `if` + `t.Fatalf` style unless nearby tests use a helper.
- Add targeted regression tests for semantic changes before running full package tests.
- For refactors, first prove package boundaries with `go list ./internal/service/...`.

Useful commands:

```bash
go list ./internal/service/...
go test -count=1 ./internal/service/...
go test -count=1 ./internal/api
go test ./...
go test -race ./...
```

Run race tests for notification/background/lease/concurrency changes when feasible. For outbound/security/auth/MCP changes, include negative controls for private IPs, unsafe redirects, invalid templates, rejected domains, missing scopes, stale or cross-operation reauth tickets, and privilege-boundary failures.

## ANTI-PATTERNS

- New business logic directly in the parent `internal/service` directory.
- Raw SQL in service logic.
- Cross-service orchestration hidden inside services when handler composition is clearer.
- Trusting handler validation for security-critical service behavior.
- Losing user scoping on list/detail/update/delete queries.
- Reimplementing outbound HTTP clients, proxy dialers, URL validation, tokens, or background leases ad hoc.
- Returning transport-specific status logic from service code instead of typed `serviceerr` values.
- Clearing configured secrets because a form sends an empty placeholder.
- Duplicating reauth factor policy outside `reauth/`.
- Adding notification, MCP, import/export, backup/restore, or outbound behavior without corresponding validation and tests.
