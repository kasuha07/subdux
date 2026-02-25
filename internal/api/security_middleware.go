package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

type fixedWindowEntry struct {
	Count       int
	WindowStart time.Time
	LastSeen    time.Time
}

type fixedWindowLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	entries map[string]fixedWindowEntry
	ops     int
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		limit:   limit,
		window:  window,
		entries: make(map[string]fixedWindowEntry),
	}
}

func (l *fixedWindowLimiter) Allow(key string) bool {
	if key == "" {
		return true
	}

	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, exists := l.entries[key]
	if !exists || now.Sub(entry.WindowStart) >= l.window {
		l.entries[key] = fixedWindowEntry{
			Count:       1,
			WindowStart: now,
			LastSeen:    now,
		}
		l.maybeCleanup(now)
		return true
	}

	if entry.Count >= l.limit {
		entry.LastSeen = now
		l.entries[key] = entry
		l.maybeCleanup(now)
		return false
	}

	entry.Count++
	entry.LastSeen = now
	l.entries[key] = entry
	l.maybeCleanup(now)
	return true
}

func (l *fixedWindowLimiter) maybeCleanup(now time.Time) {
	l.ops++
	if l.ops%200 != 0 {
		return
	}

	expireBefore := now.Add(-3 * l.window)
	for key, entry := range l.entries {
		if entry.LastSeen.Before(expireBefore) {
			delete(l.entries, key)
		}
	}
}

func authIPRateLimit(limit int, window time.Duration) echo.MiddlewareFunc {
	limiter := newFixedWindowLimiter(limit, window)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			clientIP := strings.TrimSpace(c.RealIP())
			if clientIP == "" {
				clientIP = "unknown"
			}

			if !limiter.Allow("ip:" + clientIP) {
				return c.JSON(http.StatusTooManyRequests, echo.Map{"error": "too many attempts, please try again later"})
			}

			return next(c)
		}
	}
}

type accountKeyExtractor func(c echo.Context) string

const (
	maxAuthRequestBodyBytes    int64 = 128 << 10 // 128 KiB
	maxAccountKeyReadBodyBytes int64 = 8 << 10   // 8 KiB
)

func authAccountRateLimit(limit int, window time.Duration, extractor accountKeyExtractor) echo.MiddlewareFunc {
	limiter := newFixedWindowLimiter(limit, window)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			accountKey := extractor(c)
			if accountKey == "" {
				return next(c)
			}

			if !limiter.Allow("acct:" + accountKey) {
				return c.JSON(http.StatusTooManyRequests, echo.Map{"error": "too many attempts for this account, please try again later"})
			}

			return next(c)
		}
	}
}

func readInputField(c echo.Context, field string) string {
	body, err := readRequestBodyAndRestore(c, maxAccountKeyReadBodyBytes)
	if err != nil {
		return ""
	}

	if value := readJSONField(body, field); value != "" {
		return value
	}

	if value := readFormField(body, field); value != "" {
		return value
	}

	return strings.TrimSpace(c.QueryParam(field))
}

func readJSONField(body []byte, field string) string {
	if len(body) == 0 {
		return ""
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	raw, ok := payload[field]
	if !ok {
		return ""
	}

	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}

	return strings.TrimSpace(value)
}

func readFormField(body []byte, field string) string {
	if len(body) == 0 {
		return ""
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(values.Get(field))
}

func readRequestBodyAndRestore(c echo.Context, maxBytes int64) ([]byte, error) {
	req := c.Request()
	if req.Body == nil {
		return nil, nil
	}

	if maxBytes <= 0 {
		return nil, nil
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), req.Body))

	if int64(len(body)) > maxBytes {
		return nil, nil
	}

	return body, nil
}

func loginAccountKey(c echo.Context) string {
	return strings.ToLower(strings.TrimSpace(readInputField(c, "identifier")))
}

func registerAccountKey(c echo.Context) string {
	email := strings.ToLower(strings.TrimSpace(readInputField(c, "email")))
	if email != "" {
		return "email:" + email
	}

	username := strings.ToLower(strings.TrimSpace(readInputField(c, "username")))
	if username != "" {
		return "username:" + username
	}

	return ""
}

func emailAccountKey(c echo.Context) string {
	email := strings.ToLower(strings.TrimSpace(readInputField(c, "email")))
	if email == "" {
		return ""
	}
	return "email:" + email
}

func totpAccountKey(c echo.Context) string {
	totpToken := strings.TrimSpace(readInputField(c, "totp_token"))
	if totpToken == "" {
		return ""
	}

	if userID, err := pkg.ValidateTOTPPendingToken(totpToken); err == nil && userID > 0 {
		return "user:" + strconv.FormatUint(uint64(userID), 10)
	}

	hash := sha256.Sum256([]byte(totpToken))
	return "totp:" + hex.EncodeToString(hash[:8])
}

func refreshTokenAccountKey(c echo.Context) string {
	refreshToken := strings.TrimSpace(readInputField(c, "refresh_token"))
	if refreshToken == "" {
		return ""
	}
	return "refresh:" + pkg.HashRefreshToken(refreshToken)
}

func APIKeyScopeMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if getAuthType(c) != pkg.AuthTypeAPIKey {
			return next(c)
		}

		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}

		if !isAPIKeyRouteAllowed(path) {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "api key cannot access this endpoint"})
		}

		requiredScope := requiredAPIKeyScope(c)

		if hasAPIKeyScope(c, requiredScope) {
			return next(c)
		}

		return c.JSON(http.StatusForbidden, echo.Map{"error": "api key does not have required scope"})
	}
}

func isAPIKeyRouteAllowed(path string) bool {
	if path == "/api/auth" {
		return false
	}

	if strings.HasPrefix(path, "/api/auth/") {
		return path == "/api/auth/me"
	}

	return true
}

var writeScopeRoutes = map[string]struct{}{
	"/api/auth/totp/setup": {},
}

func requiredAPIKeyScope(c echo.Context) string {
	path := c.Path()
	if path == "" {
		path = c.Request().URL.Path
	}

	if _, ok := writeScopeRoutes[path]; ok {
		return service.APIKeyScopeWrite
	}

	if isReadOnlyMethod(c.Request().Method) {
		return service.APIKeyScopeRead
	}

	return service.APIKeyScopeWrite
}

func isReadOnlyMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func securityHeadersMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set(echo.HeaderXContentTypeOptions, "nosniff")
		h.Set(echo.HeaderXFrameOptions, "DENY")
		h.Set(echo.HeaderContentSecurityPolicy, "default-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		return next(c)
	}
}

func requestBodyLimitMiddleware(maxBytes int64, skipper func(echo.Context) bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if skipper != nil && skipper(c) {
				return next(c)
			}
			if maxBytes > 0 && c.Request().Body != nil {
				c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxBytes)
			}
			return next(c)
		}
	}
}
