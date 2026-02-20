# AGENTS.md

Operational guide for coding agents in this repository.
Use this as the primary cross-repo playbook.

## Project Snapshot

- Name: `subdux` — subscription tracker SPA
- Backend: Go + Echo + GORM + pure-Go SQLite (`github.com/glebarez/sqlite`)
- Frontend: React 19 + TypeScript + Vite 7 + Bun + shadcn/ui (new-york/zinc) + Tailwind v4
- Auth: JWT (Bearer), TOTP (`pquerna/otp`), Passkey (`go-webauthn/webauthn`)
- Runtime: single Go binary embedding `web/dist`, serves SPA + `/api` + `/uploads`
- Port: `8080` (override via `PORT` env var)

## Repository Map

```
cmd/server/main.go        # Boot: Echo setup, middleware, SPA fallback
frontend.go               # go:embed all:web/dist
internal/api/
  router.go               # SetupRoutes(), getUserID(), getUserRole(), AdminMiddleware
  auth.go                 # AuthHandler — Register/Login/TOTP/Passkey
  subscription.go         # SubscriptionHandler — CRUD + Dashboard + icon upload
  admin.go                # AdminHandler — user mgmt, stats, settings, DB backup/restore
  currency.go             # CurrencyHandler — user currency list CRUD + reorder
  exchange_rate.go        # ExchangeRateHandler — rates, user preference
internal/service/         # Business logic (one file per domain, no cross-service calls)
internal/model/model.go   # All GORM models + JSON tags
internal/pkg/
  database.go             # InitDB(), AutoMigrate, DB_PATH env
  jwt.go                  # JWTClaims, GenerateToken(), GetJWTSecret()
web/src/
  App.tsx                 # Routes: ProtectedRoute / PublicRoute / AdminRoute
  features/{domain}/      # Pages and domain components
  lib/api.ts              # Fetch wrapper — JWT attach, 401 redirect, toast errors
  lib/utils.ts            # cn(), formatCurrency(), formatDate(), daysUntil()
  types/index.ts          # All TS interfaces — mirrors Go JSON tags exactly
  i18n/{en,zh-CN,ja}.ts  # Translation keys (must stay in sync across all three)
  hooks/                  # Custom hooks (e.g. useSiteSettings); prefix with `use`
  components/ui/          # Shadcn primitives — DO NOT EDIT
```

## Cursor / Copilot Rules

- No `.cursorrules` found
- No `.cursor/rules/` found
- No `.github/copilot-instructions.md` found
- If added later, treat them as higher-priority instructions

## Build, Lint, Test Commands

Run from the listed working directory.

### Frontend (`web/`)

```sh
bun install           # Install deps
bun run dev           # Dev server (proxies /api → :8080)
bun run lint          # ESLint (typescript-eslint + react-hooks + react-refresh)
bun run build         # tsc -b && vite build  →  web/dist/
bun run preview       # Preview built dist
```

### Backend (repo root)

```sh
go vet ./...                          # Lint
go test ./...                         # All tests
go build -o subdux ./cmd/server       # Requires web/dist to exist
./subdux                              # Run server
```

### Single-Test Commands (Important)

```sh
# Run one Go test by name in a specific package
go test -run TestName ./internal/service

# Verbose + uncached
go test -run TestName -v -count=1 ./internal/service

# Frontend — no test script in package.json; bun handles it directly if tests exist
bun test
bun test -t "pattern"
bun test ./src/path/to/file.test.ts
```

## Critical Build Caveats

- `go build` requires `web/dist` to exist (`go:embed`) — build frontend first
- Safe build order:
  1. `cd web && bun install && bun run build`
  2. `cd .. && go build -o subdux ./cmd/server`
- `Dockerfile` references `web/bun.lockb` (stale) — repo has `web/bun.lock`
- `Dockerfile` Go image is `1.23`; `go.mod` requires `1.25.0`

## Architecture Rules

- Backend flow: `api/ → service/ → model/ + pkg/` — no cross-layer skips
- Request parsing and field validation in handlers (not services)
- Business/data logic exclusively in services
- GORM models and JSON tags in `internal/model/model.go`
- Admin auth enforced via `AdminMiddleware` in `router.go` — never bypass
- All middleware (CORS, logger, recover, JWT) configured in `SetupRoutes` / `main.go` only
- No service-to-service calls — handlers compose multiple services when needed
- `AuthService` holds passkey session map (`sync.Mutex`) — this is intentional stateful state

## Go Code Style

### Formatting & Imports

- `gofmt` formatting — no exceptions
- Standard library imports grouped before third-party (standard `goimports` style)
- Alias imports only when needed (e.g. `echojwt "github.com/labstack/echo-jwt/v4"`)

### Naming

- Exported: `PascalCase` (`AuthService`, `SetupRoutes`, `NewAuthHandler`)
- Unexported helpers: `camelCase` (`getUserID`, `seedDefaultSettings`, `validateIcon`)
- Handler structs: `XHandler`; constructors: `NewXHandler(deps...) *XHandler`
- Service structs: `XService`; constructors: `NewXService(db *gorm.DB) *XService`
- Input structs: `CreateXInput` (value fields), `UpdateXInput` (pointer fields for partial update)

