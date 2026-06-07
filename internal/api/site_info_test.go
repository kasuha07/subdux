package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

func TestSiteInfoHandlerGet(t *testing.T) {
	db := newAdminSettingsTestDB(t)
	if err := db.Create(&model.SystemSetting{Key: "site_name", Value: "Team Subdux"}).Error; err != nil {
		t.Fatalf("failed to seed site_name: %v", err)
	}
	if err := db.Create(&model.SystemSetting{Key: "mcp_enabled", Value: "true"}).Error; err != nil {
		t.Fatalf("failed to seed mcp_enabled: %v", err)
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/site-info", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := NewSiteInfoHandler(service.NewSystemSettingsService(db))

	if err := handler.Get(c); err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["site_name"] != "Team Subdux" {
		t.Fatalf("site_name = %v, want Team Subdux", resp["site_name"])
	}
	if resp["mcp_enabled"] != true {
		t.Fatalf("mcp_enabled = %v, want true", resp["mcp_enabled"])
	}
}
