package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	backupsettings "github.com/kasuha07/subdux/internal/service/settings"
)

// decodeStoredConfig returns the decrypted config map for a persisted
// destination row, so tests can assert on what was actually stored.
func decodeStoredConfig(t *testing.T, svc *Service, id uint) map[string]any {
	t.Helper()

	var destination model.BackupDestination
	if err := svc.DB.First(&destination, id).Error; err != nil {
		t.Fatalf("load destination %d: %v", id, err)
	}
	plain, err := decryptDestinationConfig(destination.Config)
	if err != nil {
		t.Fatalf("decrypt config: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(plain), &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	return parsed
}

// TestCreateDestinationEncryptsSecretsAtRest confirms the CRUD create path
// stores an encrypted config and round-trips it back to plaintext for use.
func TestCreateDestinationLocalRoundTrip(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	dir := filepath.Join(t.TempDir(), "backups")
	id := createLocalDestination(t, svc, dir, 9)

	config := decodeStoredConfig(t, svc, id)
	if config["dir"] != dir {
		t.Fatalf("stored dir = %v, want %v", config["dir"], dir)
	}
	if config["retention_count"] != float64(9) {
		t.Fatalf("stored retention_count = %v, want 9", config["retention_count"])
	}
}

func TestCreateDestinationRejectsUnknownType(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	if _, err := svc.CreateDestination(CreateDestinationInput{Type: "dropbox", Config: "{}"}); err == nil {
		t.Fatal("expected CreateDestination to reject an unknown type")
	}
}

func TestCreateDestinationRejectsInvalidLocalDir(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	// A relative path escaping via ".." is rejected by normalizeBackupLocalDir.
	if _, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "local",
		Config: `{"dir":"../../etc"}`,
	}); err == nil {
		t.Fatal("expected CreateDestination to reject a traversal directory")
	}
}

func TestCreateDestinationReturnsSanitizedView(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"topsecret"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	if strings.Contains(created.Config, "topsecret") || strings.Contains(created.Config, "v1:") {
		t.Fatalf("create view leaked secret material: %q", created.Config)
	}
	assertDestinationSecretMasked(t, created.Config, "secret_access_key")
	if created.Config == "" {
		t.Fatal("create view config is empty")
	}
	if len(created.ConfiguredSecretFields) != 1 || created.ConfiguredSecretFields[0] != "secret_access_key" {
		t.Fatalf("configured secret fields = %v, want [secret_access_key]", created.ConfiguredSecretFields)
	}
}

