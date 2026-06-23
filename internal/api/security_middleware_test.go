package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func newSecurityMiddlewareTestContext(method string, target string, contentType string, body string) echo.Context {
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set(echo.HeaderContentType, contentType)
	}
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestLoginAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/login",
		echo.MIMEApplicationForm,
		"identifier=Alice%40Example.com&password=secret",
	)

	got := loginAccountKey(c)
	want := "alice@example.com"
	if got != want {
		t.Fatalf("loginAccountKey() = %q, want %q", got, want)
	}
}

func TestRegisterAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/register",
		echo.MIMEApplicationForm,
		"email=Test%2Balias%40Example.com&username=alice",
	)

	got := registerAccountKey(c)
	want := "email:test+alias@example.com"
	if got != want {
		t.Fatalf("registerAccountKey() = %q, want %q", got, want)
	}
}

func TestEmailAccountKeyReadsFormEncodedBody(t *testing.T) {
	c := newSecurityMiddlewareTestContext(
		http.MethodPost,
		"/api/auth/password/forgot",
		echo.MIMEApplicationForm,
		"email=Recover%40Example.com",
	)

	got := emailAccountKey(c)
	want := "email:recover@example.com"
	if got != want {
		t.Fatalf("emailAccountKey() = %q, want %q", got, want)
	}
}

func TestReadRequestBodyAndRestorePreservesBodyForDownstream(t *testing.T) {
	body := `{"identifier":"Alice@example.com","password":"secret"}`
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMEApplicationJSON, body)

	readBody, err := readRequestBodyAndRestore(c, 1024)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if string(readBody) != body {
		t.Fatalf("readRequestBodyAndRestore() body = %q, want %q", string(readBody), body)
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreSkipsLargeFixedLengthBody(t *testing.T) {
	body := strings.Repeat("a", 128)
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/auth/login", echo.MIMETextPlain, body)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreDoesNotTrustContentLengthForLimiterReads(t *testing.T) {
	body := strings.Repeat("z", 128)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.ContentLength = 1
	req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestReadRequestBodyAndRestoreSkipsLargeUnknownLengthBody(t *testing.T) {
	body := strings.Repeat("x", 64)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	req.ContentLength = -1
	req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	readBody, err := readRequestBodyAndRestore(c, 32)
	if err != nil {
		t.Fatalf("readRequestBodyAndRestore() error = %v, want nil", err)
	}
	if len(readBody) != 0 {
		t.Fatalf("readRequestBodyAndRestore() returned %d bytes, want 0", len(readBody))
	}

	downstreamBody, err := io.ReadAll(c.Request().Body)
	if err != nil {
		t.Fatalf("downstream io.ReadAll() error = %v, want nil", err)
	}
	if string(downstreamBody) != body {
		t.Fatalf("downstream body = %q, want %q", string(downstreamBody), body)
	}
}

func TestRequiredAPIKeyScopeUsesWriteForStateChangingGetRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/totp/setup", "", "")
	c.SetPath("/api/auth/totp/setup")

	got := requiredAPIKeyScope(c)
	if got != service.APIKeyScopeWrite {
		t.Fatalf("requiredAPIKeyScope() = %q, want %q", got, service.APIKeyScopeWrite)
	}
}

func TestRequiredAPIKeyScopeUsesReadForRegularGetRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/subscriptions", "", "")
	c.SetPath("/api/subscriptions")

	got := requiredAPIKeyScope(c)
	if got != service.APIKeyScopeRead {
		t.Fatalf("requiredAPIKeyScope() = %q, want %q", got, service.APIKeyScopeRead)
	}
}

func TestIsAPIKeyRouteAllowed(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/subscriptions", want: true},
		{path: "/api/auth", want: false},
		{path: "/api/auth/me", want: true},
		{path: "/api/auth/totp/setup", want: false},
		{path: "/api/auth/passkeys/register/start", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isAPIKeyRouteAllowed(tt.path)
			if got != tt.want {
				t.Fatalf("isAPIKeyRouteAllowed(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestHumanSessionOnlyMiddlewareBlocksAPIKeyPrincipal(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/api-keys", "", "")
	c.SetPath("/api/api-keys")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeRead, service.APIKeyScopeWrite},
		},
	})

	calledNext := false
	middleware := HumanSessionOnlyMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("HumanSessionOnlyMiddleware() error = %v, want nil", err)
	}

	if calledNext {
		t.Fatal("HumanSessionOnlyMiddleware() called next handler for api key principal, want blocked")
	}

	if got := c.Response().Status; got != http.StatusForbidden {
		t.Fatalf("HumanSessionOnlyMiddleware() status = %d, want %d", got, http.StatusForbidden)
	}
}

