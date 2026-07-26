package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"gorm.io/gorm"
)

// backupCreatedResponseFields is the body the retired global run-now route
// returned. The per-destination endpoint reproduces it verbatim so a client
// that already renders a run summary keeps working across the move.
var backupCreatedResponseFields = []string{
	"file",
	"run_id",
	"status",
	"delivery_status",
	"retention_status",
	"bookkeeping_status",
	"global_bookkeeping_status",
	"error",
	"results",
}

// newBackupRunTestDB extends the human-only route database with the durable
// run tables and points DATA_PATH at a temp directory, so a manual run can
// build a real archive and deliver it to a local destination without writing
// outside the test's own sandbox.
func newBackupRunTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	t.Setenv("DATA_PATH", t.TempDir())

	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(
		&model.BackupDestination{},
		&model.BackupRun{},
		&model.BackupRunDestination{},
	); err != nil {
		t.Fatalf("failed to migrate backup run tables: %v", err)
	}
	return db
}

// createLocalRunTestDestination creates an enabled local destination with the
// default plan. An empty config means the default <DATA_PATH>/backups
// directory, which newBackupRunTestDB has already redirected into a temp dir.
func createLocalRunTestDestination(t *testing.T, db *gorm.DB) *servicebackup.DestinationView {
	t.Helper()

	destination, err := servicebackup.NewService(db).CreateDestination(servicebackup.CreateDestinationInput{
		Type:    "local",
		Enabled: true,
		Config:  `{}`,
	})
	if err != nil {
		t.Fatalf("create local backup destination: %v", err)
	}
	return destination
}

// assertNoBackupRunPerformed proves a rejected request stopped before any
// backup work: no run row was opened and the destination recorded no delivery.
func assertNoBackupRunPerformed(t *testing.T, db *gorm.DB, destinationID uint) {
	t.Helper()

	var runs int64
	if err := db.Model(&model.BackupRun{}).Count(&runs).Error; err != nil {
		t.Fatalf("count backup runs: %v", err)
	}
	if runs != 0 {
		t.Fatalf("backup run count = %d after a rejected request, want 0", runs)
	}

	var stored model.BackupDestination
	if err := db.First(&stored, destinationID).Error; err != nil {
		t.Fatalf("reload backup destination: %v", err)
	}
	if stored.LastRunAt != nil || stored.LastStatus != "" {
		t.Fatalf(
			"destination recorded a run after a rejected request: last_run_at = %v, last_status = %q",
			stored.LastRunAt, stored.LastStatus,
		)
	}
}

// assertBackupCreatedBody locks the success envelope of a manual destination
// run: the backup_created message code plus every field the old global route
// published, with a real archive name and a successful status.
func assertBackupCreatedBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode run response: %v; body = %s", err, rec.Body.String())
	}

	var messageCode string
	if err := json.Unmarshal(payload["message_code"], &messageCode); err != nil {
		t.Fatalf("decode message_code: %v; body = %s", err, rec.Body.String())
	}
	if messageCode != "backup_created" {
		t.Fatalf("message_code = %q, want backup_created", messageCode)
	}

	for _, field := range backupCreatedResponseFields {
		if _, ok := payload[field]; !ok {
			t.Fatalf("run response is missing %q; body = %s", field, rec.Body.String())
		}
	}

	var file string
	if err := json.Unmarshal(payload["file"], &file); err != nil {
		t.Fatalf("decode file: %v; body = %s", err, rec.Body.String())
	}
	if file == "" {
		t.Fatalf("file = %q, want the generated archive name; body = %s", file, rec.Body.String())
	}

	var status string
	if err := json.Unmarshal(payload["status"], &status); err != nil {
		t.Fatalf("decode status: %v; body = %s", err, rec.Body.String())
	}
	if status != servicebackup.StatusOK {
		t.Fatalf("status = %q, want %q; body = %s", status, servicebackup.StatusOK, rec.Body.String())
	}
}

// TestRunBackupDestinationRejectsMalformedID asserts that an unparseable id is
// reported as a client validation error. The id is parsed before the reauth
// gate, so a typo in the path must not make the admin UI prompt for a step-up
// it cannot satisfy.
func TestRunBackupDestinationRejectsMalformedID(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	rec := postRunBackupDestinationRaw(t, e, token, "not-a-number", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "invalid_backup_destination_id") {
		t.Fatalf("body = %s, want invalid_backup_destination_id", rec.Body.String())
	}
	if hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
		t.Fatalf("body = %s, a malformed id must not prompt for re-authentication", rec.Body.String())
	}
}

// TestRunBackupDestinationRejectsUnknownID asserts that a well-formed id with
// no matching row surfaces the typed not-found service error through the
// central mapper rather than a generic 500.
func TestRunBackupDestinationRejectsUnknownID(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	// The gate sits ahead of the lookup, so reaching the not-found path at all
	// requires a valid ticket.
	ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupRun)

	rec := postRunBackupDestination(t, e, token, 9999, ticket)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_not_found") {
		t.Fatalf("body = %s, want backup_destination_not_found", rec.Body.String())
	}
}

// TestRetiredGlobalBackupRunRouteIsGone asserts the old global run-now endpoint
// is no longer routable. Leaving it registered would keep a second, unscoped
// way to fire backups alive next to the per-destination endpoint.
func TestRetiredGlobalBackupRunRouteIsGone(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/backup/run", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Either verdict proves the handler is unreachable; which one the router
	// picks depends on how it resolves the sibling /backup/* paths.
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d or %d; body = %s",
			rec.Code, http.StatusNotFound, http.StatusMethodNotAllowed, rec.Body.String())
	}
}
