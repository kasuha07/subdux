# AGENTS.md

Operational guide for coding agents in this repository.
Use this as the primary cross-repo playbook.

## Project Snapshot

- Name: `subdux`
- Backend: Go + Echo + GORM + pure-Go SQLite (`github.com/glebarez/sqlite`)
- Frontend: React 19 + TypeScript + Vite 7 + Bun + shadcn/ui + Tailwind v4
- Runtime: single Go binary embedding `web/dist` and serving SPA + API
- API base path: `/api`

## Repository Map

- `cmd/server/main.go`: app boot, middleware, API route setup, SPA fallback serving
- `frontend.go`: `go:embed all:web/dist`
- `internal/api`: HTTP handlers + route registration
- `internal/service`: business logic + DB operations
- `internal/model`: GORM models and JSON tags
- `internal/pkg`: DB setup, JWT setup, token generation
- `web/src/features`: page/domain components
- `web/src/lib`: API client, theme, utilities
- `web/src/types/index.ts`: TS API contracts aligned with backend JSON fields

## Cursor / Copilot Rules

- No `.cursorrules` found
- No `.cursor/rules/` found
- No `.github/copilot-instructions.md` found
- If added later, treat them as higher-priority instructions

## Build, Lint, Test Commands

Run from the listed working directory.

### Frontend (`/home/shiroha/devs/subdux/web`)

- Install deps: `bun install`
- Dev: `bun run dev`
- Lint: `bun run lint`
- Build (tsc + vite): `bun run build`
- Preview: `bun run preview`

### Backend (`/home/shiroha/devs/subdux`)

- Vet: `go vet ./...`
- Build: `go build -o subdux ./cmd/server`
- Run: `./subdux`
- Test all: `go test ./...`

### Single-Test Commands (Important)

- Single Go test in package: `go test -run TestName ./path/to/package`
- Verbose uncached: `go test -run TestName -v -count=1 ./path/to/package`
- Repo example: `go test -run TestRegister ./internal/service`

Frontend tests status:

- No frontend test script exists in `web/package.json`
- Bun supports direct test execution if tests are present:
  - `bun test`
  - `bun test -t "test name pattern"`
  - `bun test ./src/path/to/file.test.ts`

## Critical Build Caveats

- `go build` requires `web/dist` to exist (`go:embed`)
- Safe local build order:
  1) `cd web && bun install && bun run build`
  2) `cd .. && go build -o subdux ./cmd/server`
- `Dockerfile` copies `web/bun.lockb`, but repo contains `web/bun.lock`
- `Dockerfile` Go image is `1.23`, module says `1.25.0`

## Architecture Rules

- Keep backend flow: `api -> service -> model/pkg`
- Request parsing and validation live in handlers
- Business/data logic lives in services
- Schema and tags live in `internal/model`
- Keep admin auth checks in middleware/route grouping

## Go Code Style

### Imports and formatting

- Follow `gofmt` formatting
- Standard library imports before third-party
- Alias imports only when needed (for clarity or collision)

### Naming and types

- Exported: PascalCase (`AuthService`, `SetupRoutes`)
- Internal helpers: camelCase (`getUserID`, `seedDefaultSettings`)
- Request structs use `XInput` / `UpdateXInput`
- Use pointer fields for partial update payloads

### Contracts and errors

- Keep JSON tags snake_case; mirror them in TS interfaces
- Hide secrets/passwords in JSON (`json:"-"`)
- Handler error format: `echo.Map{"error": "message"}`
- Use specific HTTP status codes (`400/401/403/404/500`)
- Do not swallow service/database errors silently

### Data layer

- Prefer GORM query builder
- Keep SQLite driver as `github.com/glebarez/sqlite` (CGO-free)
- Update `AutoMigrate` list when adding a model

## TypeScript / React Style

### Imports and modules

- Prefer `@/` alias imports for app code
- Use `import type` for type-only imports when practical
- Keep module constants near file top

### Component and state patterns

- Feature pages/components stay under `web/src/features/<domain>`
- Use local `useState`; use `useEffect`/`useCallback` for side effects
- Type component props explicitly (`interface Props`)
- Preserve route guard patterns in `web/src/App.tsx`

### Types and API usage

- Keep API entity/DTO types in `web/src/types/index.ts`
- Preserve snake_case TS fields to match backend JSON
- Avoid `any`; use typed API calls (`api.get<T>`, `api.post<T>`, etc.)

### Error and UI conventions

- Central API auth/error behavior stays in `web/src/lib/api.ts`
- Surface actionable user-facing errors (toasts/messages)
- Use shadcn UI primitives from `web/src/components/ui`
- Do not directly rewrite generated shadcn primitives unless asked

## i18n Rules

- Use translation keys for user-facing copy
- Keep `en`, `zh-CN`, and `ja` resources in sync
- Avoid hardcoded UI strings in new features

## Agent Workflow Checklist

1. Find the nearest existing pattern in touched layer.
2. Keep backend/frontend contracts consistent.
3. Run relevant checks:
   - Frontend touched: `bun run lint`, `bun run build`
   - Backend touched: `go vet ./...`, `go test ./...`, `go build -o subdux ./cmd/server`
4. If adding tests, include exact single-test command in notes.

## Invariants (Do Not Break)

- Do not switch to CGO-dependent SQLite drivers
- Do not break `go:embed` expectation for `web/dist`
- Do not mismatch Go JSON tags and TS contract fields
- Do not bypass auth/role checks on protected routes
