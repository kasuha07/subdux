package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
)

func TestTestBackupDestinationReturnsCountAsMessageParams(t *testing.T) {
	t.Setenv("DATA_PATH", t.TempDir())

	db := newHumanOnlyRouteTestDB(t)
	if err := db.AutoMigrate(&model.BackupDestination{}); err != nil {
		t.Fatalf("failed to migrate backup destinations: %v", err)
	}
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
		t.Fatalf("test destination status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode test destination response: %v; body = %s", err, rec.Body.String())
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
	if got, want := messageParams["backup_count"], float64(0); got != want {
		t.Fatalf("message_params.backup_count = %v, want %v", got, want)
	}
	if _, ok := payload["backup_count"]; ok {
		t.Fatalf("response should not put backup_count at the top level; body = %s", rec.Body.String())
	}
}
