# Subdux — Subscription Tracker

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## PROJECT OVERVIEW

Go 1.26.4 + React 19 monorepo. Subdux builds as a single binary with the Vite frontend embedded into the Go server. It tracks recurring subscriptions, renewal actions, reports, calendar feeds, multi-currency costs, notifications, imports/exports, API keys, MCP access, admin settings, backup/restore, and human-account authentication through password, TOTP, passkey, and OIDC flows.

**Stack:** Echo v4 + GORM on SQLite, `modelcontextprotocol/go-sdk`, React 19 + React Router 7, Vite 8, Tailwind v4, Shadcn-style local UI primitives, i18next, Bun.

This root file is intentionally an index. Read the nearest child `AGENTS.md` before editing a specific area.

## AGENTS Hierarchy

```text
subdux/
├── AGENTS.md
├── internal/
│   ├── AGENTS.md
│   ├── api/
│   │   ├── AGENTS.md
│   │   ├── apimw/AGENTS.md
│   │   └── mcp/AGENTS.md
│   ├── service/AGENTS.md
│   ├── model/AGENTS.md
│   └── pkg/AGENTS.md
└── web/
    ├── AGENTS.md
    └── src/features/
        ├── AGENTS.md
        ├── settings/AGENTS.md
        └── admin/AGENTS.md
```

| Scope | File | Use when |
|------|------|----------|
| Repository-wide rules | `AGENTS.md` | Build/test/commit policy, top-level boundaries, how the AGENTS tree is organized |
| Backend index | `internal/AGENTS.md` | Any Go backend work; choose the right backend child doc first |
| API layer | `internal/api/AGENTS.md` | Routes, handlers, HTTP contracts, API error mapping |
| API middleware | `internal/api/apimw/AGENTS.md` | JWT/API-key auth, principal derivation, rate limits, body/origin/security headers |
| MCP transport | `internal/api/mcp/AGENTS.md` | MCP tool schema, transport, idempotent write tools, audit behavior |
| Service layer | `internal/service/AGENTS.md` | Business logic, package boundaries, service-level validation and tests |
| Models | `internal/model/AGENTS.md` | GORM structs and schema-shape changes |
| Infrastructure | `internal/pkg/AGENTS.md` | Migrations, DB setup, JWT, logging, crypto, timezone, runtime permissions |
| Frontend index | `web/AGENTS.md` | Any React/web work; choose feature vs shared-layer guidance |
| Feature index | `web/src/features/AGENTS.md` | Route/page ownership across auth, dashboard, subscriptions, actions, reports, calendar, settings, admin |
| Settings feature | `web/src/features/settings/AGENTS.md` | User settings, API keys, audit, import/export, notification settings |
| Admin feature | `web/src/features/admin/AGENTS.md` | Admin users/settings/SMTP/OIDC/backup/reauth flows |

## Repo Map

| Area | Primary entry points |
|------|----------------------|
| Backend | `cmd/server/main.go`, `internal/`, `frontend.go` |
| Frontend | `web/src/App.tsx`, `web/src/features/`, `web/src/lib/api.ts` |
| Shared build/release | `Makefile`, `Dockerfile`, `internal/version/` |
| Project helpers | `skill/` |

## Global Rules

- Read the nearest child `AGENTS.md` before editing. Child guidance narrows local structure; root policy still applies.
- Keep the layered flow: `internal/api` handlers and middleware -> `internal/service/*` business logic -> `internal/model` + `internal/pkg`.
- API keys are machine principals. Human-only account, audit, export, calendar-token, credential, and API-key management flows must stay behind human-session boundaries.
- Keep MCP narrower than REST. New MCP tools need explicit schema, bounded inputs, API-key auth, and trust-boundary review.
- Frontend feature code should stay under `web/src/features/{domain}` and authenticated network calls must go through `web/src/lib/api.ts`.
- Keep user-facing text translated in `en`, `zh-CN`, and `ja`.

## Build And Validation

**Build sequence:** frontend first (`web/dist`) -> Go embed -> single `subdux` binary.

```bash
make build          # bun install + web build, then go build with version ldflags
make dev            # tmux session: Go server plus Vite dev server
make frontend-deps  # bun install
make frontend       # bun install + bun run build
make frontend-lint  # bun install + bun run lint
make fmt            # gofmt -w on all Go files outside web/
make fmt-check      # fail if any Go file needs gofmt
make test           # frontend build, then go test ./...
make vet            # frontend build, then go vet ./...
make check          # gofmt check, frontend lint/build, go vet, go test
make docker         # multi-stage Docker image
```

Useful repo-level validation:

```bash
git diff --check
go list ./internal/service/...
make check
cd web && bun run test
```

## COMMIT MESSAGE REQUIREMENTS

- For any non-trivial change, commit messages MUST include a detailed body, not only a short title.
- Use Conventional Commits for the title: `type(scope): summary`. Scope is optional but preferred when it clarifies the affected area.
- Prefer common types such as `feat`, `fix`, `refactor`, `docs`, `test`, `chore`, `build`, `ci`, `perf`, `style`, and `revert`.
- Mark breaking changes with `!` in the title (for example `feat(api)!: ...`) and include a `BREAKING CHANGE:` footer explaining the impact and migration path.
- Use scoped bullets such as `Backend`, `API`, `Frontend`, `MCP`, `Security`, `i18n`, `Tests`, or `Docs`.
- Describe concrete behavior and implementation details: parsing rules, route paths, auth boundary changes, data mapping, dedup rules, validation, UX feedback, translation coverage, and test coverage when relevant.

Example:
```text
feat(import): add Wallos import

- Backend: import service parses Wallos JSON export, maps payment cycles
      including "Every N Units", extracts currencies from symbols/codes, and
      deduplicates by name+amount+currency+billing_type+next_billing_date
- API: add POST /api/import/wallos with explicit request-size limits
- Frontend: add account settings file picker with toast feedback
- i18n: cover en, zh-CN, and ja strings
- Tests: add service and handler coverage for duplicate import rows
```

## RELEASE REQUIREMENTS

- Small, non-breaking `patch` releases may be tag-only; `minor` and `major` releases must publish a GitHub Release with release notes.
- Any release containing breaking changes must publish a GitHub Release.
- Release notes should use the sections `Highlights`, `Features`, and `Fixes`, in that order, with concise bullet points.
- Breaking releases must include a `Breaking Changes` section before `Highlights`, covering the reason for the break, affected users, and migration steps.

## NOTES

- **Monorepo but not workspace:** no `go.work` and no package.json workspaces; Go and web tooling are coordinated by repo convention and `Makefile`.
- **Data directory:** default `data/` at repo/runtime root; override with `DATA_PATH`. SQLite DB and uploaded assets live below that data path.
- **Embedded assets:** `frontend.go` embeds `web/dist/`; build the frontend before compiling the production Go binary.
- **Service layout:** `internal/service/` is package-oriented; the parent package should only keep thin compatibility facades when a refactor needs them.
- **Version injection:** `Makefile` injects `VERSION`, `COMMIT`, and `BUILD_DATE` into `internal/version/` with `-ldflags`.
- **Single binary:** API is under `/api/*`, MCP is `/mcp`, calendar feed is `/api/calendar/feed`, uploaded assets are under `/uploads/*`, and the SPA is served from `/`.
- **Runtime bind:** server defaults to `:8080` unless `PORT` is set.
- **Timezone support:** system timezone only via `TZ` or OS default. No per-user timezone support.
