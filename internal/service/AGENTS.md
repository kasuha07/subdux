# Service Layer — Business Logic

**Generated:** 2026-02-22 13:12 UTC  
**Commit:** fdfaf8c  
**Branch:** main

## OVERVIEW

23 Go files implementing business logic for auth, subscriptions, notifications (email/webhook/ServerChan/Bark/Telegram), TOTP/passkey/OIDC, user defaults, and background refresh. Services receive GORM DB, return models or errors.

## STRUCTURE

```
service/
├── auth.go                    # Register, Login (bcrypt + JWT)
├── subscription.go            # CRUD, DashboardSummary (monthly cost calc)
├── subscription_schedule.go   # Billing date calculations (leap years, month-end)
├── notification*.go           # 8 files: validation, templates, channels, delivery
├── totp.go, passkey.go        # 2FA implementations
├── oidc.go                    # OAuth2/OIDC provider integration
├── user_defaults.go           # Seed default categories/payment methods
└── *_test.go                  # 8 test files (table-driven, helpers)
```

## WHERE TO LOOK

| Task | File | Key Functions |
|------|------|---------------|
| Add auth method | `auth.go` | `Register()`, `Login()` — bcrypt + JWT generation |
| Subscription CRUD | `subscription.go` | `Create()`, `Update()`, `Delete()`, `List()` |
| Billing calculations | `subscription_schedule.go` | `CalculateNextBillingDate()` — handles leap years |
| Notification delivery | `notification.go` | `SendNotification()` — dispatches to channels |
| Webhook validation | `notification_validation.go` | `ValidateWebhookConfig()` — header/URL checks |
| Email templates | `notification_template.go` | `RenderTemplate()` — Go templates |
| 2FA setup | `totp.go`, `passkey.go` | `GenerateTOTP()`, `RegisterPasskey()` |
| OAuth login | `oidc.go` | `HandleOIDCCallback()` |

## CONVENTIONS

### Constructor Pattern
```go
func NewXService(db *gorm.DB) *XService {
    return &XService{db: db}
}
```
- Receive DB only, no other dependencies
- No service-to-service calls (handlers compose)

### Input Structs
- `CreateXInput` — value fields (all required)
- `UpdateXInput` — pointer fields (partial updates, nil = no change)

### Return Patterns
- `(*Model, error)` for single results
- `([]Model, error)` for lists
- `error` for operations with no return value

### Dashboard Summary
Monthly cost calculation: `weekly×4.33 + monthly×1 + yearly÷12`

### Date Handling
- All dates normalized to start-of-day UTC
- Leap year aware (Feb 29 → Feb 28 on non-leap years)
- Month-end normalization (Jan 31 → Feb 28/29)

## TESTING

8 test files with 1,042 lines:
- **Pattern:** Table-driven with `t.Run()` sub-tests
- **Helpers:** `mustDate()`, `newTestDB()`, `createTestUser()` marked with `t.Helper()`
- **Database:** In-memory SQLite via `t.TempDir()`
- **Assertions:** Plain `if` + `t.Fatal*()`, no external libraries

## ANTI-PATTERNS

- **No raw SQL** — GORM query builder only
- **No cross-service calls** — services are independent
- **No validation in services** — handlers validate, services compute
