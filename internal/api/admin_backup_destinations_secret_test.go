package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

// destinationSecretResponse is the slice of the destination DTO these tests
// care about. Config arrives as a JSON string because it is one opaque blob to
// the transport layer, so it needs a second decode.
type destinationSecretResponse struct {
	ID                     uint     `json:"id"`
	Revision               uint64   `json:"revision"`
	Config                 string   `json:"config"`
	ConfiguredSecretFields []string `json:"configured_secret_fields"`
}

func decodeDestinationSecretResponse(t *testing.T, rec *httptest.ResponseRecorder) (destinationSecretResponse, map[string]any) {
	t.Helper()

	var response destinationSecretResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode destination response: %v; body = %s", err, rec.Body.String())
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(response.Config), &config); err != nil {
		t.Fatalf("decode destination config %q: %v", response.Config, err)
	}
	return response, config
}

func postCreateBackupDestination(t *testing.T, e *echo.Echo, token, destinationType, config, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	encodedConfig, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode create config: %v", err)
	}
	body := fmt.Sprintf(`{"type":%q,"enabled":true,"config":%s,"sort_order":0}`, destinationType, encodedConfig)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/backup/destinations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(apimw.ReauthTicketHeader, ticket)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// putBackupDestinationConfig updates a destination's config blob, optionally
// naming secret fields to wipe. It is the config-carrying counterpart of
// putBackupDestination, which only flips the enabled flag.
func putBackupDestinationConfig(
	t *testing.T,
	e *echo.Echo,
	token string,
	id uint,
	revision uint64,
	config string,
	clearedSecretFields []string,
	ticket string,
) *httptest.ResponseRecorder {
	t.Helper()

	encodedConfig, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode update config: %v", err)
	}
	encodedCleared, err := json.Marshal(clearedSecretFields)
	if err != nil {
		t.Fatalf("encode cleared secret fields: %v", err)
	}
	body := fmt.Sprintf(
		`{"revision":%d,"config":%s,"cleared_secret_fields":%s}`,
		revision, encodedConfig, encodedCleared,
	)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/backup/destinations/%d", id), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(apimw.ReauthTicketHeader, ticket)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// assertEncryptionPasswordMasked checks that the archive password never travels
// back to the client, whatever its stored value: the response always carries an
// empty string, and only configured_secret_fields reveals whether one is set.
func assertEncryptionPasswordMasked(t *testing.T, rec *httptest.ResponseRecorder, wantConfigured bool) destinationSecretResponse {
	t.Helper()

	response, config := decodeDestinationSecretResponse(t, rec)

	value, ok := config["encryption_password"]
	if !ok {
		t.Fatalf("config is missing encryption_password; config = %s", response.Config)
	}
	if value != "" {
		t.Fatalf("encryption_password = %v, want the masked empty string; config = %s", value, response.Config)
	}

	gotConfigured := slices.Contains(response.ConfiguredSecretFields, "encryption_password")
	if gotConfigured != wantConfigured {
		t.Fatalf(
			"configured_secret_fields = %v, want encryption_password present = %t",
			response.ConfiguredSecretFields, wantConfigured,
		)
	}
	return response
}

// TestBackupDestinationEncryptionPasswordIsASecretField proves the archive
// password is handled as a secret on a plain local destination, not just on the
// remote types that carry transport credentials. The password protects the
// archive contents rather than the connection, so a destination that only
// writes to the server's own disk still needs the full masking,
// preserve-on-empty and explicit-clear treatment.
func TestBackupDestinationEncryptionPasswordIsASecretField(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())

	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(&model.BackupDestination{}); err != nil {
		t.Fatalf("failed to migrate backup destinations: %v", err)
	}
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	createTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationCreate)
	rec := postCreateBackupDestination(t, e, token, "local", `{"encrypt_enabled":true,"encryption_password":"archive-passphrase"}`, createTicket)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	created := assertEncryptionPasswordMasked(t, rec, true)
	if strings.Contains(rec.Body.String(), "archive-passphrase") {
		t.Fatalf("create response leaked the encryption password; body = %s", rec.Body.String())
	}

	t.Run("an empty submitted value preserves the stored password", func(t *testing.T) {
		// The client re-sends the masked (empty) field it was given. Success is
		// itself the proof that the merge carried the old value forward: an
		// encryption-enabled plan with a genuinely empty password is rejected by
		// the config validator, so a lost secret would surface as a 400 here.
		ticket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, created.ID, created.Revision)
		rec := putBackupDestinationConfig(t, e, token, created.ID, created.Revision, `{"encrypt_enabled":true,"encryption_password":""}`, nil, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("update status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		created = assertEncryptionPasswordMasked(t, rec, true)
	})

	t.Run("clearing the password while encryption stays on is refused", func(t *testing.T) {
		// Clearing is a real write, not a no-op: it has to collide with the
		// invariant that an encrypting plan needs a password. If the clear were
		// silently ignored, this request would succeed.
		ticket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, created.ID, created.Revision)
		rec := putBackupDestinationConfig(
			t, e, token, created.ID, created.Revision,
			`{"encrypt_enabled":true,"encryption_password":""}`,
			[]string{"encryption_password"},
			ticket,
		)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !hasErrorCode(rec.Body.String(), "encryption_password_is_required_when_backup_encryption_is_enabled") {
			t.Fatalf("body = %s, want the encryption-password-required error", rec.Body.String())
		}
	})

	t.Run("cleared_secret_fields wipes the stored password", func(t *testing.T) {
		// The rejected update above left the row untouched, so the revision is
		// still the one the previous successful write produced.
		ticket := destinationReauthTicket(t, e, token, servicereauth.ReauthOperationBackupDestinationUpdate, created.ID, created.Revision)
		rec := putBackupDestinationConfig(
			t, e, token, created.ID, created.Revision,
			`{"encrypt_enabled":false,"encryption_password":""}`,
			[]string{"encryption_password"},
			ticket,
		)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		assertEncryptionPasswordMasked(t, rec, false)
	})
}