func TestHumanSessionOnlyMiddlewareAllowsHumanSession(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodPost, "/api/api-keys", "", "")
	c.SetPath("/api/api-keys")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeUser,
		},
	})

	calledNext := false
	middleware := HumanSessionOnlyMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("HumanSessionOnlyMiddleware() error = %v, want nil", err)
	}

	if !calledNext {
		t.Fatal("HumanSessionOnlyMiddleware() blocked human session, want allowed")
	}

	if got := c.Response().Status; got != http.StatusNoContent {
		t.Fatalf("HumanSessionOnlyMiddleware() status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestHumanOnlyRoutesBlockAPIKeyPrincipal(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	apiKeyResp, err := service.NewAPIKeyService(db).Create(user.ID, user.Role, service.CreateAPIKeyInput{
		Name:    "Agent",
		KeyKind: service.APIKeyKindAPIIntegration,
		Scopes:  []string{service.APIKeyScopeRead, service.APIKeyScopeWrite},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	e := newHumanOnlyRouteTestServer(t, db)
	tests := []struct {
		name   string
		method string
		target string
	}{
		{
			name:   "change password",
			method: http.MethodPut,
			target: "/api/auth/password",
		},
		{
			name:   "send email change code",
			method: http.MethodPost,
			target: "/api/auth/email/change/send-code",
		},
		{
			name:   "confirm email change",
			method: http.MethodPost,
			target: "/api/auth/email/change/confirm",
		},
		{
			name:   "setup totp",
			method: http.MethodGet,
			target: "/api/auth/totp/setup",
		},
		{
			name:   "confirm totp",
			method: http.MethodPost,
			target: "/api/auth/totp/confirm",
		},
		{
			name:   "disable totp",
			method: http.MethodPost,
			target: "/api/auth/totp/disable",
		},
		{
			name:   "list passkeys",
			method: http.MethodGet,
			target: "/api/auth/passkeys",
		},
		{
			name:   "begin passkey registration",
			method: http.MethodPost,
			target: "/api/auth/passkeys/register/start",
		},
		{
			name:   "finish passkey registration",
			method: http.MethodPost,
			target: "/api/auth/passkeys/register/finish",
		},
		{
			name:   "delete passkey",
			method: http.MethodDelete,
			target: "/api/auth/passkeys/1",
		},
		{
			name:   "list oidc connections",
			method: http.MethodGet,
			target: "/api/auth/oidc/connections",
		},
		{
			name:   "begin oidc connect",
			method: http.MethodPost,
			target: "/api/auth/oidc/connect/start",
		},
		{
			name:   "delete oidc connection",
			method: http.MethodDelete,
			target: "/api/auth/oidc/connections/1",
		},
		{
			name:   "list api keys",
			method: http.MethodGet,
			target: "/api/api-keys",
		},
		{
			name:   "create api key",
			method: http.MethodPost,
			target: "/api/api-keys",
		},
		{
			name:   "delete api key",
			method: http.MethodDelete,
			target: "/api/api-keys/1",
		},
		{
			name:   "list audit events",
			method: http.MethodGet,
			target: "/api/audit-events",
		},
		{
			name:   "list calendar tokens",
			method: http.MethodGet,
			target: "/api/calendar/tokens",
		},
		{
			name:   "create calendar token",
			method: http.MethodPost,
			target: "/api/calendar/tokens",
		},
		{
			name:   "delete calendar token",
			method: http.MethodDelete,
			target: "/api/calendar/tokens/1",
		},
		{
			name:   "export",
			method: http.MethodGet,
			target: "/api/export",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.target, nil)
			req.Header.Set("X-API-Key", apiKeyResp.Key)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "human session required") {
				t.Fatalf("body = %s, want human session required error", rec.Body.String())
			}
		})
	}
}

