# API Middleware

**Updated:** 2026-07-04
**Commit:** b895c6b
**Branch:** main

## OVERVIEW

`internal/api/apimw` owns API authentication middleware, request principal derivation, rate limiting, request/body/origin guards, and shared HTTP security headers. This is the transport boundary for deciding who the caller is before handler logic runs.

## STRUCTURE

```text
apimw/
├── auth.go                 # AdminMiddleware, JWTOrAPIKeyMiddleware, MCPEnabledMiddleware
├── principal.go            # Principal derivation and request-scoped identity cache
├── security.go             # Scope checks, human-only gate, rate limits, headers, body limits
├── cookies.go              # Auth/refresh cookie helpers
└── security_internal_test.go
```

## WHERE TO LOOK

| Task | File | Notes |
|------|------|-------|
| JWT vs API-key auth flow | `auth.go` | Preserve Bearer-first then `X-API-Key` fallback behavior |
| Principal fields/role downgrade | `principal.go` | API-key principals must not inherit admin/human role privileges |
| API-key scope or human-only gate | `security.go` | Keep hostile-path tests for API keys and missing scopes |
| Rate limits | `security.go` | Reuse existing account-key extractors before adding new limiter shapes |
| Request size/origin/security headers | `security.go` | Keep explicit skipper usage narrow and auditable |
| Refresh/auth cookies | `cookies.go` | Preserve current cookie flags and lifetime behavior |

## CONVENTIONS

- Derive request identity through `Principal` and `From(c)`. Do not reparse JWT claims in handlers.
- API-key principals are machine principals. Their effective role remains least-privilege `user`.
- Keep middleware ordering explicit in `internal/api/router.go`; do not hide route-boundary changes inside unrelated helpers.
- `HumanSessionOnlyMiddleware` should return the established human-session error rather than silently treating API keys as users.
- `RequestBodyLimitMiddleware` skippers should remain path-specific and narrowly justified.
- Reuse existing limiter key extractors such as `LoginAccountKey`, `RefreshTokenAccountKey`, and `AuthenticatedUserAccountKey` where appropriate.

## TESTING

```bash
go test -count=1 ./internal/api/apimw ./internal/api
```

For auth or boundary changes, add negative tests for API keys, bad scopes, missing auth, cross-origin requests, and oversized bodies.