func TestUpdateDestinationReturnsSanitizedView(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "s3",
		Config: `{"endpoint":"s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"original"}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	updated, err := svc.UpdateDestination(created.ID, UpdateDestinationInput{
		Revision: created.Revision,
		Config:   strPtr(`{"endpoint":"s3.example.com","bucket":"b2","access_key_id":"AKIA","secret_access_key":"rotated"}`),
	})
	if err != nil {
		t.Fatalf("UpdateDestination() error = %v", err)
	}

	if strings.Contains(updated.Config, "rotated") || strings.Contains(updated.Config, "v1:") {
		t.Fatalf("update view leaked secret material: %q", updated.Config)
	}
	assertDestinationSecretMasked(t, updated.Config, "secret_access_key")
	if len(updated.ConfiguredSecretFields) != 1 || updated.ConfiguredSecretFields[0] != "secret_access_key" {
		t.Fatalf("configured secret fields = %v, want [secret_access_key]", updated.ConfiguredSecretFields)
	}
	if stored := decodeStoredConfig(t, svc, created.ID); stored["secret_access_key"] != "rotated" {
		t.Fatalf("stored secret = %v, want rotated", stored["secret_access_key"])
	}
}

func TestDestinationRevisionGuardsUpdateAndDelete(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{
		Type:   "local",
		Config: `{}`,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	if created.Revision != 1 {
		t.Fatalf("created revision = %d, want 1", created.Revision)
	}

	updated, err := svc.UpdateDestination(created.ID, UpdateDestinationInput{
		Revision: created.Revision,
		Enabled:  boolPtr(true),
	})
	if err != nil {
		t.Fatalf("UpdateDestination() error = %v", err)
	}
	if updated.Revision != created.Revision+1 {
		t.Fatalf("updated revision = %d, want %d", updated.Revision, created.Revision+1)
	}

	if _, err := svc.UpdateDestination(created.ID, UpdateDestinationInput{
		Revision: created.Revision,
		Enabled:  boolPtr(false),
	}); !errors.Is(err, ErrBackupDestinationChanged) {
		t.Fatalf("stale UpdateDestination() error = %v, want ErrBackupDestinationChanged", err)
	}
	if err := svc.DeleteDestination(created.ID, created.Revision); !errors.Is(err, ErrBackupDestinationChanged) {
		t.Fatalf("stale DeleteDestination() error = %v, want ErrBackupDestinationChanged", err)
	}
	if err := svc.DeleteDestination(created.ID, updated.Revision); err != nil {
		t.Fatalf("current DeleteDestination() error = %v, want nil", err)
	}
}

func TestUpdateDestinationRejectsDisablingLastEnabledDestinationWhenScheduleEnabled(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destination, err := svc.CreateDestination(CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	enabled := true
	if err := applySettings(t, svc, UpdateSettingsInput{ScheduleEnabled: &enabled}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	disabled := false
	if _, err := svc.UpdateDestination(destination.ID, UpdateDestinationInput{
		Revision: destination.Revision,
		Enabled:  &disabled,
	}); !errors.Is(err, ErrNoEnabledBackupDestination) {
		t.Fatalf("UpdateDestination() error = %v, want ErrNoEnabledBackupDestination", err)
	}

	var stored model.BackupDestination
	if err := svc.DB.First(&stored, destination.ID).Error; err != nil {
		t.Fatalf("load destination after rejected update: %v", err)
	}
	if !stored.Enabled || stored.Revision != destination.Revision {
		t.Fatalf("destination changed after rejected update: enabled=%v revision=%d, want enabled=true revision=%d", stored.Enabled, stored.Revision, destination.Revision)
	}
}

func TestDeleteDestinationRejectsDeletingLastEnabledDestinationWhenScheduleEnabled(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destination, err := svc.CreateDestination(CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	enabled := true
	if err := applySettings(t, svc, UpdateSettingsInput{ScheduleEnabled: &enabled}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	if err := svc.DeleteDestination(destination.ID, destination.Revision); !errors.Is(err, ErrNoEnabledBackupDestination) {
		t.Fatalf("DeleteDestination() error = %v, want ErrNoEnabledBackupDestination", err)
	}

	var stored model.BackupDestination
	if err := svc.DB.First(&stored, destination.ID).Error; err != nil {
		t.Fatalf("load destination after rejected delete: %v", err)
	}
	if stored.Revision != destination.Revision || !stored.Enabled {
		t.Fatalf("destination changed after rejected delete: enabled=%v revision=%d, want enabled=true revision=%d", stored.Enabled, stored.Revision, destination.Revision)
	}
}

func TestDestinationDisableAndDeleteRemainAllowedWhenScheduleDisabled(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destination, err := svc.CreateDestination(CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	disabled := false
	updated, err := svc.UpdateDestination(destination.ID, UpdateDestinationInput{
		Revision: destination.Revision,
		Enabled:  &disabled,
	})
	if err != nil {
		t.Fatalf("UpdateDestination() error = %v, want nil with schedule disabled", err)
	}
	if updated.Enabled {
		t.Fatal("destination remains enabled after schedule-disabled update")
	}
	if err := svc.DeleteDestination(destination.ID, updated.Revision); err != nil {
		t.Fatalf("DeleteDestination() error = %v, want nil with schedule disabled", err)
	}
}

func TestUpdateDestinationRollsBackWhenUpdatedViewCannotBeBuilt(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destination, err := svc.CreateDestination(CreateDestinationInput{Type: "local", Enabled: true, Config: `{}`})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	corruptConfig := `{"dir":`
	if err := svc.DB.Model(&model.BackupDestination{}).
		Where("id = ?", destination.ID).
		Update("config", corruptConfig).Error; err != nil {
		t.Fatalf("corrupt destination config: %v", err)
	}

	disabled := false
	if _, err := svc.UpdateDestination(destination.ID, UpdateDestinationInput{
		Revision: destination.Revision,
		Enabled:  &disabled,
	}); !errors.Is(err, ErrInvalidBackupDestinationConfig) {
		t.Fatalf("UpdateDestination() error = %v, want ErrInvalidBackupDestinationConfig", err)
	}

	var stored model.BackupDestination
	if err := svc.DB.First(&stored, destination.ID).Error; err != nil {
		t.Fatalf("load destination after rejected update: %v", err)
	}
	if !stored.Enabled || stored.Revision != destination.Revision || stored.Config != corruptConfig {
		t.Fatalf("destination changed after view failure: enabled=%v revision=%d config=%q, want enabled=true revision=%d config=%q", stored.Enabled, stored.Revision, stored.Config, destination.Revision, corruptConfig)
	}
}

func assertDestinationSecretMasked(t *testing.T, config, field string) {
	t.Helper()

	var parsed map[string]any
	if err := json.Unmarshal([]byte(config), &parsed); err != nil {
		t.Fatalf("unmarshal destination view config: %v", err)
	}
	if parsed[field] != "" {
		t.Fatalf("destination view %s = %v, want empty mask", field, parsed[field])
	}
}

// TestSanitizeDestinationConfigMasksSecrets exercises the destination DTO
// sanitizer directly against the s3 secret-field table, so the response
// masking contract is covered before the s3 target itself lands.
func TestSanitizeDestinationConfigMasksSecrets(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-backup-settings-key")

	plain := `{"bucket":"b","access_key_id":"AKIA","secret_access_key":"topsecret","unexpected_secret":"do-not-return","nested":{"token":"also-secret"}}`
	encrypted, err := encryptDestinationConfig(plain)
	if err != nil {
		t.Fatalf("encryptDestinationConfig() error = %v", err)
	}
	// A secret must not survive in the encrypted-at-rest blob in plaintext.
	if strings.Contains(encrypted, "topsecret") {
		t.Fatal("secret stored in plaintext")
	}

	view, err := destinationView(model.BackupDestination{Type: "s3", Config: encrypted})
	if err != nil {
		t.Fatalf("destinationView() error = %v", err)
	}
	masked, configured := view.Config, view.ConfiguredSecretFields

	var parsed map[string]any
	if err := json.Unmarshal([]byte(masked), &parsed); err != nil {
		t.Fatalf("unmarshal masked config: %v", err)
	}
	if parsed["secret_access_key"] != "" {
		t.Fatalf("masked secret_access_key = %v, want empty", parsed["secret_access_key"])
	}
	if parsed["bucket"] != "b" || parsed["access_key_id"] != "AKIA" {
		t.Fatalf("non-secret fields altered: %+v", parsed)
	}
	if _, ok := parsed["unexpected_secret"]; ok {
		t.Fatalf("destination view returned an unknown config field: %+v", parsed)
	}
	if _, ok := parsed["nested"]; ok {
		t.Fatalf("destination view returned an unknown nested config field: %+v", parsed)
	}
	if len(configured) != 1 || configured[0] != "secret_access_key" {
		t.Fatalf("configured secret fields = %v, want [secret_access_key]", configured)
	}
}

func TestSanitizeDestinationConfigHidesWebDAVURLCredentials(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-backup-settings-key")

	encrypted, err := encryptDestinationConfig(`{"url":"https://user:topsecret@dav.example.com/dav","path":"backups","username":"dav-user","password":"dav-password"}`)
	if err != nil {
		t.Fatalf("encryptDestinationConfig() error = %v", err)
	}

	view, err := destinationView(model.BackupDestination{Type: "webdav", Config: encrypted})
	if err != nil {
		t.Fatalf("destinationView() error = %v", err)
	}
	masked, configured := view.Config, view.ConfiguredSecretFields

	var parsed map[string]any
	if err := json.Unmarshal([]byte(masked), &parsed); err != nil {
		t.Fatalf("unmarshal masked config: %v", err)
	}
	if parsed["url"] != "" {
		t.Fatalf("unsafe WebDAV URL = %v, want empty", parsed["url"])
	}
	if parsed["path"] != "backups" || parsed["username"] != "dav-user" {
		t.Fatalf("safe WebDAV fields altered: %+v", parsed)
	}
	if parsed["password"] != "" {
		t.Fatalf("WebDAV password = %v, want empty mask", parsed["password"])
	}
	if len(configured) != 1 || configured[0] != "password" {
		t.Fatalf("configured secret fields = %v, want [password]", configured)
	}
}

func TestSanitizeDestinationConfigHidesS3EndpointCredentials(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-backup-settings-key")

	encrypted, err := encryptDestinationConfig(`{"endpoint":"https://user:topsecret@s3.example.com","bucket":"b","access_key_id":"AKIA","secret_access_key":"s3-secret"}`)
	if err != nil {
		t.Fatalf("encryptDestinationConfig() error = %v", err)
	}

	view, err := destinationView(model.BackupDestination{Type: "s3", Config: encrypted})
	if err != nil {
		t.Fatalf("destinationView() error = %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(view.Config), &parsed); err != nil {
		t.Fatalf("unmarshal masked config: %v", err)
	}
	if parsed["endpoint"] != "" {
		t.Fatalf("unsafe S3 endpoint = %v, want empty", parsed["endpoint"])
	}
	if parsed["bucket"] != "b" || parsed["access_key_id"] != "AKIA" {
		t.Fatalf("safe S3 fields altered: %+v", parsed)
	}
	if parsed["secret_access_key"] != "" {
		t.Fatalf("S3 secret_access_key = %v, want empty mask", parsed["secret_access_key"])
	}
	if len(view.ConfiguredSecretFields) != 1 || view.ConfiguredSecretFields[0] != "secret_access_key" {
		t.Fatalf("configured secret fields = %v, want [secret_access_key]", view.ConfiguredSecretFields)
	}
}

func TestListDestinationsRejectsUnreadableConfig(t *testing.T) {
	tests := []struct {
		name   string
		config string
	}{
		{name: "decrypt failure", config: "enc:v1:not-valid-ciphertext"},
		{name: "parse failure", config: `{"bucket":`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newBackupTestDB(t)
			if err := svc.DB.Create(&model.BackupDestination{
				Revision: 1,
				Type:     "s3",
				Config:   tc.config,
			}).Error; err != nil {
				t.Fatalf("create corrupt destination: %v", err)
			}

			_, err := svc.ListDestinations()
			if err == nil {
				t.Fatal("ListDestinations() error = nil, want an explicit config error")
			}
			kind, ok := serviceerr.KindOf(err)
			if !ok || kind != serviceerr.KindInvalid {
				t.Fatalf("ListDestinations() error kind = %v (ok=%t), want KindInvalid; err = %v", kind, ok, err)
			}
			var typedErr *serviceerr.Error
			if !errors.As(err, &typedErr) {
				t.Fatalf("ListDestinations() error is not typed: %v", err)
			}
			if typedErr.Code != ErrInvalidBackupDestinationConfig.Code {
				t.Fatalf("ListDestinations() error code = %q, want %q; err = %v", typedErr.Code, ErrInvalidBackupDestinationConfig.Code, err)
			}
		})
	}
}

// TestMergeDestinationConfigSecretHandling covers the three secret paths on
// update: preserve when submitted empty, replace when submitted non-empty, and
// wipe when explicitly cleared.
func TestMergeDestinationConfigSecretHandling(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-backup-settings-key")

	existing, err := encryptDestinationConfig(`{"bucket":"b","secret_access_key":"original"}`)
	if err != nil {
		t.Fatalf("encrypt existing: %v", err)
	}

	decode := func(t *testing.T, raw string) map[string]any {
		t.Helper()
		var parsed map[string]any
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			t.Fatalf("unmarshal merged config: %v", err)
		}
		return parsed
	}

	t.Run("preserve when empty", func(t *testing.T) {
		merged, err := mergeDestinationConfigWithExistingSecrets("s3", existing, `{"bucket":"b2","secret_access_key":""}`, nil)
		if err != nil {
			t.Fatalf("merge error = %v", err)
		}
		config := decode(t, merged)
		if config["secret_access_key"] != "original" {
			t.Fatalf("secret = %v, want preserved original", config["secret_access_key"])
		}
		if config["bucket"] != "b2" {
			t.Fatalf("bucket = %v, want b2", config["bucket"])
		}
	})

	t.Run("replace when provided", func(t *testing.T) {
		merged, err := mergeDestinationConfigWithExistingSecrets("s3", existing, `{"secret_access_key":"rotated"}`, nil)
		if err != nil {
			t.Fatalf("merge error = %v", err)
		}
		if decode(t, merged)["secret_access_key"] != "rotated" {
			t.Fatal("secret was not replaced with the submitted value")
		}
	})

	t.Run("wipe when cleared", func(t *testing.T) {
		merged, err := mergeDestinationConfigWithExistingSecrets("s3", existing, `{"secret_access_key":""}`, []string{"secret_access_key"})
		if err != nil {
			t.Fatalf("merge error = %v", err)
		}
		if decode(t, merged)["secret_access_key"] != "" {
			t.Fatal("secret was not cleared")
		}
	})
}

// TestListDestinationsReturnsRows confirms the CRUD list path returns created
// local destinations in order.
func TestListDestinationsReturnsRows(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	createLocalDestination(t, svc, filepath.Join(t.TempDir(), "a"), 7)
	createLocalDestination(t, svc, filepath.Join(t.TempDir(), "b"), 3)

	views, err := svc.ListDestinations()
	if err != nil {
		t.Fatalf("ListDestinations() error = %v", err)
	}
	if len(views) != 2 {
		t.Fatalf("ListDestinations() = %d rows, want 2", len(views))
	}
	for _, view := range views {
		if view.Type != "local" {
			t.Fatalf("unexpected destination type %q", view.Type)
		}
	}
}

func TestDeleteDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	created, err := svc.CreateDestination(CreateDestinationInput{Type: "local", Config: `{}`})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	if err := svc.DeleteDestination(created.ID, created.Revision); err != nil {
		t.Fatalf("DeleteDestination() error = %v", err)
	}
	if err := svc.DeleteDestination(created.ID, created.Revision); err == nil {
		t.Fatal("expected DeleteDestination to fail for a missing row")
	}
}

// TestRunBackupPartialFailureDeliversToHealthyTargets confirms that when one of
// several enabled destinations fails, the others still receive the archive and
// the run is not reported as a total failure.
func TestRunBackupPartialFailureDeliversToHealthyTargets(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	fixedNow := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	goodDir := filepath.Join(t.TempDir(), "good")
	goodID := createLocalDestination(t, svc, goodDir, 7)

	// A destination whose parent path is a regular file cannot be created and
	// fails at Put time.
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	badID := createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)

	result, err := svc.RunBackup(context.Background())
	if err != nil {
		t.Fatalf("RunBackup() error = %v, want nil for partial success", err)
	}
	if result.Status != StatusPartial || result.DeliveryStatus != StatusPartial {
		t.Fatalf("RunBackup() aggregate status = %q/%q, want partial/partial", result.Status, result.DeliveryStatus)
	}
	if len(result.Results) != 2 {
		t.Fatalf("RunBackup() results = %d, want 2", len(result.Results))
	}

	outcome := map[uint]DestinationRunResult{}
	for _, r := range result.Results {
		outcome[r.DestinationID] = r
	}
	if !outcome[goodID].Success {
		t.Fatalf("good destination not successful: %+v", outcome[goodID])
	}
	if outcome[badID].Success {
		t.Fatalf("bad destination unexpectedly succeeded: %+v", outcome[badID])
	}

	// The archive must actually be present at the healthy destination.
	if _, err := os.Stat(filepath.Join(goodDir, result.ArchiveName)); err != nil {
		t.Fatalf("archive missing at good destination: %v", err)
	}

	// Per-destination status rows reflect the split outcome.
	var good, bad model.BackupDestination
	if err := svc.DB.First(&good, goodID).Error; err != nil {
		t.Fatalf("load good: %v", err)
	}
	if err := svc.DB.First(&bad, badID).Error; err != nil {
		t.Fatalf("load bad: %v", err)
	}
	if good.LastStatus != StatusOK || good.LastRunAt == nil {
		t.Fatalf("good status = %q run_at = %v, want success/set", good.LastStatus, good.LastRunAt)
	}
	if bad.LastStatus != StatusFailed || bad.LastError == "" {
		t.Fatalf("bad status = %q err = %q, want failed/non-empty", bad.LastStatus, bad.LastError)
	}

	var run model.BackupRun
	if err := svc.DB.First(&run, result.RunID).Error; err != nil {
		t.Fatalf("load backup run: %v", err)
	}
	if run.Status != StatusPartial || run.BookkeepingStatus != StatusOK {
		t.Fatalf("persisted run status = %q bookkeeping = %q, want partial/success", run.Status, run.BookkeepingStatus)
	}
	var runDestinations []model.BackupRunDestination
	if err := svc.DB.Where("run_id = ?", result.RunID).Order("destination_id ASC").Find(&runDestinations).Error; err != nil {
		t.Fatalf("load backup run destinations: %v", err)
	}
	if len(runDestinations) != 2 {
		t.Fatalf("persisted run destinations = %d, want 2", len(runDestinations))
	}
	if runDestinations[0].BookkeepingStatus != StatusOK || runDestinations[1].BookkeepingStatus != StatusOK {
		t.Fatalf("run destination bookkeeping = %+v, want both success", runDestinations)
	}
	if run.ArchivePath != "" {
		t.Fatalf("manual run archive_path = %q, want empty after cleanup", run.ArchivePath)
	}
}

// TestRunBackupAllTargetsFail confirms a run where every enabled destination
// fails surfaces as a total-failure error.
func TestRunBackupAllTargetsFail(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	fixedNow := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)

	if _, err := svc.RunBackup(context.Background()); err == nil {
		t.Fatal("expected RunBackup to fail when every destination fails")
	}
}

func TestRunBackupDoesNotTurnDeliveredArchiveIntoRequestFailureWhenStatusWriteFails(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	dir := filepath.Join(t.TempDir(), "backups")
	destinationID := createLocalDestination(t, svc, dir, 7)
	if err := svc.DB.Exec(`
CREATE TRIGGER fail_backup_destination_success_status
BEFORE UPDATE OF last_status ON backup_destinations
WHEN NEW.last_status = 'success'
BEGIN
	SELECT RAISE(ABORT, 'forced destination success status write failure');
END`).Error; err != nil {
		t.Fatalf("create status trigger: %v", err)
	}

	result, err := svc.RunBackup(context.Background())
	if err != nil {
		t.Fatalf("RunBackup() error = %v, want nil after delivered archive", err)
	}
	if len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("RunBackup() results = %+v, want one delivered destination", result.Results)
	}
	if result.Status != StatusPartial || result.BookkeepingStatus != StatusFailed {
		t.Fatalf("RunBackup() status = %q bookkeeping = %q, want partial/failed", result.Status, result.BookkeepingStatus)
	}
	if result.Results[0].BookkeepingStatus != StatusFailed || !strings.Contains(result.Results[0].BookkeepingError, "forced destination success status write failure") {
		t.Fatalf("destination result = %+v, want bookkeeping failure", result.Results[0])
	}
	var run model.BackupRun
	if err := svc.DB.First(&run, result.RunID).Error; err != nil {
		t.Fatalf("load backup run after status write failure: %v", err)
	}
	if run.Status != StatusPartial || run.BookkeepingStatus != StatusFailed {
		t.Fatalf("persisted run status = %q bookkeeping = %q, want partial/failed", run.Status, run.BookkeepingStatus)
	}
	var runDestination model.BackupRunDestination
	if err := svc.DB.Where("run_id = ? AND destination_id = ?", result.RunID, destinationID).First(&runDestination).Error; err != nil {
		t.Fatalf("load run destination after status write failure: %v", err)
	}
	if runDestination.DeliveryStatus != StatusOK || runDestination.BookkeepingStatus != StatusFailed {
		t.Fatalf("persisted destination stages = %q/%q, want success/failed", runDestination.DeliveryStatus, runDestination.BookkeepingStatus)
	}
	if _, err := os.Stat(filepath.Join(dir, result.ArchiveName)); err != nil {
		t.Fatalf("archive missing after status write failure: %v", err)
	}

	var destination model.BackupDestination
	if err := svc.DB.First(&destination, destinationID).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if destination.LastStatus != "" || destination.LastRunAt != nil || destination.LastBookkeepingStatus != "pending" {
		t.Fatalf("destination status = %q run_at = %v bookkeeping = %q, want unchanged after rejected write", destination.LastStatus, destination.LastRunAt, destination.LastBookkeepingStatus)
	}
}

func TestRunScheduledBackupPersistsPartialAndResumesWithoutRedeliveringSuccess(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	enabled := true
	timeOfDay := "03:00"

	goodDir := filepath.Join(t.TempDir(), "good")
	goodID := createLocalDestination(t, svc, goodDir, 7)
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	badID := createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)
	if err := applySettings(t, svc, UpdateSettingsInput{ScheduleEnabled: &enabled, TimeOfDay: &timeOfDay}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	fixedNow := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)
	ownerID := serviceutil.NewBackgroundTaskOwnerID()
	previousRunAt := "2026-06-14T04:00:00Z"
	if err := backupsettings.SaveString(svc.DB, KeyLastRunAt, previousRunAt); err != nil {
		t.Fatalf("seed previous backup last_run_at: %v", err)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("partial RunScheduledBackup() error = %v, want nil", err)
	}
	lastStatus, err := backupsettings.GetString(context.Background(), svc.DB, KeyLastStatus, "")
	if err != nil || lastStatus != StatusPartial {
		t.Fatalf("backup last_status = %q, err = %v, want partial/nil", lastStatus, err)
	}
	lastRunAt, err := backupsettings.GetString(nil, svc.DB, KeyLastRunAt, "")
	if err != nil || lastRunAt != previousRunAt {
		t.Fatalf("backup last_run_at = %q, err = %v, want previous value %q for partial run", lastRunAt, err, previousRunAt)
	}
	if got := countBackupArchives(t, goodDir); got != 1 {
		t.Fatalf("good destination archive count after partial run = %d, want 1", got)
	}
	var partialRun model.BackupRun
	if err := svc.DB.Where("status = ?", StatusPartial).Order("id DESC").First(&partialRun).Error; err != nil {
		t.Fatalf("load partial backup run: %v", err)
	}
	if partialRun.ArchivePath == "" {
		t.Fatal("partial scheduled run archive_path is empty, want persisted spool path")
	}
	archiveInfo, err := os.Stat(partialRun.ArchivePath)
	if err != nil {
		t.Fatalf("stat partial scheduled archive: %v", err)
	}
	if archiveInfo.Mode().Perm() != 0o600 {
		t.Fatalf("partial scheduled archive mode = %o, want 600", archiveInfo.Mode().Perm())
	}
	if err := os.Remove(blockingFile); err != nil {
		t.Fatalf("remove blocking file: %v", err)
	}

	var bad model.BackupDestination
	if err := svc.DB.First(&bad, badID).Error; err != nil {
		t.Fatalf("load bad destination: %v", err)
	}
	if bad.Revision == 0 {
		t.Fatal("bad destination revision = 0, want a revision-bound retry")
	}

	// A finalized-but-incomplete run is retried on a spacing interval, not on
	// every scheduler tick, so the resume only happens once that interval has
	// elapsed. Advance the clock past it before asserting the resume.
	restoreRetry := pkg.SetNowForTest(fixedNow.Add(backupRunRetryInterval + time.Minute))
	t.Cleanup(restoreRetry)

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("resumed RunScheduledBackup() error = %v", err)
	}
	lastStatus, err = backupsettings.GetString(nil, svc.DB, KeyLastStatus, "")
	if err != nil || lastStatus != StatusOK {
		t.Fatalf("backup last_status after resume = %q, err = %v, want success/nil", lastStatus, err)
	}
	lastRunAt, err = backupsettings.GetString(nil, svc.DB, KeyLastRunAt, "")
	if err != nil || lastRunAt == "" || lastRunAt == previousRunAt {
		t.Fatalf("backup last_run_at after resume = %q, err = %v, want set/nil", lastRunAt, err)
	}
	if got := countBackupArchives(t, goodDir); got != 1 {
		t.Fatalf("good destination archive count after resume = %d, want 1", got)
	}
	var good model.BackupDestination
	if err := svc.DB.First(&good, goodID).Error; err != nil {
		t.Fatalf("load good destination: %v", err)
	}
	if good.LastStatus != StatusOK || good.LastBookkeepingStatus != StatusOK {
		t.Fatalf("good destination final status = %q/%q, want success/success", good.LastStatus, good.LastBookkeepingStatus)
	}
	var completedRun model.BackupRun
	if err := svc.DB.First(&completedRun, partialRun.ID).Error; err != nil {
		t.Fatalf("reload completed backup run: %v", err)
	}
	if completedRun.ArchivePath != "" {
		t.Fatalf("completed scheduled run archive_path = %q, want empty after cleanup", completedRun.ArchivePath)
	}
}

func TestRunScheduledBackupStartsNewRunAfterDestinationRevisionChanges(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	enabled := true
	timeOfDay := "03:00"

	goodDir := filepath.Join(t.TempDir(), "good")
	goodID := createLocalDestination(t, svc, goodDir, 7)
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	badID := createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)
	if err := applySettings(t, svc, UpdateSettingsInput{ScheduleEnabled: &enabled, TimeOfDay: &timeOfDay}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	fixedNow := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)
	ownerID := serviceutil.NewBackgroundTaskOwnerID()
	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("initial partial RunScheduledBackup() error = %v", err)
	}

	var oldRun model.BackupRun
	if err := svc.DB.Where("status = ?", StatusPartial).Order("id DESC").First(&oldRun).Error; err != nil {
		t.Fatalf("load old partial run: %v", err)
	}
	var bad model.BackupDestination
	if err := svc.DB.First(&bad, badID).Error; err != nil {
		t.Fatalf("load bad destination: %v", err)
	}
	recoveredConfig := fmt.Sprintf(`{"dir":%q,"retention_count":7}`, filepath.Join(t.TempDir(), "recovered"))
	if _, err := svc.UpdateDestination(badID, UpdateDestinationInput{Revision: bad.Revision, Config: &recoveredConfig}); err != nil {
		t.Fatalf("UpdateDestination() error = %v", err)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("new-run RunScheduledBackup() error = %v", err)
	}
	if got := countBackupArchives(t, goodDir); got != 2 {
		t.Fatalf("good destination archive count after revision boundary = %d, want 2", got)
	}
	var superseded model.BackupRun
	if err := svc.DB.First(&superseded, oldRun.ID).Error; err != nil {
		t.Fatalf("reload old run: %v", err)
	}
	if superseded.Status != StatusSuperseded || superseded.ArchivePath != "" {
		t.Fatalf("old run status/path = %q/%q, want superseded/empty", superseded.Status, superseded.ArchivePath)
	}
	var latest model.BackupRun
	if err := svc.DB.Where("source = ?", backupRunSourceScheduled).Order("id DESC").First(&latest).Error; err != nil {
		t.Fatalf("load latest scheduled run: %v", err)
	}
	if latest.ID == oldRun.ID || latest.Status != StatusOK {
		t.Fatalf("latest run = %+v, want a new successful run", latest)
	}
	var good model.BackupDestination
	if err := svc.DB.First(&good, goodID).Error; err != nil {
		t.Fatalf("reload good destination: %v", err)
	}
	if good.LastStatus != StatusOK {
		t.Fatalf("good destination status = %q, want success", good.LastStatus)
	}
}

func countBackupArchives(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read backup directory %s: %v", dir, err)
	}
	count := 0
	for _, entry := range entries {
		if backupFileNamePattern.MatchString(entry.Name()) {
			count++
		}
	}
	return count
}

func TestRunBackupAggregatesDestinationFailureStatusWriteFailure(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	destinationID := createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)
	if err := svc.DB.Exec(`
CREATE TRIGGER fail_backup_destination_failure_status
BEFORE UPDATE OF last_status ON backup_destinations
WHEN NEW.last_status = 'failed'
BEGIN
	SELECT RAISE(ABORT, 'forced destination failure status write failure');
END`).Error; err != nil {
		t.Fatalf("create status trigger: %v", err)
	}

	_, err := svc.RunBackup(context.Background())
	if !errors.Is(err, ErrAllBackupDestinationsFailed) || !strings.Contains(err.Error(), "forced destination failure status write failure") {
		t.Fatalf("RunBackup() error = %v, want aggregated delivery and status write failures", err)
	}

	var destination model.BackupDestination
	if err := svc.DB.First(&destination, destinationID).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if destination.LastStatus != "" || destination.LastError != "" {
		t.Fatalf("destination status = %q error = %q, want unchanged after rejected write", destination.LastStatus, destination.LastError)
	}
}

// TestPerDestinationRetentionIsIndependent confirms each destination prunes to
// its own retention count.
func TestPerDestinationRetentionIsIndependent(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	keepManyDir := filepath.Join(t.TempDir(), "many")
	keepOneDir := filepath.Join(t.TempDir(), "one")
	createLocalDestination(t, svc, keepManyDir, 5)
	createLocalDestination(t, svc, keepOneDir, 1)

	// Run three backups on distinct clock-seconds so retention has something to
	// prune at the retention=1 destination.
	for i := 0; i < 3; i++ {
		fixedNow := time.Date(2026, 6, 15, 3, 0, i, 0, time.UTC)
		restore := pkg.SetNowForTest(fixedNow)
		if _, err := svc.RunBackup(context.Background()); err != nil {
			restore()
			t.Fatalf("RunBackup() iteration %d error = %v", i, err)
		}
		restore()
	}

	countArchives := func(dir string) int {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir %s: %v", dir, err)
		}
		n := 0
		for _, entry := range entries {
			if backupFileNamePattern.MatchString(entry.Name()) {
				n++
			}
		}
		return n
	}

	if got := countArchives(keepManyDir); got != 3 {
		t.Fatalf("retention=5 destination has %d archives, want 3", got)
	}
	if got := countArchives(keepOneDir); got != 1 {
		t.Fatalf("retention=1 destination has %d archives, want 1", got)
	}
}

// newFailingScheduledRunFixture wires a schedule whose only destination cannot
// be written to, so every run terminates as failed. It returns the service, the
// pinned clock, and the background-task owner id.
func newFailingScheduledRunFixture(t *testing.T) (*Service, time.Time, string) {
	t.Helper()

	svc, _ := newBackupTestDB(t)
	enabled := true
	timeOfDay := "03:00"

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	createLocalDestination(t, svc, filepath.Join(blockingFile, "backups"), 7)
	if err := applySettings(t, svc, UpdateSettingsInput{ScheduleEnabled: &enabled, TimeOfDay: &timeOfDay}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	fixedNow := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)
	t.Cleanup(pkg.SetNowForTest(fixedNow))
	if err := backupsettings.SaveString(svc.DB, KeyLastRunAt, "2026-06-14T04:00:00Z"); err != nil {
		t.Fatalf("seed last_run_at: %v", err)
	}
	return svc, fixedNow, serviceutil.NewBackgroundTaskOwnerID()
}

func countBackupRuns(t *testing.T, svc *Service) int64 {
	t.Helper()

	var count int64
	if err := svc.DB.Model(&model.BackupRun{}).Count(&count).Error; err != nil {
		t.Fatalf("count backup runs: %v", err)
	}
	return count
}

// TestScheduledBackupRetryIsSpacedAcrossTicks pins the failure mode that the
// resume path would otherwise create: the scheduler ticks every minute and the
// task lease is re-acquirable by its own owner, so an incomplete run must not be
// retried on every tick. Before this spacing existed, a permanently failing
// destination consumed a full build and delivery cycle 60 times an hour forever,
// bypassing the configured schedule entirely.
func TestScheduledBackupRetryIsSpacedAcrossTicks(t *testing.T) {
	svc, fixedNow, ownerID := newFailingScheduledRunFixture(t)

	// First tick is due and runs, failing on the unwritable destination.
	_ = svc.RunScheduledBackup(ownerID)
	if status, _ := backupsettings.GetString(context.Background(), svc.DB, KeyLastStatus, ""); status != StatusFailed {
		t.Fatalf("last_status after first run = %q, want %q", status, StatusFailed)
	}
	if got := countBackupRuns(t, svc); got != 1 {
		t.Fatalf("backup run count after first tick = %d, want 1", got)
	}

	// An immediately following tick must be a no-op. A sentinel detects whether
	// the run executed at all, because a real execution rewrites last_status.
	if err := backupsettings.SaveString(svc.DB, KeyLastStatus, "sentinel"); err != nil {
		t.Fatalf("seed sentinel: %v", err)
	}
	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("throttled RunScheduledBackup() error = %v, want nil", err)
	}
	if status, _ := backupsettings.GetString(context.Background(), svc.DB, KeyLastStatus, ""); status != "sentinel" {
		t.Fatalf("last_status after throttled tick = %q, want the run to be skipped", status)
	}
	if got := countBackupRuns(t, svc); got != 1 {
		t.Fatalf("backup run count after throttled tick = %d, want 1 (no duplicate run)", got)
	}

	// Once the retry interval has elapsed the same run resumes.
	t.Cleanup(pkg.SetNowForTest(fixedNow.Add(backupRunRetryInterval + time.Minute)))
	if err := svc.RunScheduledBackup(ownerID); err == nil {
		t.Fatal("resumed RunScheduledBackup() error = nil, want the destination failure")
	}
	if status, _ := backupsettings.GetString(context.Background(), svc.DB, KeyLastStatus, ""); status != StatusFailed {
		t.Fatalf("last_status after retry interval = %q, want %q", status, StatusFailed)
	}
	if got := countBackupRuns(t, svc); got != 1 {
		t.Fatalf("backup run count after resume = %d, want 1 (same run resumed)", got)
	}
}

// TestScheduledBackupResumeWindowSupersedesStaleRun confirms an incomplete run
// cannot own the schedule indefinitely. Past the resume window its staged
// archive is a stale snapshot, so the run is abandoned (releasing its spool) and
// the next due tick takes a fresh backup instead of retrying old bytes forever.
func TestScheduledBackupResumeWindowSupersedesStaleRun(t *testing.T) {
	svc, fixedNow, ownerID := newFailingScheduledRunFixture(t)

	_ = svc.RunScheduledBackup(ownerID)
	var stale model.BackupRun
	if err := svc.DB.Order("id DESC").First(&stale).Error; err != nil {
		t.Fatalf("load stale run: %v", err)
	}
	stagedPath := strings.TrimSpace(stale.ArchivePath)
	if stagedPath == "" {
		t.Fatal("stale run archive_path is empty, want a persisted spool path")
	}

	t.Cleanup(pkg.SetNowForTest(fixedNow.Add(backupRunResumeWindow + time.Hour)))
	_ = svc.RunScheduledBackup(ownerID)

	var superseded model.BackupRun
	if err := svc.DB.First(&superseded, stale.ID).Error; err != nil {
		t.Fatalf("reload stale run: %v", err)
	}
	if superseded.Status != StatusSuperseded {
		t.Fatalf("stale run status = %q, want %q", superseded.Status, StatusSuperseded)
	}
	if superseded.ArchivePath != "" {
		t.Fatalf("superseded run archive_path = %q, want empty", superseded.ArchivePath)
	}
	if _, err := os.Stat(stagedPath); !os.IsNotExist(err) {
		t.Fatalf("stale spool file still present at %s (stat err = %v), want it removed", stagedPath, err)
	}
	if got := countBackupRuns(t, svc); got != 2 {
		t.Fatalf("backup run count = %d, want 2 (stale run abandoned, fresh run started)", got)
	}
}

func TestScheduledRunResumeDue(t *testing.T) {
	finished := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name string
		run  model.BackupRun
		now  time.Time
		want bool
	}{
		{
			// The process died mid-flight, so no attempt was actually spent and
			// there is nothing to back off from.
			name: "never finalized resumes immediately",
			run:  model.BackupRun{StartedAt: finished},
			now:  finished.Add(time.Second),
			want: true,
		},
		{
			name: "within retry interval waits",
			run:  model.BackupRun{StartedAt: finished, FinishedAt: &finished},
			now:  finished.Add(backupRunRetryInterval - time.Second),
			want: false,
		},
		{
			name: "at retry interval resumes",
			run:  model.BackupRun{StartedAt: finished, FinishedAt: &finished},
			now:  finished.Add(backupRunRetryInterval),
			want: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := scheduledRunResumeDue(tc.run, tc.now); got != tc.want {
				t.Fatalf("scheduledRunResumeDue() = %v, want %v", got, tc.want)
			}
		})
	}
}
