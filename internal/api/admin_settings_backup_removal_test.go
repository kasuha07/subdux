package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
)

// removedBackupSettingKeys are the scheduled-backup fields the global admin
// settings payload used to carry. Every destination is now a self-contained
// backup plan owning its own schedule, so none of these may be published by
// GET /admin/settings or honoured by PUT /admin/settings any more.
var removedBackupSettingKeys = []string{
	"backup_schedule_enabled",
	"backup_time_of_day",
	"backup_include_assets",
	"backup_encrypt_enabled",
	"backup_encryption_password_configured",
	"backup_last_run_at",
	"backup_last_status",
	"backup_last_error",
}

// TestGetAdminSettingsOmitsRetiredBackupFields locks the read side of the
// settings contract. A leftover field would be worse than cosmetic: the admin
// UI would render a schedule control that no scheduler consults.
func TestGetAdminSettingsOmitsRetiredBackupFields(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	rec := getAdminSettings(t, e, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	assertNoBackupSettingKeys(t, rec.Body.String())
}

// TestUpdateSettingsIgnoresRetiredBackupFields pins the behaviour a stale
// client sees. The retired keys are simply absent from UpdateSettingsInput, so
// the JSON binder drops them and the request succeeds on its remaining fields
// rather than failing with a confusing validation error. Asserting that
// explicitly means a later switch to strict binding cannot change the contract
// unnoticed.
func TestUpdateSettingsIgnoresRetiredBackupFields(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	body := `{"backup_time_of_day":"04:30","backup_schedule_enabled":true,"site_name":"Backup Field Ignored"}`
	rec := putAdminSettings(t, e, token, body, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"settings_updated"`) {
		t.Fatalf("body = %s, want the settings_updated message code", rec.Body.String())
	}

	// The recognised field in the same payload was applied, which proves the
	// request was processed rather than skipped wholesale because of the
	// unknown keys.
	settings, err := adminservice.NewService(db).GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings.SiteName != "Backup Field Ignored" {
		t.Fatalf("SiteName = %q, want %q", settings.SiteName, "Backup Field Ignored")
	}

	// "Ignored" has to mean nothing was written: a retired key persisted under
	// its old name would be silently resurrected by any future reader.
	var stored []model.SystemSetting
	if err := db.Where("key LIKE ?", "backup%").Find(&stored).Error; err != nil {
		t.Fatalf("query persisted backup settings: %v", err)
	}
	if len(stored) != 0 {
		t.Fatalf("persisted backup settings = %+v, want none", stored)
	}

	// And the retired keys stay off the read side after the write attempt.
	rec = getAdminSettings(t, e, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("settings read status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	assertNoBackupSettingKeys(t, rec.Body.String())
}

// assertNoBackupSettingKeys checks the explicitly retired keys by name, then
// sweeps the whole backup_ prefix so a field cannot creep back under a new
// spelling.
func assertNoBackupSettingKeys(t *testing.T, body string) {
	t.Helper()

	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode settings response: %v; body = %s", err, body)
	}

	for _, key := range removedBackupSettingKeys {
		if _, ok := payload[key]; ok {
			t.Fatalf("settings response still exposes retired field %q; body = %s", key, body)
		}
	}

	for key := range payload {
		if strings.HasPrefix(key, "backup_") {
			t.Fatalf("settings response exposes backup field %q; body = %s", key, body)
		}
	}
}