### Contracts & Error Handling

- JSON tags: `snake_case` on all exported model/DTO fields
- Secrets/passwords: `json:"-"` — never serialized
- Handler error response: `echo.Map{"error": "message"}` — always a plain string
- HTTP status codes: `400` bad input · `401` unauthenticated · `403` forbidden · `404` not found · `409` conflict · `500` internal
- Services return `(*Model, error)` or `error` — never swallow errors silently
- Use `errors.New("message")` for domain errors; `fmt.Errorf("...: %w", err)` only when wrapping

### Data Layer

- All queries via GORM query builder — no raw SQL
- SQLite driver: `github.com/glebarez/sqlite` (CGO-free) — do not replace
- `AutoMigrate` runs on every startup — additive only, never destructive
- DB path default: `data/subdux.db` (override via `DB_PATH` env)
- FK style: `UserID uint \`gorm:"index;not null"\`` — no FK constraints defined
- When adding a new model, update `AutoMigrate(...)` in `pkg/database.go`

## TypeScript / React Style

### TypeScript Strict Config

`tsconfig.app.json` enforces: `strict`, `noUnusedLocals`, `noUnusedParameters`,
`noFallthroughCasesInSwitch`, `erasableSyntaxOnly`, `verbatimModuleSyntax`.
All must pass — never add `// @ts-ignore`, `// @ts-expect-error`, or cast to `any`.

### Imports

- Use `@/` alias for all app code (`@/lib/api`, `@/types`, `@/hooks/...`)
- Use `import type` for type-only imports (`import type { User } from "@/types"`)
- Never import Radix primitives directly — always use Shadcn wrappers

### Components & Pages

- Pages: `function XPage()` default export with local `useState` for all state
- Feature components: `web/src/features/{domain}/` in `kebab-case.tsx` files
- Type all component props explicitly: `interface Props { ... }`
- Route guards in `App.tsx`: `ProtectedRoute`, `PublicRoute`, `AdminRoute` — preserve these patterns
- Custom hooks in `web/src/hooks/` — prefix with `use`

### Types & API

- All entity/DTO types in `web/src/types/index.ts` — must match Go JSON tags exactly (`snake_case`)
- API calls: `api.get<T>()`, `api.post<T>()`, `api.put<T>()`, `api.delete<T>()`, `api.uploadFile<T>()`
- Errors are auto-toasted inside `lib/api.ts` — components must not re-toast the same error
- 401 auto-redirects to `/login` inside `lib/api.ts` — no manual redirect needed in components
- `isAuthenticated()` checks token existence only (not expiry)
- `isAdmin()` reads cached `"user"` from localStorage — role must equal `"admin"`

### Styling

- Tailwind v4 via `@tailwindcss/vite` plugin — no `tailwind.config.*` file
- Theme vars in `src/index.css` under `:root` and `.dark` (oklch color space)
- Shadcn: new-york variant, zinc palette — add components via `bunx shadcn@latest add {name}` from `web/`
- Use `cn()` from `@/lib/utils` for conditional class merging
- Never edit files under `web/src/components/ui/` — auto-generated, will be overwritten

## i18n Rules

- All user-facing strings must use translation keys via `useTranslation()` / `t()`
- Keep `en`, `zh-CN`, and `ja` in `web/src/i18n/` in sync whenever adding/changing copy
- Hardcoded UI strings in new features are not allowed

## Where to Look

| Task | Location |
|------|----------|
| New API endpoint | `internal/api/router.go` → handler method → service method |
| New model field | `internal/model/model.go` + `AutoMigrate` in `pkg/database.go` |
| New page | `web/src/features/{domain}/{name}-page.tsx` + route in `App.tsx` |
| New Shadcn component | `bunx shadcn@latest add {name}` from `web/` |
| API type contract | `web/src/types/index.ts` (must match Go JSON tags) |
| Auth/JWT config | `internal/pkg/jwt.go` |
| DB config / migration | `internal/pkg/database.go` |
| Admin settings seed | `seedDefaultSettings()` in `internal/api/router.go` |

## Agent Workflow Checklist

1. Find the nearest existing pattern in the touched layer before writing new code.
2. Keep backend JSON tags and TS interface fields in sync (`snake_case` both sides).
3. Run checks after any change:
   - Frontend touched: `bun run lint` then `bun run build` (both must exit 0)
   - Backend touched: `go vet ./...` then `go test ./...` then `go build -o subdux ./cmd/server`
4. If adding a Go test, note the exact single-test command.
5. If adding a new model, update `AutoMigrate` in `pkg/database.go`.

## Git Commit Rules

- Commit messages must be detailed, not single-line only.
- Use a short subject line plus a descriptive body explaining what changed and why.
- Include key impacted areas/files and notable behavior changes in the body.

## Invariants (Do Not Break)

- Never switch to a CGO-dependent SQLite driver
- Never break `go:embed` — `web/dist` must exist before `go build`
- Never mismatch Go JSON tags and TS contract field names
- Never bypass `AdminMiddleware` on admin routes
- Never edit `web/src/components/ui/` files
- Never import Radix UI primitives directly — use Shadcn wrappers
- Never add `// @ts-ignore`, `// @ts-expect-error`, or cast to `any`
