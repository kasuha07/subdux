# Go Backend

## OVERVIEW

Four-package layered backend: `api/` (handlers) → `service/` (business logic) → `model/` (GORM structs) + `pkg/` (shared infra).

## STRUCTURE

```
internal/
├── api/
│   ├── router.go          # SetupRoutes() — route groups, JWT middleware, service wiring
│   ├── auth.go            # AuthHandler — Register, Login (public)
│   └── subscription.go    # SubscriptionHandler — CRUD + Dashboard (protected)
├── model/                 # GORM structs split by domain
├── pkg/
│   ├── database.go        # InitDB() — SQLite connection, auto-migrate
│   └── jwt.go             # JWTClaims, GenerateToken(), GetJWTSecret()
└── service/
    ├── auth.go            # AuthService — Register, Login (bcrypt + JWT)
    └── subscription.go    # SubscriptionService — CRUD, DashboardSummary
```

## WHERE TO LOOK

| Task | Start here | Then |
|------|-----------|------|
| Add new endpoint | `api/router.go` (add route) | Create handler method → service method |
| Add model field | `model/*_models.go` | Add GORM tag + json tag, restart to auto-migrate |
| Change auth rules | `pkg/jwt.go` (token config) | `service/auth.go` (validation logic) |
| Change DB config | `pkg/database.go` | Env var: `DATA_PATH` |

## CONVENTIONS

### Handler pattern (api/)
```go
// Handlers receive echo.Context, extract userID, delegate to service
func (h *SubscriptionHandler) Create(c echo.Context) error {
    userID := getUserID(c)          // Helper extracts from JWT claims
    var input service.CreateXInput  // Input types defined in service package
    if err := c.Bind(&input); err != nil { ... }
    // Validate → call service → return JSON
}
```
- Input validation lives in handlers (not service)
- `getUserID()` in `router.go` — shared helper, extracts `uint` from JWT token claims
- Error format: always `echo.Map{"error": "message"}`

### Service pattern (service/)
- Constructor: `NewXService(db *gorm.DB)` — receives DB, no other deps
- Input structs: `CreateXInput` (value fields), `UpdateXInput` (pointer fields for partial update)
- Services return `(*Model, error)` or `error`
- Dashboard summary calculates monthly cost: weekly×4.33, monthly×1, yearly÷12

### Model pattern (model/)
- GORM tags: `primaryKey`, `uniqueIndex`, `not null`, `size:N`, `default:'value'`
- JSON tags: `snake_case`, password tagged `json:"-"`
- No foreign key constraints defined — just `UserID uint` with index
- `CreatedAt`/`UpdatedAt` auto-managed by GORM

### Database (pkg/)
- Pure Go SQLite: `github.com/glebarez/sqlite` — CGO_ENABLED=0 safe
- `AutoMigrate` on every startup — additive only, no rollbacks
- Default data directory: `data/`, overridden by `DATA_PATH` env var (db at `$DATA_PATH/subdux.db`, assets at `$DATA_PATH/assets/`)
- Logger: Silent mode (no SQL logging)

## ANTI-PATTERNS

- **No raw SQL** — all queries via GORM query builder
- **No service-to-service calls** — handlers compose, services don't cross-depend
- **No middleware beyond router.go** — CORS, logger, recover, JWT all configured in SetupRoutes
