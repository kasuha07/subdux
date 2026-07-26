package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	"gorm.io/gorm"
)

// newDestinationProbeTestDB is the smallest database the unsaved-config probe
// needs: destination rows plus an admin to authenticate as.
func newDestinationProbeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	t.Setenv("DATA_PATH", t.TempDir())

	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(&model.BackupDestination{}); err != nil {
		t.Fatalf("failed to migrate backup destinations: %v", err)
	}
	return db
}

// postDestinationProbe calls the unsaved-config probe with no reauth ticket,
// which is the whole point of the endpoint: it reads and writes nothing, so the
// admin token alone is enough.
func postDestinationProbe(t *testing.T, e http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/admin/backup/destinations/test", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestTestBackupDestinationConfigProbesUnsavedConfig covers the create flow the
// endpoint exists for: a destination that has never been saved is reachable, the
// response reuses the saved-probe envelope, and nothing is persisted.
func TestTestBackupDestinationConfigProbesUnsavedConfig(t *testing.T) {
	db := newDestinationProbeTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	token := reauthGateTestToken(t, admin)
	e := newHumanOnlyRouteTestServer(t, db)

	dir := t.TempDir()
	archive := filepath.Join(dir, "subdux-backup-20260615-030000-aaa111.zip")
	if err := os.WriteFile(archive, []byte("x"), 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	config, err := json.Marshal(fmt.Sprintf(`{"dir":%q}`, dir))
	if err != nil {
		t.Fatalf("encode config: %v", err)
	}
	rec := postDestinationProbe(t, e, token, fmt.Sprintf(`{"type":"local","config":%s}`, config))

	if rec.Code != http.StatusOK {
		t.Fatalf("probe status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode probe response: %v; body = %s", err, rec.Body.String())
	}
	var messageCode string
	if err := json.Unmarshal(payload["message_code"], &messageCode); err != nil {
		t.Fatalf("decode message_code: %v; body = %s", err, rec.Body.String())
	}
	if messageCode != "backup_destination_reachable" {
		t.Fatalf("message_code = %q, want backup_destination_reachable", messageCode)
	}
	var messageParams map[string]any
	if err := json.Unmarshal(payload["message_params"], &messageParams); err != nil {
		t.Fatalf("decode message_params: %v; body = %s", err, rec.Body.String())
	}
	if got, want := messageParams["backup_count"], float64(1); got != want {
		t.Fatalf("message_params.backup_count = %v, want %v", got, want)
	}

	var stored int64
	if err := db.Model(&model.BackupDestination{}).Count(&stored).Error; err != nil {
		t.Fatalf("count destinations: %v", err)
	}
	if stored != 0 {
		t.Fatalf("probe persisted %d destinations, want 0", stored)
	}
}

// TestTestBackupDestinationConfigRefusesStoredSecretForChangedEndpoint asserts
// the step-up gate on destination writes is not routable around: an unticketed
// probe cannot pair a stored credential with an endpoint it just supplied.
func TestTestBackupDestinationConfigRefusesStoredSecretForChangedEndpoint(t *testing.T) {
	db := newDestinationProbeTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	token := reauthGateTestToken(t, admin)

	destination, err := servicebackup.NewService(db).CreateDestination(servicebackup.CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"stored"}`,
	})
	if err != nil {
		t.Fatalf("create backup destination: %v", err)
	}

	e := newHumanOnlyRouteTestServer(t, db)
	config, err := json.Marshal(`{"endpoint":"s3.attacker.example","bucket":"b","access_key_id":"AKIA","secret_access_key":""}`)
	if err != nil {
		t.Fatalf("encode config: %v", err)
	}
	rec := postDestinationProbe(t, e, token, fmt.Sprintf(
		`{"type":"s3","config":%s,"destination_id":%d}`, config, destination.ID,
	))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("probe status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_secret_required_for_test") {
		t.Fatalf("body = %s, want backup_destination_secret_required_for_test", rec.Body.String())
	}
}

// TestTestBackupDestinationConfigRejectsUnknownDestination confirms the typed
// not-found service error reaches the client through the central mapper rather
// than as a generic 500.
func TestTestBackupDestinationConfigRejectsUnknownDestination(t *testing.T) {
	db := newDestinationProbeTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	token := reauthGateTestToken(t, admin)
	e := newHumanOnlyRouteTestServer(t, db)

	rec := postDestinationProbe(t, e, token, `{"type":"local","config":"{}","destination_id":9999}`)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("probe status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_not_found") {
		t.Fatalf("body = %s, want backup_destination_not_found", rec.Body.String())
	}
}

// TestTestBackupDestinationConfigReportsUnreachableTargetAsBadGateway covers the
// feature's most common failure. A refused connection arrives as an untyped error
// from net/http, so it must not fall through to the mutation fallback and be
// reported as a 500 reading "failed to save backup destination" — nothing is
// being saved, and a wrong endpoint is the admin's to correct.
func TestTestBackupDestinationConfigReportsUnreachableTargetAsBadGateway(t *testing.T) {
	db := newDestinationProbeTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	token := reauthGateTestToken(t, admin)
	e := newHumanOnlyRouteTestServer(t, db)

	// Bind and immediately release a port so the address is well formed but
	// nothing is listening on it.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	closedURL := "http://" + listener.Addr().String() + "/dav"
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}

	config, err := json.Marshal(fmt.Sprintf(`{"url":%q,"username":"u","password":"p"}`, closedURL))
	if err != nil {
		t.Fatalf("encode config: %v", err)
	}
	rec := postDestinationProbe(t, e, token, fmt.Sprintf(`{"type":"webdav","config":%s}`, config))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("probe status = %d, want %d; body = %s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
	if !hasErrorCode(rec.Body.String(), "backup_destination_unreachable") {
		t.Fatalf("body = %s, want backup_destination_unreachable", rec.Body.String())
	}
	if hasErrorCode(rec.Body.String(), "failed_to_save_backup_destination") {
		t.Fatalf("body = %s, a probe failure must not report a save failure", rec.Body.String())
	}
}

// TestSavedDestinationProbeRouteStillResolves guards the router: the new static
// /backup/destinations/test path sits next to the /:id parameter segment, so
// both the saved-destination probe and the id-scoped mutations must keep
// resolving to their own handlers.
func TestSavedDestinationProbeRouteStillResolves(t *testing.T) {
	db := newDestinationProbeTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	token := reauthGateTestToken(t, admin)

	destination, err := servicebackup.NewService(db).CreateDestination(servicebackup.CreateDestinationInput{
		Type:   "local",
		Config: `{}`,
	})
	if err != nil {
		t.Fatalf("create backup destination: %v", err)
	}

	e := newHumanOnlyRouteTestServer(t, db)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/admin/backup/destinations/%d/test", destination.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("saved destination probe status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// The id-scoped mutations must still reach their own handlers; both are
	// reauth-gated, so a missing ticket (rather than a 404/405) proves the route
	// resolved past the router.
	for _, mutation := range []struct {
		method string
		target string
		body   string
	}{
		{http.MethodPut, fmt.Sprintf("/api/admin/backup/destinations/%d", destination.ID), `{"revision":1,"config":"{}"}`},
		{http.MethodDelete, fmt.Sprintf("/api/admin/backup/destinations/%d?revision=1", destination.ID), ""},
	} {
		req = httptest.NewRequest(mutation.method, mutation.target, strings.NewReader(mutation.body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound || rec.Code == http.StatusMethodNotAllowed {
			t.Fatalf("%s %s stopped resolving: status = %d; body = %s",
				mutation.method, mutation.target, rec.Code, rec.Body.String())
		}
	}

	// The static probe route must win over the :id parameter for its own path.
	// "test" is not a numeric id, so if the param route captured it instead the
	// request would be rejected as a malformed id.
	rec = postDestinationProbe(t, e, token, `{"type":"local","config":"{}"}`)
	if hasErrorCode(rec.Body.String(), "invalid_backup_destination_id") {
		t.Fatalf("the :id route captured /destinations/test; body = %s", rec.Body.String())
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("static probe route status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}
