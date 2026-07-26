package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// The backup/restore step-up gates live in AdminHandler.BackupDB,
// AdminHandler.RunBackupDestination, and AdminHandler.RestoreDB, which call
// ReauthService.Consume before doing any work. The service-level Consume tests
// prove ticket semantics in isolation; these tests prove the wiring end-to-end
// through the router — that the sensitive endpoints actually refuse a request
// with no ticket, a wrong-operation ticket, or an already-spent ticket, and only
// proceed once a valid, operation-matched ticket is presented.
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
	return mintReauthTicketWithCode(t, e, token, operation, "")
}

func mintReauthTicketWithCode(t *testing.T, e *echo.Echo, token, operation, code string) string {
	t.Helper()

	body := fmt.Sprintf(`{"operation":%q,"password":%q,"code":%q}`, operation, reauthGateTestPassword, code)
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

func enableReauthGateTestTOTP(t *testing.T, db *gorm.DB, userID uint, secret string) {
	t.Helper()
	if err := db.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
		"totp_enabled": true,
		"totp_secret":  secret,
	}).Error; err != nil {
		t.Fatalf("failed to enable totp: %v", err)
	}
}

func postBackup(t *testing.T, e *echo.Echo, token, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := `{"include_assets":false,"password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/backup", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// postRunBackupDestinationRaw fires the per-destination run endpoint with the
// path segment verbatim, so callers can also exercise an unparseable id.
func postRunBackupDestinationRaw(t *testing.T, e *echo.Echo, token, rawID, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/backup/destinations/%s/run", rawID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postRunBackupDestination(t *testing.T, e *echo.Echo, token string, id uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()
	return postRunBackupDestinationRaw(t, e, token, fmt.Sprintf("%d", id), ticket)
}

func putAdminSettings(t *testing.T, e *echo.Echo, token, body, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPut, "/api/admin/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func getAdminSettings(t *testing.T, e *echo.Echo, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
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
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
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
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
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
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
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
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func deleteAPIKey(t *testing.T, e *echo.Echo, token string, id uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/api-keys/%d", id), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postAdminUser(t *testing.T, e *echo.Echo, token, username, email, role, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"username":%q,"email":%q,"password":%q,"role":%q}`, username, email, "new-user-passphrase", role)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/users", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func putAdminUserRole(t *testing.T, e *echo.Echo, token string, userID uint, role, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"role":%q}`, role)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/role", userID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func deleteAdminUser(t *testing.T, e *echo.Echo, token string, userID uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/users/%d", userID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postAdminDisableUserTOTP(t *testing.T, e *echo.Echo, token string, userID uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/users/%d/disable-totp", userID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postAdminDisableUserPasskeys(t *testing.T, e *echo.Echo, token string, userID uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/users/%d/disable-passkeys", userID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		restoreTicket := mintReauthTicket(t, e, token, "restore")
		rec := postBackup(t, e, token, restoreTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

// TestRunBackupDestinationGateRequiresValidReauthTicket proves the step-up gate
// that used to guard the global run-now route now guards the per-destination
// run endpoint. A run ships a full database snapshot to a configured target, so
// it stays proven-present-admin work even though it changes no configuration.
func TestRunBackupDestinationGateRequiresValidReauthTicket(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)

	t.Run("missing ticket is refused and performs no run", func(t *testing.T) {
		rec := postRunBackupDestination(t, e, token, destination.ID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}

		// The gate runs before any backup work, so a refused request must leave
		// no run row behind and no delivery recorded on the destination. Status
		// alone would not catch a gate that refuses only after firing the run.
		assertNoBackupRunPerformed(t, db, destination.ID)
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postRunBackupDestination(t, e, token, destination.ID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		assertNoBackupRunPerformed(t, db, destination.ID)
	})

	t.Run("a destination-bound ticket is refused for this unbound operation", func(t *testing.T) {
		// Unlike create/update/delete, a run alters no configuration, so there is
		// no revision for the admin to confirm and backup_run is deliberately
		// unbound. The refusal lands at the mint step: a client cannot obtain a
		// destination-scoped backup_run ticket to present here in the first place.
		rec := issueDestinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupRun, destination.ID, destination.Revision)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("bound backup_run mint status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), `"ticket"`) {
			t.Fatalf("body = %s, a bound backup_run ticket must not be issued", rec.Body.String())
		}
	})

	t.Run("valid ticket runs the destination and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupRun)

		rec := postRunBackupDestination(t, e, token, destination.ID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		assertBackupCreatedBody(t, rec)

		// The same ticket must not authorize a second run.
		rec = postRunBackupDestination(t, e, token, destination.ID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestChangeUserRoleRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	target := model.User{
		Username: "role-target",
		Email:    "role-target@example.com",
		Password: "x",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := putAdminUserRole(t, e, token, target.ID, "admin", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := putAdminUserRole(t, e, token, target.ID, "admin", backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationChangeUserRole)

		rec := putAdminUserRole(t, e, token, target.ID, "admin", ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var updated model.User
		if err := db.Select("id", "role").First(&updated, target.ID).Error; err != nil {
			t.Fatalf("failed to reload target user: %v", err)
		}
		if updated.Role != "admin" {
			t.Fatalf("target role = %q, want admin", updated.Role)
		}

		rec = putAdminUserRole(t, e, token, target.ID, "user", ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestCreateAdminUserRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("regular user creation does not require ticket", func(t *testing.T) {
		rec := postAdminUser(t, e, token, "create-regular", "create-regular@example.com", "user", "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
	})

	t.Run("missing ticket is refused for admin role", func(t *testing.T) {
		rec := postAdminUser(t, e, token, "create-admin-missing", "create-admin-missing@example.com", "admin", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postAdminUser(t, e, token, "create-admin-wrong", "create-admin-wrong@example.com", "admin", backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationCreateAdminUser)

		rec := postAdminUser(t, e, token, "create-admin-valid", "create-admin-valid@example.com", "admin", ticket)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}
		var created model.User
		if err := db.Select("id", "role").Where("email = ?", "create-admin-valid@example.com").First(&created).Error; err != nil {
			t.Fatalf("failed to reload created user: %v", err)
		}
		if created.Role != "admin" {
			t.Fatalf("created role = %q, want admin", created.Role)
		}

		rec = postAdminUser(t, e, token, "create-admin-reuse", "create-admin-reuse@example.com", "admin", ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestDeleteUserRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	target := model.User{
		Username: "delete-target",
		Email:    "delete-target@example.com",
		Password: "x",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := deleteAdminUser(t, e, token, target.ID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := deleteAdminUser(t, e, token, target.ID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationDeleteUser)

		rec := deleteAdminUser(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var deleted model.User
		if err := db.First(&deleted, target.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("deleted user lookup error = %v, want %v", err, gorm.ErrRecordNotFound)
		}

		rec = deleteAdminUser(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestAdminDisableUserTOTPRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	totpSecret := "JBSWY3DPEHPK3PXP"
	target := model.User{
		Username:    "disable-totp-target",
		Email:       "disable-totp-target@example.com",
		Password:    "x",
		Role:        "user",
		Status:      "active",
		TotpEnabled: true,
		TotpSecret:  &totpSecret,
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}
	backupCode := model.UserBackupCode{UserID: target.ID, CodeHash: "backup-hash"}
	if err := db.Create(&backupCode).Error; err != nil {
		t.Fatalf("failed to create target backup code: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	assertTOTPStillEnabled := func(t *testing.T) {
		t.Helper()
		var stored model.User
		if err := db.Select("id", "totp_enabled", "totp_secret").First(&stored, target.ID).Error; err != nil {
			t.Fatalf("failed to reload target user: %v", err)
		}
		if !stored.TotpEnabled || stored.TotpSecret == nil {
			t.Fatalf("target TOTP state = enabled:%t secret:%v, want unchanged", stored.TotpEnabled, stored.TotpSecret)
		}
		var count int64
		if err := db.Model(&model.UserBackupCode{}).Where("user_id = ?", target.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count backup codes: %v", err)
		}
		if count != 1 {
			t.Fatalf("target backup code count = %d, want 1", count)
		}
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postAdminDisableUserTOTP(t, e, token, target.ID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		assertTOTPStillEnabled(t)
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postAdminDisableUserTOTP(t, e, token, target.ID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		assertTOTPStillEnabled(t)
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationAdminDisableTOTP)

		rec := postAdminDisableUserTOTP(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var updated model.User
		if err := db.Select("id", "totp_enabled", "totp_secret").First(&updated, target.ID).Error; err != nil {
			t.Fatalf("failed to reload target user: %v", err)
		}
		if updated.TotpEnabled || updated.TotpSecret != nil {
			t.Fatalf("target TOTP state = enabled:%t secret:%v, want disabled and cleared", updated.TotpEnabled, updated.TotpSecret)
		}

		rec = postAdminDisableUserTOTP(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestAdminDisableUserPasskeysRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	target := model.User{
		Username: "disable-passkeys-target",
		Email:    "disable-passkeys-target@example.com",
		Password: "x",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}
	passkey := model.PasskeyCredential{
		UserID:       target.ID,
		Name:         "Laptop",
		CredentialID: "cred-admin-disable-passkeys",
		Credential:   []byte("credential"),
	}
	if err := db.Create(&passkey).Error; err != nil {
		t.Fatalf("failed to create target passkey: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	assertPasskeysStillPresent := func(t *testing.T) {
		t.Helper()
		var count int64
		if err := db.Model(&model.PasskeyCredential{}).Where("user_id = ?", target.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count passkeys: %v", err)
		}
		if count != 1 {
			t.Fatalf("target passkey count = %d, want 1", count)
		}
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := postAdminDisableUserPasskeys(t, e, token, target.ID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		assertPasskeysStillPresent(t)
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postAdminDisableUserPasskeys(t, e, token, target.ID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
		assertPasskeysStillPresent(t)
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationAdminDisablePasskeys)

		rec := postAdminDisableUserPasskeys(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var count int64
		if err := db.Model(&model.PasskeyCredential{}).Where("user_id = ?", target.ID).Count(&count).Error; err != nil {
			t.Fatalf("failed to count passkeys: %v", err)
		}
		if count != 0 {
			t.Fatalf("target passkey count = %d, want 0", count)
		}

		rec = postAdminDisableUserPasskeys(t, e, token, target.ID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := postCreateAPIKey(t, e, token, "Script", backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationCreateAPIKey)

		rec := postCreateAPIKey(t, e, token, "Script", ticket)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		rec = postCreateAPIKey(t, e, token, "Second Script", ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

func TestReauthPasswordRequiresTOTPWhenEnabled(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	enableReauthGateTestTOTP(t, db, admin.ID, "JBSWY3DPEHPK3PXP")
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("methods advertise the upgraded password factor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/reauth/methods?operation=backup", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp struct {
			Password             bool `json:"password"`
			PasswordRequiresTOTP bool `json:"password_requires_totp"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode methods response: %v; body = %s", err, rec.Body.String())
		}
		if !resp.Password {
			t.Fatal("methods.password = false, want true")
		}
		if !resp.PasswordRequiresTOTP {
			t.Fatal("methods.password_requires_totp = false, want true")
		}
	})

	t.Run("missing or invalid totp code cannot mint a ticket", func(t *testing.T) {
		body := fmt.Sprintf(`{"operation":%q,"password":%q}`, servicereauth.ReauthOperationBackup, reauthGateTestPassword)
		req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("missing code status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("missing code body = %s, want re-authentication required", rec.Body.String())
		}

		body = fmt.Sprintf(`{"operation":%q,"password":%q,"code":"000000"}`, servicereauth.ReauthOperationBackup, reauthGateTestPassword)
		req = httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("invalid code status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("invalid code body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("current totp code still allows the sensitive operation", func(t *testing.T) {
		code, err := totp.GenerateCode("JBSWY3DPEHPK3PXP", time.Now().UTC())
		if err != nil {
			t.Fatalf("GenerateCode() error = %v, want nil", err)
		}

		ticket := mintReauthTicketWithCode(t, e, token, servicereauth.ReauthOperationBackup, code)
		rec := postBackup(t, e, token, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("backup status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}

func TestReauthPasswordDisabledForPasskeyAccount(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	if err := db.Create(&model.PasskeyCredential{
		UserID:       admin.ID,
		Name:         "laptop",
		CredentialID: "cred-gate-passkey-only",
		Credential:   []byte("{}"),
	}).Error; err != nil {
		t.Fatalf("failed to seed passkey: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("methods withhold the password factor", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/reauth/methods?operation=backup", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var resp struct {
			Password bool `json:"password"`
			Passkey  bool `json:"passkey"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode methods response: %v; body = %s", err, rec.Body.String())
		}
		if resp.Password {
			t.Fatal("methods.password = true for passkey-only account, want false")
		}
		if !resp.Passkey {
			t.Fatal("methods.passkey = false, want true")
		}
	})

	t.Run("password step-up is refused", func(t *testing.T) {
		body := fmt.Sprintf(`{"operation":%q,"password":%q}`, servicereauth.ReauthOperationBackup, reauthGateTestPassword)
		req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "password re-authentication is not available for this account; use a passkey") {
			t.Fatalf("body = %s, want a passkey-directed message", rec.Body.String())
		}
	})
}

func TestAPIKeyListDoesNotRequireReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	_, err := apikeyservice.NewService(db).Create(admin.ID, admin.Role, apikeyservice.CreateInput{
		Name:    "Integration",
		KeyKind: apikeyservice.APIKeyKindAPIIntegration,
		Scopes:  []string{apikeyservice.APIKeyScopeRead},
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
}

func TestDeleteAPIKeyRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	createAPIKey := func(t *testing.T, name string) uint {
		t.Helper()
		apiKeyResp, err := apikeyservice.NewService(db).Create(admin.ID, admin.Role, apikeyservice.CreateInput{
			Name:    name,
			KeyKind: apikeyservice.APIKeyKindAPIIntegration,
			Scopes:  []string{apikeyservice.APIKeyScopeRead},
		})
		if err != nil {
			t.Fatalf("failed to create api key: %v", err)
		}
		return apiKeyResp.APIKey.ID
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		keyID := createAPIKey(t, "Missing ticket")
		rec := deleteAPIKey(t, e, token, keyID, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		keyID := createAPIKey(t, "Wrong operation")
		backupTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackup)
		rec := deleteAPIKey(t, e, token, keyID, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		keyID := createAPIKey(t, "Delete me")
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationDeleteAPIKey)

		rec := deleteAPIKey(t, e, token, keyID, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var deleted model.APIKey
		if err := db.First(&deleted, keyID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("deleted api key lookup error = %v, want %v", err, gorm.ErrRecordNotFound)
		}

		secondKeyID := createAPIKey(t, "Second delete")
		rec = deleteAPIKey(t, e, token, secondKeyID, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}

// TestUpdateSettingsNoLongerRequiresReauthTicket is the successor of the retired
// backup-schedule gate. Admin settings used to demand a backup_schedule ticket
// because they carried the global scheduled-backup configuration; that
// configuration now lives on individual destinations, so no admin setting is a
// step-up operation any more. The protection it provided did not disappear — it
// moved to the destination endpoints, which
// TestRunBackupDestinationGateRequiresValidReauthTicket and
// TestBackupDestinationReauthScopesIntentAndRevision cover.
//
// The retired operation itself must be rejected end to end: leaving it mintable
// would hand clients a ticket that unlocks nothing while looking like a
// successful step-up.
func TestUpdateSettingsNoLongerRequiresReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	t.Run("settings update succeeds without a ticket", func(t *testing.T) {
		rec := putAdminSettings(t, e, token, `{"site_name":"Subdux Test"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		// The write really landed: a gate that silently swallowed the update
		// would also return 200.
		settings, err := adminservice.NewService(db).GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		if settings.SiteName != "Subdux Test" {
			t.Fatalf("SiteName = %q, want %q", settings.SiteName, "Subdux Test")
		}
	})

	t.Run("the retired backup_schedule operation cannot mint a ticket", func(t *testing.T) {
		body := fmt.Sprintf(`{"operation":"backup_schedule","password":%q,"code":""}`, reauthGateTestPassword)
		req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "invalid reauth operation") {
			t.Fatalf("body = %s, want invalid reauth operation", rec.Body.String())
		}
		// The correct password must not earn a ticket for an unknown operation.
		if strings.Contains(rec.Body.String(), `"ticket"`) {
			t.Fatalf("body = %s, want no ticket for a retired operation", rec.Body.String())
		}
	})

	t.Run("the retired backup_schedule operation offers no reauth methods", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/reauth/methods?operation=backup_schedule", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "invalid reauth operation") {
			t.Fatalf("body = %s, want invalid reauth operation", rec.Body.String())
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused on confirm", func(t *testing.T) {
		subduxTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationImportSubdux)
		rec := postWallosImport(t, e, token, true, subduxTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationImportWallos)

		rec := postWallosImport(t, e, token, true, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec = postWallosImport(t, e, token, true, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("api key can preview but cannot confirm", func(t *testing.T) {
		apiKeyResp, err := apikeyservice.NewService(db).Create(admin.ID, admin.Role, apikeyservice.CreateInput{
			Name:    "Integration",
			KeyKind: apikeyservice.APIKeyKindAPIIntegration,
			Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
		})
		if err != nil {
			t.Fatalf("failed to create api key: %v", err)
		}

		rec := postWallosImportWithAPIKey(t, e, apiKeyResp.Key, false, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("preview status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationImportWallos)
		rec = postWallosImportWithAPIKey(t, e, apiKeyResp.Key, true, ticket)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("confirm status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "human session required") {
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		backupTicket := mintReauthTicket(t, e, token, "backup")
		rec := postRestore(t, e, token, backupTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
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
		if hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, ticket should have cleared the gate", rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "no file uploaded") {
			t.Fatalf("body = %s, want no file uploaded", rec.Body.String())
		}

		// The gate consumed the ticket even though the request failed after it,
		// so a retry with the same ticket is refused at the gate.
		rec = postRestore(t, e, token, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
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
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("wrong-operation ticket is refused on confirm", func(t *testing.T) {
		exportTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationExportRedacted)
		rec := postSubduxImport(t, e, token, true, exportTicket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("valid ticket is accepted and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationImportSubdux)

		rec := postSubduxImport(t, e, token, true, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		rec = postSubduxImport(t, e, token, true, ticket)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})
}
