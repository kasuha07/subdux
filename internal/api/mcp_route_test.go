package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func newMCPRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.APIKey{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.SubscriptionActionSnooze{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.UserCurrency{},
		&model.UserPreference{},
		&model.ExchangeRate{},
		&model.AuditEvent{},
		&model.MCPIdempotencyKey{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func createMCPRouteTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "mcp-user",
		Email:    "mcp@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createMCPRouteAPIKey(t *testing.T, db *gorm.DB, user model.User) string {
	t.Helper()

	resp, err := apikeyservice.NewService(db).Create(user.ID, user.Role, apikeyservice.CreateInput{
		Name:    "Agent",
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}
	return resp.Key
}

func enableMCPRoute(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Where("key = ?", "mcp_enabled").
		Assign(model.SystemSetting{Value: "true"}).
		FirstOrCreate(&model.SystemSetting{Key: "mcp_enabled"}).Error; err != nil {
		t.Fatalf("failed to enable mcp: %v", err)
	}
}

func setupMCPRouteTest(t *testing.T, db *gorm.DB) *echo.Echo {
	t.Helper()

	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}
	e := echo.New()
	SetupRoutes(context.Background(), e, db, serviceutil.NewBackgroundTaskMonitor())
	return e
}

func TestSetupRoutesRegistersMCPAtRoot(t *testing.T) {
	db := newMCPRouteTestDB(t)
	user := createMCPRouteTestUser(t, db)
	apiKey := createMCPRouteAPIKey(t, db, user)
	enableMCPRoute(t, db)

	e := setupMCPRouteTest(t, db)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if _, ok := resp["result"]; !ok {
		t.Fatalf("response missing result: %#v", resp)
	}
}

func TestSetupRoutesMCPMethodNotAllowedStillValidatesHeaders(t *testing.T) {
	db := newMCPRouteTestDB(t)
	user := createMCPRouteTestUser(t, db)
	apiKey := createMCPRouteAPIKey(t, db, user)
	enableMCPRoute(t, db)

	e := setupMCPRouteTest(t, db)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotAcceptable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotAcceptable, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusUnsupportedMediaType, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec = httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusMethodNotAllowed, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestSetupRoutesMCPDisabledByDefault(t *testing.T) {
	db := newMCPRouteTestDB(t)
	user := createMCPRouteTestUser(t, db)
	apiKey := createMCPRouteAPIKey(t, db, user)

	e := setupMCPRouteTest(t, db)
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestSetupRoutesSiteInfoIncludesMCPEnabled(t *testing.T) {
	db := newMCPRouteTestDB(t)
	enableMCPRoute(t, db)

	e := setupMCPRouteTest(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/site-info", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["mcp_enabled"] != true {
		t.Fatalf("mcp_enabled = %v, want true", resp["mcp_enabled"])
	}
}
