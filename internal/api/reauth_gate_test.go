package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// The backup/restore step-up gates live in AdminHandler.BackupDB and
// AdminHandler.RestoreDB, which call ReauthService.Consume before doing any
// work. The service-level Consume tests prove ticket semantics in isolation;
// these tests prove the wiring end-to-end through the router — that the
// sensitive endpoints actually refuse a request with no ticket, a
// wrong-operation ticket, or an already-spent ticket, and only proceed once a
// valid, operation-matched ticket is presented.
//
// The router builds its own ReauthService internally, so the only way to obtain
// a ticket the handler will accept is to mint one through the real
// /api/reauth/password endpoint against the same instance. That requires
// the admin to have a genuine bcrypt password (unlike createReauthTestAdmin,
// whose stored password is not a valid hash).

const reauthGateTestPassword = "s3cret-passphrase"

func createReauthGateTestAdmin(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(reauthGateTestPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	admin := model.User{
		Username: "reauth-gate-admin",
		Email:    "reauth-gate-admin@example.com",
		Password: string(hash),
		Role:     "admin",
		Status:   "active",
	}
	if err := db.Create(&admin).Error; err != nil {
		t.Fatalf("failed to create admin user: %v", err)
	}
	return admin
}

func reauthGateTestToken(t *testing.T, admin model.User) string {
	t.Helper()
	token, err := pkg.GenerateAccessToken(admin.ID, admin.Username, admin.Email, admin.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return token
}

// mintReauthTicket obtains a real, consumable ticket for operation by driving
// the password step-up endpoint on the same router the gate reads from.
func mintReauthTicket(t *testing.T, e *echo.Echo, token, operation string) string {
	t.Helper()

	body := fmt.Sprintf(`{"operation":%q,"password":%q}`, operation, reauthGateTestPassword)
	req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("mint ticket status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode ticket response: %v; body = %s", err, rec.Body.String())
	}
	if resp.Ticket == "" {
		t.Fatalf("mint ticket returned empty ticket; body = %s", rec.Body.String())
	}
	return resp.Ticket
}

func postBackup(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"include_assets":false,"password":"","reauth_ticket":%q}`, ticket)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/backup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func putAdminSettings(t *testing.T, e *echo.Echo, token, body, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postRestore(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	// No multipart body: a request that clears the gate falls through to
	// "no file uploaded", which is how we distinguish "gate passed" from
	// "gate refused" without performing a destructive database replace.
	req := httptest.NewRequest(http.MethodPost, "/api/admin/restore", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postSubduxImport(t *testing.T, e *echo.Echo, token string, confirm bool, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"data":{"currencies":[],"categories":[],"payment_methods":[],"subscriptions":[],"notifications":{"channels":[],"templates":[]}},"confirm":%t}`, confirm)
	req := httptest.NewRequest(http.MethodPost, "/api/import/subdux", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postWallosImport(t *testing.T, e *echo.Echo, token string, confirm bool, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"data":[],"confirm":%t}`, confirm)
	req := httptest.NewRequest(http.MethodPost, "/api/import/wallos", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postWallosImportWithAPIKey(t *testing.T, e *echo.Echo, apiKey string, confirm bool, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"data":[],"confirm":%t}`, confirm)
	req := httptest.NewRequest(http.MethodPost, "/api/import/wallos", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-API-Key", apiKey)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postCreateAPIKey(t *testing.T, e *echo.Echo, token, name, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"name":%q,"key_kind":"api_integration","scopes":["read"]}`, name)
	req := httptest.NewRequest(http.MethodPost, "/api/api-keys", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(reauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestBackupDBGateRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postBackup(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		restoreTicket := mintReauthTicket(t, e, token, "restore")
		rec := postBackup(t, e, token, restoreTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, "backup")

		rec := postBackup(t, e, token, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// The same ticket must not authorize a second backup.
		rec = postBackup(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestCreateAPIKeyRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postCreateAPIKey(t, e, token, "Script", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, service.ReauthOperationBackup)
		rec := postCreateAPIKey(t, e, token, "Script", backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, service.ReauthOperationCreateAPIKey)

		rec := postCreateAPIKey(t, e, token, "Script", ticket)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		rec = postCreateAPIKey(t, e, token, "Second Script", ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestAPIKeyListAndDeleteDoNotRequireReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	apiKeyResp, err := service.NewAPIKeyService(db).Create(admin.ID, admin.Role, service.CreateAPIKeyInput{
		Name:    "Integration",
		KeyKind: service.APIKeyKindAPIIntegration,
		Scopes:  []string{service.APIKeyScopeRead},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/api-keys", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/api-keys/%d", apiKeyResp.APIKey.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestUpdateSettingsBackupScheduleGateRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("ordinary settings do not require reauth", func(t *testing.T) {
		rec := putAdminSettings(t, e, token, `{"site_name":"Subdux Test"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := putAdminSettings(t, e, token, `{"backup_schedule_enabled":true}`, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}

		settings, err := service.NewAdminService(db).GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		if settings.BackupScheduleEnabled {
			t.Fatal("BackupScheduleEnabled = true after refused update, want false")
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, service.ReauthOperationBackup)
		rec := putAdminSettings(t, e, token, `{"backup_time_of_day":"04:30"}`, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, service.ReauthOperationBackupSchedule)
		body := `{"backup_schedule_enabled":true,"backup_time_of_day":"04:30","backup_include_assets":true,"backup_retention_count":3}`

		rec := putAdminSettings(t, e, token, body, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		settings, err := service.NewAdminService(db).GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		if !settings.BackupScheduleEnabled {
			t.Fatal("BackupScheduleEnabled = false, want true")
		}
		if settings.BackupTimeOfDay != "04:30" {
			t.Fatalf("BackupTimeOfDay = %q, want 04:30", settings.BackupTimeOfDay)
		}
		if !settings.BackupIncludeAssets {
			t.Fatal("BackupIncludeAssets = false, want true")
		}
		if settings.BackupRetentionCount != 3 {
			t.Fatalf("BackupRetentionCount = %d, want 3", settings.BackupRetentionCount)
		}

		rec = putAdminSettings(t, e, token, `{"backup_time_of_day":"05:15"}`, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestImportWallosGateRequiresValidReauthTicketOnConfirm(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("preview does not require reauth", func(t *testing.T) {
		rec := postWallosImport(t, e, token, false, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("missing ticket is refused on confirm", func(t *testing.T) {
		rec := postWallosImport(t, e, token, true, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused on confirm", func(t *testing.T) {
		subduxTicket := mintReauthTicket(t, e, token, service.ReauthOperationImportSubdux)
		rec := postWallosImport(t, e, token, true, subduxTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, service.ReauthOperationImportWallos)

		rec := postWallosImport(t, e, token, true, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec = postWallosImport(t, e, token, true, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("api key can preview but cannot confirm", func(t *testing.T) {
		apiKeyResp, err := service.NewAPIKeyService(db).Create(admin.ID, admin.Role, service.CreateAPIKeyInput{
			Name:    "Integration",
			KeyKind: service.APIKeyKindAPIIntegration,
			Scopes:  []string{service.APIKeyScopeRead, service.APIKeyScopeWrite},
		})
		if err != nil {
			t.Fatalf("failed to create api key: %v", err)
		}

		rec := postWallosImportWithAPIKey(t, e, apiKeyResp.Key, false, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("preview status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		ticket := mintReauthTicket(t, e, token, service.ReauthOperationImportWallos)
		rec = postWallosImportWithAPIKey(t, e, apiKeyResp.Key, true, ticket)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("confirm status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "human session required") {
			t.Fatalf("body = %s, want human session required", rec.Body.String())
		}
	})
}

func TestRestoreDBGateRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postRestore(t, e, token, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, "backup")
		rec := postRestore(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket clears the gate and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, "restore")

		// A valid ticket passes the gate; the request then fails downstream on
		// the missing upload rather than on re-auth, proving the gate let it
		// through.
		rec := postRestore(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, ticket should have cleared the gate", rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "no file uploaded") {
			t.Fatalf("body = %s, want no file uploaded", rec.Body.String())
		}

		// The gate consumed the ticket even though the request failed after it,
		// so a retry with the same ticket is refused at the gate.
		rec = postRestore(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestImportSubduxGateRequiresValidReauthTicketOnConfirm(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("preview does not require reauth", func(t *testing.T) {
		rec := postSubduxImport(t, e, token, false, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})

	t.Run("missing ticket is refused on confirm", func(t *testing.T) {
		rec := postSubduxImport(t, e, token, true, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused on confirm", func(t *testing.T) {
		exportTicket := mintReauthTicket(t, e, token, service.ReauthOperationExportRedacted)
		rec := postSubduxImport(t, e, token, true, exportTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, service.ReauthOperationImportSubdux)

		rec := postSubduxImport(t, e, token, true, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec = postSubduxImport(t, e, token, true, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}
