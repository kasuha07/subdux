package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
)

func restoreDestinationBody(archiveName string) string {
	return fmt.Sprintf(`{"archive_name":%q,"password":""}`, archiveName)
}

// TestRestoreBackupDestinationRejectsMalformedID asserts that an unparseable id
// is reported as a client validation error, and that a malformed id must not
// make the admin UI prompt for a step-up it cannot satisfy — the id is parsed
// before the reauth gate.
func TestRestoreBackupDestinationRejectsMalformedID(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	rec := postRestoreBackupDestinationRaw(t, e, token, "not-a-number", restoreDestinationBody("archive.zip"), "")
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

// TestRestoreBackupDestinationRequiresReauthTicket proves the restore endpoint
// refuses a request with no ticket, and that the refusal happens before the
// destination is even looked at.
func TestRestoreBackupDestinationRequiresReauthTicket(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)

	rec := postRestoreBackupDestination(t, e, token, destination.ID, restoreDestinationBody("archive.zip"), "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
		t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
	}

	// Nothing changed: the destination still reports no stored backups, proving
	// the refusal happened before any archive lookup or restore work.
	list := getBackupDestinationBackups(t, e, token, destination.ID)
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d; body = %s", list.Code, http.StatusOK, list.Body.String())
	}
	var body backupDestinationBackupListResponse
	if err := json.Unmarshal(list.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode backups list: %v; body = %s", err, list.Body.String())
	}
	if len(body.Backups) != 0 {
		t.Fatalf("backups = %v, want none", body.Backups)
	}
}

// TestRestoreBackupDestinationRejectsWrongOperationTicket proves a ticket minted
// for a different operation cannot be replayed against restore.
func TestRestoreBackupDestinationRejectsWrongOperationTicket(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)
	wrongTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupRun)

	rec := postRestoreBackupDestination(t, e, token, destination.ID, restoreDestinationBody("archive.zip"), wrongTicket)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
		t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
	}
}

// TestRestoreBackupDestinationRejectsReplayedTicket proves a ticket is single-use
// even when the request it first authorized fails downstream (here, an unknown
// destination) — the ticket is spent the moment the gate accepts it, before the
// service is ever called.
func TestRestoreBackupDestinationRejectsReplayedTicket(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationRestore)

	rec := postRestoreBackupDestination(t, e, token, 9999, restoreDestinationBody("archive.zip"), ticket)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("first use status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_not_found") {
		t.Fatalf("first use body = %s, want backup_destination_not_found", rec.Body.String())
	}

	rec = postRestoreBackupDestination(t, e, token, 9999, restoreDestinationBody("archive.zip"), ticket)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("replayed ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
		t.Fatalf("replayed ticket body = %s, want re-authentication required", rec.Body.String())
	}
}

// TestRestoreBackupDestinationRejectsUnknownDestination asserts a well-formed id
// with no matching row surfaces the typed not-found service error.
func TestRestoreBackupDestinationRejectsUnknownDestination(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationRestore)

	rec := postRestoreBackupDestination(t, e, token, 9999, restoreDestinationBody("archive.zip"), ticket)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_not_found") {
		t.Fatalf("body = %s, want backup_destination_not_found", rec.Body.String())
	}
}

// TestRestoreBackupDestinationRejectsUnknownArchive asserts a real destination
// with an archive name outside its own listing is refused before any download
// or restore is attempted.
func TestRestoreBackupDestinationRejectsUnknownArchive(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)
	ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationRestore)

	rec := postRestoreBackupDestination(t, e, token, destination.ID, restoreDestinationBody("subdux-backup-20260615-030000-bogus.zip"), ticket)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_archive_not_found") {
		t.Fatalf("body = %s, want backup_archive_not_found", rec.Body.String())
	}
}

// TestRestoreBackupDestinationRejectsOversizedArchive asserts the listing
// precheck refuses an archive the destination itself reports as over the
// 32 MiB restore cap before any download is attempted, and that the reported
// max_mb matches the cap the handler actually enforces.
func TestRestoreBackupDestinationRejectsOversizedArchive(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)

	// createLocalRunTestDestination saves an empty config, which resolves to
	// <DATA_PATH>/backups; newBackupRunTestDB has already pointed DATA_PATH at
	// a temp dir via t.Setenv, so it is still readable here.
	destDir := filepath.Join(os.Getenv("DATA_PATH"), "backups")
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		t.Fatalf("mkdir destination dir: %v", err)
	}

	const archiveName = "subdux-backup-20260726-030000-oversized0.zip"
	// Sparse file: only the reported size matters to the precheck, so this
	// costs no real disk space.
	f, err := os.Create(filepath.Join(destDir, archiveName))
	if err != nil {
		t.Fatalf("create sparse archive: %v", err)
	}
	if err := f.Truncate(33 << 20); err != nil {
		_ = f.Close()
		t.Fatalf("truncate sparse archive: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close sparse archive: %v", err)
	}

	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)

	ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationRestore)

	rec := postRestoreBackupDestination(t, e, token, destination.ID, restoreDestinationBody(archiveName), ticket)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_file_is_too_large_max_mb") {
		t.Fatalf("body = %s, want backup_file_is_too_large_max_mb", rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body: %v; body = %s", err, rec.Body.String())
	}
	params, ok := body["error_params"].(map[string]any)
	if !ok {
		t.Fatalf("error_params missing or malformed; body = %s", rec.Body.String())
	}
	// JSON numbers decode into float64 via map[string]any.
	if maxMB, ok := params["max_mb"].(float64); !ok || maxMB != 32 {
		t.Fatalf("error_params.max_mb = %v, want 32; body = %s", params["max_mb"], rec.Body.String())
	}

	// The listing precheck (restore_destination.go: listed.Size > cap) must
	// refuse before any transfer, so nothing is ever staged under TMPDIR.
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir(TMPDIR) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("TMPDIR entries = %v, want none (precheck must refuse before any transfer)", entries)
	}
}

// TestListBackupDestinationBackupsNeedsNoReauthTicket confirms the listing
// endpoint is reachable without a step-up ticket, matching the connectivity
// probe it sits beside.
func TestListBackupDestinationBackupsNeedsNoReauthTicket(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	destination := createLocalRunTestDestination(t, db)

	rec := getBackupDestinationBackups(t, e, token, destination.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body backupDestinationBackupListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode backups list: %v; body = %s", err, rec.Body.String())
	}
	if len(body.Backups) != 0 {
		t.Fatalf("backups = %v, want none", body.Backups)
	}
}

func TestListBackupDestinationBackupsRejectsMalformedID(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	rec := getBackupDestinationBackupsRaw(t, e, token, "not-a-number")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "invalid_backup_destination_id") {
		t.Fatalf("body = %s, want invalid_backup_destination_id", rec.Body.String())
	}
}

func TestListBackupDestinationBackupsRejectsUnknownDestination(t *testing.T) {
	db := newBackupRunTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	rec := getBackupDestinationBackups(t, e, token, 9999)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_not_found") {
		t.Fatalf("body = %s, want backup_destination_not_found", rec.Body.String())
	}
}
