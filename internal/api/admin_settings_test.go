package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func newAdminSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-admin-settings-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings: %v", err)
	}
	return db
}

func TestUpdateSettingsRejectsInvalidEmailDomainWhitelist(t *testing.T) {
	e := echo.New()
	body := `{"email_domain_whitelist":"http://example.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsTooLongEmailDomainWhitelist(t *testing.T) {
	e := echo.New()
	domains := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		domains = append(domains, strings.Repeat("a", 59)+string(rune('a'+i))+".example.com")
	}
	body := `{"email_domain_whitelist":"` + strings.Join(domains, ";") + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidIconProxyDomainWhitelist(t *testing.T) {
	e := echo.New()
	body := `{"icon_proxy_domain_whitelist":"https://www.google.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidSSRFFilterMode(t *testing.T) {
	e := echo.New()
	body := `{"ssrf_domain_filter_mode":"deny"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateSettingsRejectsInvalidSSRFIPFilterList(t *testing.T) {
	e := echo.New()
	body := `{"ssrf_ip_filter_list":"10.0.0.0/99"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestAdminHandlerTestSSRFReturnsPolicyResult(t *testing.T) {
	e := echo.New()
	body := `{"target":"127.0.0.1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.TestSSRF(c); err != nil {
		t.Fatalf("TestSSRF() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var result service.SSRFTestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Allowed {
		t.Fatal("Allowed = true, want false for loopback target")
	}
	if result.Host != "127.0.0.1" {
		t.Fatalf("Host = %q, want 127.0.0.1", result.Host)
	}
	if !strings.Contains(result.Reason, "localhost or private network addresses") {
		t.Fatalf("Reason = %q, want restricted target reason", result.Reason)
	}
}

func TestAdminHandlerTestSSRFRejectsInvalidTarget(t *testing.T) {
	e := echo.New()
	body := `{"target":"http://exa mple.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.TestSSRF(c); err != nil {
		t.Fatalf("TestSSRF() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestAdminSSRFTestRouteRequiresAdminRole(t *testing.T) {
	t.Setenv("JWT_SECRET", "admin-ssrf-test-jwt-secret-0123456789")
	db := newAdminSettingsTestDB(t)
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())

	token, err := pkg.GenerateAccessToken(1, "alice", "alice@example.com", "user")
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/admin/settings/ssrf/test", strings.NewReader(`{"target":"127.0.0.1"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestUpdateSettingsRejectsTooLongIconProxyDomainWhitelist(t *testing.T) {
	e := echo.New()
	domains := make([]string, 0, 9)
	for i := 0; i < 9; i++ {
		domains = append(domains, strings.Repeat("a", 59)+string(rune('a'+i))+".example.com")
	}
	body := `{"icon_proxy_domain_whitelist":"` + strings.Join(domains, ";") + `"}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	db := newAdminSettingsTestDB(t)
	handler := &AdminHandler{Service: service.NewAdminService(db)}
	if err := handler.UpdateSettings(c); err != nil {
		t.Fatalf("UpdateSettings() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
