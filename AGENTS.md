# PROJECT KNOWLEDGE BASE

**Generated:** 2026-02-18
**Commit:** cba4d25
**Branch:** master

## OVERVIEW

Subdux — self-hosted subscription tracker. Go backend (Echo + GORM + pure-Go SQLite) serves a React 19 SPA via `go:embed`. Single binary, single container, zero external dependencies.

## STRUCTURE

```
subdux/
├── cmd/server/main.go     # Entry point — Echo setup, SPA handler, env config
├── frontend.go            # go:embed directive (embeds web/dist into binary)
├── internal/              # Go backend — see internal/AGENTS.md
│   ├── api/               # HTTP handlers + route registration
│   ├── model/             # GORM structs (User, Subscription)
│   ├── pkg/               # Shared infra (DB init, JWT)
│   └── service/           # Business logic layer
├── web/                   # React frontend — see web/AGENTS.md
│   └── src/
│       ├── components/ui/ # Shadcn/UI primitives (auto-generated)
│       ├── features/      # Page-level components by domain
│       ├── lib/           # API client, utilities
│       └── types/         # TypeScript interfaces
├── Dockerfile             # 3-stage: bun build → go build → distroless
├── docker-compose.yml     # Single service, named volume for SQLite
├── go.mod                 # Module: github.com/shiroha/subdux
└── data/                  # SQLite DB (gitignored, auto-created)
```

## WHERE TO LOOK

| Task | Location | Notes |
|------|----------|-------|
| Add API endpoint | `internal/api/` handler + `internal/service/` logic + `router.go` registration | Follow existing handler→service pattern |
| Add DB model field | `internal/model/model.go` + `web/src/types/index.ts` | GORM auto-migrates on startup |
| Add frontend page | `web/src/features/{domain}/` + route in `App.tsx` | Follow feature-folder convention |
| Add Shadcn component | `bunx shadcn@latest add {name}` from `web/` dir | Check components.json for config |
| Change auth flow | `internal/pkg/jwt.go` + `internal/service/auth.go` + `web/src/lib/api.ts` | JWT stored in localStorage |
| Modify SPA serving | `cmd/server/main.go` setupSPA function | Custom fallback, not Echo's static middleware |
| Docker build | `Dockerfile` | ⚠️ Go version mismatch: Dockerfile uses `golang:1.23-alpine`, go.mod says `1.25.0` |
| Environment config | `DB_PATH`, `JWT_SECRET`, `PORT` env vars | Defaults: `data/subdux.db`, hardcoded fallback secret, `8080` |

## CONVENTIONS

- **Architecture**: Build-time separation (separate frontend build), run-time monolith (single binary)
- **Embed pattern**: `frontend.go` at package root embeds `web/dist` → `cmd/server/main.go` reads via `fs.Sub(subdux.StaticFS, "web/dist")`
- **No CGO**: Uses `github.com/glebarez/sqlite` (wraps `modernc.org/sqlite`) — NOT `gorm.io/driver/sqlite`
- **Error responses**: All errors return `echo.Map{"error": "message"}` — no error codes, no structured errors
- **JSON tags**: Go structs use `snake_case` json tags matching TypeScript interfaces
- **Auth**: JWT in `Authorization: Bearer <token>` header, 72h expiry, HS256 signing
- **Frontend state**: Pure `useState` — no context, no external store
- **Path alias**: `@/` → `web/src/` in TypeScript, configured in vite.config.ts + tsconfig.app.json

## ANTI-PATTERNS (THIS PROJECT)

- **NEVER** use `gorm.io/driver/sqlite` — requires CGO, breaks distroless deployment
- **NEVER** import from `web/src/components/ui/` to modify — these are Shadcn auto-generated. Compose them in feature components
- **NEVER** add `as any` or `@ts-ignore` in TypeScript
- **NEVER** use `echo.Static()` for SPA — custom `setupSPA()` handles HTML5 history fallback

## UNIQUE STYLES

- **Shadcn new-york style** with zinc base color — oklch color system in `index.css`
- **Vercel-style minimalism**: Black & white, high contrast, sans-serif
- **Update inputs use pointer fields** (`*string`, `*float64`) for PATCH-like partial updates via PUT

## COMMANDS

```bash
# Frontend dev (from web/)
bun install              # Install deps
bun run dev              # Vite dev server (proxies /api → :8080)
bun run build            # TypeScript check + Vite build → web/dist/

# Backend dev (from root)
go vet ./...             # Lint
go build -o subdux ./cmd/server  # Build binary (needs web/dist/ to exist)
./subdux                 # Run on :8080

# Docker
docker compose up --build  # Full build + run
```

## NOTES

- `web/dist/` must exist before `go build` — the `go:embed` directive fails otherwise
- Dockerfile lockfile is `bun.lockb` but actual file is `bun.lock` — build may fail
- No tests exist yet — no test infrastructure on either side
- GORM `AutoMigrate` runs on every startup — schema changes are additive only (no down migrations)
- JWT default secret is hardcoded: `subdux-default-secret-change-in-production`
- `isAuthenticated()` only checks token existence, not expiry — expired tokens get 401 from backend, then auto-redirect