func TestMCPClientAPIKeyCannotAccessRESTBusinessRoutes(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	apiKeyResp, err := service.NewAPIKeyService(db).Create(user.ID, user.Role, service.CreateAPIKeyInput{
		Name:    "MCP",
		KeyKind: service.APIKeyKindMCPClient,
		Scopes:  []string{service.APIKeyScopeRead, service.APIKeyScopeWrite},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/subscriptions", nil)
	req.Header.Set("X-API-Key", apiKeyResp.Key)
	rec := httptest.NewRecorder()
	newHumanOnlyRouteTestServer(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestAuditEventsUserEndpointOnlyReturnsOwnEvents(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	other := model.User{
		Username: "other-audit-user",
		Email:    "other-audit@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	if err := db.Create(&model.AuditEvent{
		EventID:      "own",
		UserID:       user.ID,
		KeyID:        1,
		KeyKind:      service.APIKeyKindMCPClient,
		ScopeUsed:    service.APIKeyScopeWrite,
		Transport:    service.AuditTransportMCP,
		ToolName:     "create_subscription",
		ResourceType: service.AuditResourceSubscription,
		Action:       "create",
		Status:       service.AuditStatusSuccess,
		OccurredAt:   pkg.NowUTC(),
	}).Error; err != nil {
		t.Fatalf("failed to create own audit event: %v", err)
	}
	if err := db.Create(&model.AuditEvent{
		EventID:      "other",
		UserID:       other.ID,
		KeyID:        2,
		KeyKind:      service.APIKeyKindMCPClient,
		ScopeUsed:    service.APIKeyScopeWrite,
		Transport:    service.AuditTransportMCP,
		ToolName:     "delete_subscription",
		ResourceType: service.AuditResourceSubscription,
		Action:       "delete",
		Status:       service.AuditStatusSuccess,
		OccurredAt:   pkg.NowUTC(),
	}).Error; err != nil {
		t.Fatalf("failed to create other audit event: %v", err)
	}

	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/audit-events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	newHumanOnlyRouteTestServer(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"event_id":"own"`) {
		t.Fatalf("body = %s, want own event", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"event_id":"other"`) {
		t.Fatalf("body = %s, want no other-user event", rec.Body.String())
	}
}

func TestHumanOnlyRoutesAllowHumanSession(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	newHumanOnlyRouteTestServer(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestAPIKeyAllowedRoutesStillAcceptAPIKeyPrincipal(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createHumanOnlyRouteTestUser(t, db)
	apiKeyResp, err := service.NewAPIKeyService(db).Create(user.ID, user.Role, service.CreateAPIKeyInput{
		Name:    "Reader",
		KeyKind: service.APIKeyKindAPIIntegration,
		Scopes:  []string{service.APIKeyScopeRead},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	e := newHumanOnlyRouteTestServer(t, db)
	tests := []struct {
		name   string
		target string
	}{
		{name: "auth me", target: "/api/auth/me"},
		{name: "subscriptions", target: "/api/subscriptions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Header.Set("X-API-Key", apiKeyResp.Key)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

func newHumanOnlyRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("JWT_SECRET", "human-only-route-test-jwt-secret-0123456789")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "human-only-route-test-settings-key")

	dbPath := filepath.Join(t.TempDir(), "subdux-human-only-route-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.APIKey{},
		&model.RefreshToken{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationLog{},
		&model.NotificationTemplate{},
		&model.CalendarToken{},
		&model.PasskeyCredential{},
		&model.OIDCConnection{},
		&model.EmailVerificationCode{},
		&model.UserBackupCode{},
		&model.AuditEvent{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	return db
}

func createHumanOnlyRouteTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "human-route-user",
		Email:    "human-route@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func newHumanOnlyRouteTestServer(t *testing.T, db *gorm.DB) *echo.Echo {
	t.Helper()

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())
	return e
}

func TestAPIKeyScopeMiddlewareBlocksAuthNamespaceRoot(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth", "", "")
	c.SetPath("/api/auth")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeWrite},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if calledNext {
		t.Fatal("APIKeyScopeMiddleware() called next handler for /api/auth, want blocked")
	}

	if got := c.Response().Status; got != http.StatusForbidden {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusForbidden)
	}
}

func TestAPIKeyScopeMiddlewareBlocksRestrictedAuthRoutes(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/totp/setup", "", "")
	c.SetPath("/api/auth/totp/setup")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeWrite},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if calledNext {
		t.Fatal("APIKeyScopeMiddleware() called next handler for restricted auth route, want blocked")
	}

	if got := c.Response().Status; got != http.StatusForbidden {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusForbidden)
	}
}

func TestAPIKeyScopeMiddlewareAllowsAuthMeRoute(t *testing.T) {
	c := newSecurityMiddlewareTestContext(http.MethodGet, "/api/auth/me", "", "")
	c.SetPath("/api/auth/me")
	c.Set("user", &jwt.Token{
		Claims: &pkg.JWTClaims{
			AuthType: pkg.AuthTypeAPIKey,
			Scopes:   []string{service.APIKeyScopeRead},
		},
	})

	calledNext := false
	middleware := APIKeyScopeMiddleware(func(c echo.Context) error {
		calledNext = true
		return c.NoContent(http.StatusNoContent)
	})

	if err := middleware(c); err != nil {
		t.Fatalf("APIKeyScopeMiddleware() error = %v, want nil", err)
	}

	if !calledNext {
		t.Fatal("APIKeyScopeMiddleware() blocked /api/auth/me for api key, want allowed")
	}

	if got := c.Response().Status; got != http.StatusNoContent {
		t.Fatalf("APIKeyScopeMiddleware() status = %d, want %d", got, http.StatusNoContent)
	}
}
