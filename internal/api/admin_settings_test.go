package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
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
