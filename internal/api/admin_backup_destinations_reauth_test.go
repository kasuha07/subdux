package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func issueDestinationReauthTicket(t *testing.T, e *echo.Echo, token, operation string, destinationID uint, revision uint64) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(
		`{"operation":%q,"destination_id":%d,"destination_revision":%d,"password":%q,"code":""}`,
		operation,
		destinationID,
		revision,
		reauthGateTestPassword,
	)
	req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func destinationReauthTicket(t *testing.T, e *echo.Echo, token, operation string, destinationID uint, revision uint64) string {
	t.Helper()
	rec := issueDestinationReauthTicket(t, e, token, operation, destinationID, revision)
	if rec.Code != http.StatusOK {
		t.Fatalf("mint destination ticket status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode destination ticket: %v", err)
	}
	if response.Ticket == "" {
		t.Fatalf("mint destination ticket returned empty ticket; body = %s", rec.Body.String())
	}
	return response.Ticket
}

func putBackupDestination(t *testing.T, e *echo.Echo, token string, id uint, revision uint64, ticket string) *httptest.ResponseRecorder {
	t.Helper()
	body := fmt.Sprintf(`{"revision":%d,"enabled":false}`, revision)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/backup/destinations/%d", id), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(apimw.ReauthTicketHeader, ticket)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func deleteBackupDestination(t *testing.T, e *echo.Echo, token string, id uint, revision uint64, ticket string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/backup/destinations/%d?revision=%d", id, revision), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(apimw.ReauthTicketHeader, ticket)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestBackupDestinationReauthScopesIntentAndRevision(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(&model.BackupDestination{}); err != nil {
		t.Fatalf("failed to migrate backup destinations: %v", err)
	}
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	backupService := servicebackup.NewService(db)
	first, err := backupService.CreateDestination(servicebackup.CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("create first destination: %v", err)
	}
	second, err := backupService.CreateDestination(servicebackup.CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("create second destination: %v", err)
	}

	t.Run("create uses its own operation", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationCreate)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/backup/destinations", strings.NewReader(`{"type":"local","enabled":false,"config":"{}","sort_order":2}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
		}

		wrongIntentTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationCreate)
		if rec := putBackupDestination(t, e, token, first.ID, first.Revision, wrongIntentTicket); rec.Code != http.StatusBadRequest {
			t.Fatalf("create ticket on update status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("update requires a bound target", func(t *testing.T) {
		rec := issueDestinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, 0, 0)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("unbound update ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("update binding rejects another destination and stale revision", func(t *testing.T) {
		ticket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, first.ID, first.Revision)
		if rec := putBackupDestination(t, e, token, second.ID, second.Revision, ticket); rec.Code != http.StatusBadRequest {
			t.Fatalf("cross-destination update status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if rec := putBackupDestination(t, e, token, first.ID, first.Revision, ticket); rec.Code != http.StatusBadRequest {
			t.Fatalf("reused mismatched update ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}

		staleTicket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, first.ID, first.Revision)
		enabled := false
		if _, err := backupService.UpdateDestination(first.ID, servicebackup.UpdateDestinationInput{Revision: first.Revision, Enabled: &enabled}); err != nil {
			t.Fatalf("advance destination revision: %v", err)
		}
		if rec := putBackupDestination(t, e, token, first.ID, first.Revision, staleTicket); rec.Code != http.StatusConflict {
			t.Fatalf("stale update status = %d, want %d; body = %s", rec.Code, http.StatusConflict, rec.Body.String())
		}
	})

	var current model.BackupDestination
	if err := db.First(&current, first.ID).Error; err != nil {
		t.Fatalf("reload first destination: %v", err)
	}
	t.Run("valid update advances revision", func(t *testing.T) {
		ticket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, current.ID, current.Revision)
		rec := putBackupDestination(t, e, token, current.ID, current.Revision, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("valid update status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
	if err := db.First(&current, first.ID).Error; err != nil {
		t.Fatalf("reload first destination after update: %v", err)
	}

	// The password reauth endpoint intentionally rate-limits each account. Use
	// a fresh router for the delete-intent assertions so this test does not
	// depend on the limiter window after exercising the create/update paths.
	deleteServer := newHumanOnlyRouteTestServer(t, db)
	t.Run("delete has a distinct bound operation", func(t *testing.T) {
		wrongIntentTicket := destinationReauthTicket(t, deleteServer, token, servicereauth.ReauthOperationBackupDestinationUpdate, current.ID, current.Revision)
		if rec := deleteBackupDestination(t, deleteServer, token, current.ID, current.Revision, wrongIntentTicket); rec.Code != http.StatusBadRequest {
			t.Fatalf("update ticket on delete status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}

		ticket := destinationReauthTicket(t, deleteServer, token, servicereauth.ReauthOperationBackupDestinationDelete, current.ID, current.Revision)
		if rec := deleteBackupDestination(t, deleteServer, token, second.ID, second.Revision, ticket); rec.Code != http.StatusBadRequest {
			t.Fatalf("cross-destination delete status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if rec := deleteBackupDestination(t, deleteServer, token, current.ID, current.Revision, ticket); rec.Code != http.StatusBadRequest {
			t.Fatalf("reused mismatched delete ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}

		ticket = destinationReauthTicket(t, deleteServer, token, servicereauth.ReauthOperationBackupDestinationDelete, current.ID, current.Revision)
		rec := deleteBackupDestination(t, deleteServer, token, current.ID, current.Revision, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("valid delete status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}

// TestDeleteBackupDestinationRequiresRevision asserts that a missing
// ?revision= query parameter is reported as a client validation error, not as
// a re-authentication prompt: the request never gets far enough to consume a
// reauth ticket, so it must fail with invalid_backup_destination_revision
// rather than re_authentication_required.
func TestDeleteBackupDestinationRequiresRevision(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(&model.BackupDestination{}); err != nil {
		t.Fatalf("failed to migrate backup destinations: %v", err)
	}
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	backupService := servicebackup.NewService(db)
	destination, err := backupService.CreateDestination(servicebackup.CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("create destination: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/backup/destinations/%d", destination.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code < 400 || rec.Code >= 500 {
		t.Fatalf("missing revision delete status = %d, want a 4xx client error; body = %s", rec.Code, rec.Body.String())
	}

	var response struct {
		ErrorCode string `json:"error_code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.ErrorCode != "invalid_backup_destination_revision" {
		t.Fatalf("error_code = %q, want %q; body = %s", response.ErrorCode, "invalid_backup_destination_revision", rec.Body.String())
	}
}
