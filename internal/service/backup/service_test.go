package backup

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	backupsettings "github.com/kasuha07/subdux/internal/service/settings"
	yekazip "github.com/yeka/zip"
	"gorm.io/gorm"
)

// newBackupTestDB provisions a Service backed by a temp SQLite database with
// the system settings table migrated and DATA_PATH pointed at a temp dir.
func newBackupTestDB(t *testing.T) (*Service, string) {
	t.Helper()

	dataDir := t.TempDir()
	t.Setenv("DATA_PATH", dataDir)
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-backup-settings-key")

	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.BackgroundTaskLease{}); err != nil {
		t.Fatalf("failed to migrate background task leases: %v", err)
	}
	return NewService(db), dataDir
}

// makeBackupDBFile produces a valid SQLite file at a temp path using VACUUM INTO
// so archive builders have a real database to embed.
func makeBackupDBFile(t *testing.T, svc *Service) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "source.db")
	if err := svc.DB.Exec("VACUUM INTO ?", dbPath).Error; err != nil {
		t.Fatalf("VACUUM INTO failed: %v", err)
	}
	return dbPath
}

func applySettings(t *testing.T, svc *Service, input UpdateSettingsInput) error {
	t.Helper()

	return svc.DB.Transaction(func(tx *gorm.DB) error {
		return ApplySettings(tx, input)
	})
}

// createLocalDestination creates and enables a local backup destination with
// the given directory and retention count, returning its ID. An empty dir means
// the default <DATA_PATH>/backups.
func createLocalDestination(t *testing.T, svc *Service, dir string, retention int) uint {
	t.Helper()

	config := fmt.Sprintf(`{"dir":%q,"retention_count":%d}`, dir, retention)
	destination, err := svc.CreateDestination(CreateDestinationInput{
		Type:    "local",
		Enabled: true,
		Config:  config,
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	return destination.ID
}

func TestWriteBackupZipEncryptedRoundTrip(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	dbPath := makeBackupDBFile(t, svc)

	tests := []struct {
		name          string
		includeAssets bool
		wantAssetsDir bool
	}{
		{name: "db only", includeAssets: false, wantAssetsDir: false},
		{name: "with assets", includeAssets: true, wantAssetsDir: true},
	}

	const password = "correct horse battery staple"

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "backup.zip")
			if err := writeBackupZipFromDB(archivePath, dbPath, tc.includeAssets, password); err != nil {
				t.Fatalf("writeBackupZipFromDB() error = %v", err)
			}

			reader, err := yekazip.OpenReader(archivePath)
			if err != nil {
				t.Fatalf("yekazip.OpenReader() error = %v", err)
			}
			defer reader.Close()

			foundDB := false
			foundAssetsDir := false
			for _, entry := range reader.File {
				switch entry.Name {
				case "subdux.db":
					foundDB = true
					if !entry.IsEncrypted() {
						t.Fatal("subdux.db entry should be encrypted")
					}
					entry.SetPassword(password)
					rc, openErr := entry.Open()
					if openErr != nil {
						t.Fatalf("open encrypted entry with correct password: %v", openErr)
					}
					data, readErr := io.ReadAll(rc)
					rc.Close()
					if readErr != nil {
						t.Fatalf("read encrypted entry with correct password: %v", readErr)
					}
					if len(data) == 0 {
						t.Fatal("decrypted subdux.db is empty")
					}
				case "assets/":
					foundAssetsDir = true
				}
			}

			if !foundDB {
				t.Fatal("archive missing subdux.db entry")
			}
			if tc.wantAssetsDir && !foundAssetsDir {
				t.Fatal("archive missing assets/ directory entry")
			}
			if !tc.wantAssetsDir && foundAssetsDir {
				t.Fatal("archive unexpectedly contains assets/ directory entry")
			}

			// Wrong password must fail to decrypt.
			wrongReader, err := yekazip.OpenReader(archivePath)
			if err != nil {
				t.Fatalf("reopen archive: %v", err)
			}
			defer wrongReader.Close()
			for _, entry := range wrongReader.File {
				if entry.Name != "subdux.db" {
					continue
				}
				entry.SetPassword("wrong-password")
				rc, openErr := entry.Open()
				if openErr != nil {
					break
				}
				if _, readErr := io.ReadAll(rc); readErr == nil {
					rc.Close()
					t.Fatal("reading with wrong password unexpectedly succeeded")
				}
				rc.Close()
			}
		})
	}
}

func TestWriteBackupZipPlainRestoreCompatible(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	dbPath := makeBackupDBFile(t, svc)

	archivePath := filepath.Join(t.TempDir(), "backup.zip")
	if err := writeBackupZipFromDB(archivePath, dbPath, false, ""); err != nil {
		t.Fatalf("writeBackupZipFromDB() error = %v", err)
	}
	archiveInfo, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("stat archive: %v", err)
	}
	if got := archiveInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("archive permissions = %04o, want 0600", got)
	}

	// A plain archive must be readable by the standard library archive/zip so it
	// remains restore-compatible with the existing download path.
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("stdlib zip.OpenReader() error = %v", err)
	}
	defer reader.Close()

	foundDB := false
	for _, entry := range reader.File {
		if entry.Name != "subdux.db" {
			continue
		}
		foundDB = true
		if entry.Flags&0x1 != 0 {
			t.Fatal("plain archive subdux.db entry should not be encrypted")
		}
		rc, openErr := entry.Open()
		if openErr != nil {
			t.Fatalf("open plain entry: %v", openErr)
		}
		header := make([]byte, len(sqliteFileHeader))
		if _, readErr := io.ReadFull(rc, header); readErr != nil {
			rc.Close()
			t.Fatalf("read plain entry header: %v", readErr)
		}
		rc.Close()
		if string(header) != string(sqliteFileHeader) {
			t.Fatalf("subdux.db does not start with SQLite header: %q", header)
		}
	}
	if !foundDB {
		t.Fatal("plain archive missing subdux.db entry")
	}
}

func TestApplyLocalBackupRetention(t *testing.T) {
	dir := t.TempDir()

	matching := []string{
		"subdux-backup-20260101-000000.zip",
		"subdux-backup-20260102-000000.zip",
		"subdux-backup-20260103-000000.zip",
		"subdux-backup-20260104-000000.zip",
		"subdux-backup-20260105-000000.zip",
	}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, name := range matching {
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte("dummy"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		modTime := base.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(full, modTime, modTime); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	nonMatching := []string{"other.txt", "subdux.db", "backup-notzip.zip.bak", "subdux-backup.txt"}
	for _, name := range nonMatching {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("keep"), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	target, err := newLocalTarget(map[string]any{"dir": dir, "retention_count": 2})
	if err != nil {
		t.Fatalf("newLocalTarget() error = %v", err)
	}
	if err := applyTargetRetention(context.Background(), target); err != nil {
		t.Fatalf("applyTargetRetention() error = %v", err)
	}

	wantKept := map[string]bool{
		"subdux-backup-20260105-000000.zip": true,
		"subdux-backup-20260104-000000.zip": true,
	}
	for _, name := range matching {
		_, err := os.Stat(filepath.Join(dir, name))
		if wantKept[name] {
			if err != nil {
				t.Fatalf("expected %s to be kept, got error %v", name, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected %s to be deleted, stat err = %v", name, err)
		}
	}

	for _, name := range nonMatching {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("non-matching file %s should be untouched, got %v", name, err)
		}
	}
}

func TestBackupDue(t *testing.T) {
	utc := time.UTC
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load Asia/Shanghai: %v", err)
	}

	tests := []struct {
		name      string
		now       time.Time
		timeOfDay string
		lastRunAt time.Time
		loc       *time.Location
		want      bool
	}{
		{
			name:      "never run and time passed",
			now:       time.Date(2026, 5, 20, 4, 0, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Time{},
			loc:       utc,
			want:      true,
		},
		{
			name:      "never run but time not reached",
			now:       time.Date(2026, 5, 20, 2, 0, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Time{},
			loc:       utc,
			want:      false,
		},
		{
			name:      "already ran today",
			now:       time.Date(2026, 5, 20, 5, 0, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Date(2026, 5, 20, 3, 0, 5, 0, utc),
			loc:       utc,
			want:      false,
		},
		{
			name:      "ran yesterday and time passed",
			now:       time.Date(2026, 5, 20, 3, 30, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Date(2026, 5, 19, 3, 0, 0, 0, utc),
			loc:       utc,
			want:      true,
		},
		{
			name:      "exactly at scheduled minute",
			now:       time.Date(2026, 5, 20, 3, 0, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Time{},
			loc:       utc,
			want:      true,
		},
		{
			name:      "invalid time of day",
			now:       time.Date(2026, 5, 20, 12, 0, 0, 0, utc),
			timeOfDay: "25:00",
			lastRunAt: time.Time{},
			loc:       utc,
			want:      false,
		},
		{
			name:      "shanghai time reached",
			now:       time.Date(2026, 5, 20, 20, 30, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Time{},
			loc:       shanghai,
			want:      true,
		},
		{
			name:      "shanghai already ran same local day",
			now:       time.Date(2026, 5, 20, 20, 30, 0, 0, utc),
			timeOfDay: "03:00",
			lastRunAt: time.Date(2026, 5, 20, 19, 5, 0, 0, utc),
			loc:       shanghai,
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := backupDue(tc.now, tc.timeOfDay, tc.lastRunAt, tc.loc); got != tc.want {
				t.Fatalf("backupDue() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApplySettingsHappyPath(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	createLocalDestination(t, svc, "", 7)

	enabled := true
	timeOfDay := "02:30"
	includeAssets := true

	if err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
		IncludeAssets:   &includeAssets,
	}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	gotEnabled, err := backupsettings.GetBool(context.Background(), svc.DB, KeyScheduleEnabled, false)
	if err != nil || !gotEnabled {
		t.Fatalf("backup schedule = %v, err = %v, want true/nil", gotEnabled, err)
	}
	gotTime, err := backupsettings.GetString(nil, svc.DB, KeyTimeOfDay, "")
	if err != nil || gotTime != timeOfDay {
		t.Fatalf("backup time = %q, err = %v, want %q/nil", gotTime, err, timeOfDay)
	}
	gotIncludeAssets, err := backupsettings.GetBool(nil, svc.DB, KeyIncludeAssets, false)
	if err != nil || !gotIncludeAssets {
		t.Fatalf("backup include_assets = %v, err = %v, want true/nil", gotIncludeAssets, err)
	}
	gotPassword, err := backupsettings.GetString(nil, svc.DB, KeyEncryptionPassword, "")
	if err != nil || gotPassword != "" {
		t.Fatalf("backup password = %q, err = %v, want empty/nil", gotPassword, err)
	}
}

func TestApplySettingsRejectsEnablingScheduleWithoutEnabledDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	enabled := true
	timeOfDay := "02:30"
	includeAssets := true
	err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
		IncludeAssets:   &includeAssets,
	})
	if !errors.Is(err, ErrNoEnabledBackupDestination) {
		t.Fatalf("ApplySettings() error = %v, want ErrNoEnabledBackupDestination", err)
	}

	scheduleEnabled, err := backupsettings.GetBool(nil, svc.DB, KeyScheduleEnabled, false)
	if err != nil {
		t.Fatalf("read backup schedule: %v", err)
	}
	if scheduleEnabled {
		t.Fatal("backup schedule persisted as enabled after rejected transaction")
	}
	storedTimeOfDay, err := backupsettings.GetString(nil, svc.DB, KeyTimeOfDay, "03:00")
	if err != nil {
		t.Fatalf("read backup time after rejected transaction: %v", err)
	}
	if storedTimeOfDay != "03:00" {
		t.Fatalf("backup time persisted as %q after rejected transaction, want default 03:00", storedTimeOfDay)
	}
	storedIncludeAssets, err := backupsettings.GetBool(nil, svc.DB, KeyIncludeAssets, false)
	if err != nil {
		t.Fatalf("read include_assets after rejected transaction: %v", err)
	}
	if storedIncludeAssets {
		t.Fatal("backup include_assets persisted after rejected transaction")
	}
}

func TestApplySettingsValidation(t *testing.T) {
	tests := []struct {
		name    string
		input   UpdateSettingsInput
		wantErr error
	}{
		{
			name:    "invalid time of day",
			input:   UpdateSettingsInput{TimeOfDay: strPtr("24:00")},
			wantErr: ErrInvalidBackupTimeOfDay,
		},
		{
			name:    "encrypt enabled without password",
			input:   UpdateSettingsInput{EncryptEnabled: boolPtr(true)},
			wantErr: ErrBackupEncryptionPasswordRequired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newBackupTestDB(t)
			err := applySettings(t, svc, tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ApplySettings() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestApplySettingsEncryptionPasswordFlow(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	password := "s3cr3t-backup"
	enable := true
	if err := applySettings(t, svc, UpdateSettingsInput{
		EncryptEnabled:     &enable,
		EncryptionPassword: &password,
	}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	gotEnabled, err := backupsettings.GetBool(nil, svc.DB, KeyEncryptEnabled, false)
	if err != nil || !gotEnabled {
		t.Fatalf("backup encrypt_enabled = %v, err = %v, want true/nil", gotEnabled, err)
	}

	var stored model.SystemSetting
	if err := svc.DB.Where("key = ?", KeyEncryptionPassword).First(&stored).Error; err != nil {
		t.Fatalf("read stored password: %v", err)
	}
	if stored.Value == password {
		t.Fatal("backup encryption password stored in plaintext")
	}

	if err := applySettings(t, svc, UpdateSettingsInput{EncryptEnabled: &enable}); err != nil {
		t.Fatalf("re-enable with stored password error = %v", err)
	}

	cfg, err := svc.loadRuntimeConfig()
	if err != nil {
		t.Fatalf("loadRuntimeConfig() error = %v", err)
	}
	if cfg.EncryptPassword != password {
		t.Fatalf("decrypted password = %q, want %q", cfg.EncryptPassword, password)
	}
}

func TestRunBackupAndList(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)

	fixedNow := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	wantDir := filepath.Join(dataDir, "backups")
	destID := createLocalDestination(t, svc, "", 7)

	result, err := svc.RunBackup(context.Background())
	if err != nil {
		t.Fatalf("RunBackup() error = %v", err)
	}
	if len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("RunBackup() results = %+v, want one successful delivery", result.Results)
	}
	if result.Results[0].DestinationID != destID {
		t.Fatalf("RunBackup() destination id = %d, want %d", result.Results[0].DestinationID, destID)
	}

	if _, err := os.Stat(filepath.Join(wantDir, result.ArchiveName)); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	dir, items, err := svc.ListLocalBackups()
	if err != nil {
		t.Fatalf("ListLocalBackups() error = %v", err)
	}
	if dir != wantDir {
		t.Fatalf("ListLocalBackups dir = %q, want %q", dir, wantDir)
	}
	if len(items) != 1 {
		t.Fatalf("ListLocalBackups items = %d, want 1", len(items))
	}
	if items[0].Name != result.ArchiveName {
		t.Fatalf("listed name = %q, want %q", items[0].Name, result.ArchiveName)
	}
	if items[0].Encrypted {
		t.Fatal("plain backup should not be reported as encrypted")
	}

	// A destination-scoped listing must agree with the local convenience view.
	objects, err := svc.ListDestinationBackups(context.Background(), destID)
	if err != nil {
		t.Fatalf("ListDestinationBackups() error = %v", err)
	}
	if len(objects) != 1 || objects[0].Name != result.ArchiveName {
		t.Fatalf("ListDestinationBackups() = %+v, want one archive %q", objects, result.ArchiveName)
	}
}

// TestRunBackupWithoutDestinationFails confirms a run with no enabled
// destination is a hard error rather than a silent no-op.
func TestRunBackupWithoutDestinationFails(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	_, err := svc.RunBackup(context.Background())
	if !errors.Is(err, ErrNoEnabledBackupDestination) {
		t.Fatalf("RunBackup() error = %v, want ErrNoEnabledBackupDestination", err)
	}
}

func TestStageBackupArchiveRejectsBackslashInArchiveName(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	archivePath := filepath.Join(t.TempDir(), "source.zip")
	if err := os.WriteFile(archivePath, []byte("archive"), 0o600); err != nil {
		t.Fatalf("write source archive: %v", err)
	}

	if _, err := svc.stageBackupArchive(1, `subdux-backup-\\evil.zip`, archivePath); !errors.Is(err, ErrBackupArchiveUnavailable) {
		t.Fatalf("stageBackupArchive() error = %v, want ErrBackupArchiveUnavailable", err)
	}
}

func TestCleanupBackupRunArchiveKeepsPointerWhenRemovalFails(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)
	archivePath := filepath.Join(dataDir, backupStagingDirName, "1-subdux-backup-test.zip")
	if err := os.MkdirAll(filepath.Join(archivePath, "contents"), 0o700); err != nil {
		t.Fatalf("create non-empty spool directory: %v", err)
	}

	run := model.BackupRun{
		Source:                  backupRunSourceScheduled,
		ArchiveName:             "subdux-backup-test.zip",
		ArchivePath:             archivePath,
		Status:                  StatusOK,
		DeliveryStatus:          StatusOK,
		RetentionStatus:         StatusOK,
		BookkeepingStatus:       StatusOK,
		GlobalBookkeepingStatus: StatusOK,
		StartedAt:               pkg.Now(),
	}
	if err := svc.DB.Create(&run).Error; err != nil {
		t.Fatalf("create backup run: %v", err)
	}

	if err := svc.cleanupBackupRunArchive(run.ID); err == nil {
		t.Fatal("cleanupBackupRunArchive() succeeded for a non-empty spool directory")
	}
	var stored model.BackupRun
	if err := svc.DB.First(&stored, run.ID).Error; err != nil {
		t.Fatalf("reload backup run: %v", err)
	}
	if stored.ArchivePath != archivePath {
		t.Fatalf("archive_path = %q, want retained pointer %q", stored.ArchivePath, archivePath)
	}
	if stored.GlobalBookkeepingStatus != StatusFailed {
		t.Fatalf("global bookkeeping status = %q, want failed", stored.GlobalBookkeepingStatus)
	}
	state, err := svc.findResumableScheduledRun()
	if err != nil {
		t.Fatalf("findResumableScheduledRun() error = %v", err)
	}
	if state == nil || state.run.ID != run.ID {
		t.Fatalf("findResumableScheduledRun() = %+v, want cleanup-pending run %d", state, run.ID)
	}
}

func TestSupersedeScheduledRunKeepsPointerWhenRemovalFails(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)
	archivePath := filepath.Join(dataDir, backupStagingDirName, "1-subdux-backup-test.zip")
	if err := os.MkdirAll(filepath.Join(archivePath, "contents"), 0o700); err != nil {
		t.Fatalf("create non-empty spool directory: %v", err)
	}

	run := model.BackupRun{
		Source:                  backupRunSourceScheduled,
		ArchiveName:             "subdux-backup-test.zip",
		ArchivePath:             archivePath,
		Status:                  StatusPartial,
		DeliveryStatus:          StatusPartial,
		RetentionStatus:         StatusNotAttempted,
		BookkeepingStatus:       StatusOK,
		GlobalBookkeepingStatus: StatusOK,
		StartedAt:               pkg.Now(),
	}
	if err := svc.DB.Create(&run).Error; err != nil {
		t.Fatalf("create backup run: %v", err)
	}

	if err := svc.supersedeScheduledRun(&run, "destination changed"); err == nil {
		t.Fatal("supersedeScheduledRun() succeeded for a non-empty spool directory")
	}
	var stored model.BackupRun
	if err := svc.DB.First(&stored, run.ID).Error; err != nil {
		t.Fatalf("reload backup run: %v", err)
	}
	if stored.Status != StatusPartial || stored.ArchivePath != archivePath {
		t.Fatalf("cleanup-pending run status/path = %q/%q, want partial/%q", stored.Status, stored.ArchivePath, archivePath)
	}
	if stored.GlobalBookkeepingStatus != StatusFailed {
		t.Fatalf("global bookkeeping status = %q, want failed", stored.GlobalBookkeepingStatus)
	}
}

func TestFindResumableScheduledRunSupersedesMissingSpool(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)
	archivePath := filepath.Join(dataDir, backupStagingDirName, "1-subdux-backup-test.zip")
	run := model.BackupRun{
		Source:                  backupRunSourceScheduled,
		ArchiveName:             "subdux-backup-test.zip",
		ArchivePath:             archivePath,
		Status:                  StatusPartial,
		DeliveryStatus:          StatusPartial,
		RetentionStatus:         StatusNotAttempted,
		BookkeepingStatus:       StatusOK,
		GlobalBookkeepingStatus: StatusOK,
		StartedAt:               pkg.Now(),
	}
	if err := svc.DB.Create(&run).Error; err != nil {
		t.Fatalf("create backup run: %v", err)
	}

	state, err := svc.findResumableScheduledRun()
	if err != nil {
		t.Fatalf("findResumableScheduledRun() error = %v", err)
	}
	if state != nil {
		t.Fatal("findResumableScheduledRun() returned a run with a missing spool")
	}
	var stored model.BackupRun
	if err := svc.DB.First(&stored, run.ID).Error; err != nil {
		t.Fatalf("reload backup run: %v", err)
	}
	if stored.Status != StatusSuperseded || stored.ArchivePath != "" {
		t.Fatalf("superseded run status/path = %q/%q, want superseded/empty", stored.Status, stored.ArchivePath)
	}
}

func TestLoadBackupDestinationForRunRejectsStaleRevision(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destinationID := createLocalDestination(t, svc, "", 7)
	var destination model.BackupDestination
	if err := svc.DB.First(&destination, destinationID).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}

	_, err := svc.loadBackupDestinationForRun(model.BackupRunDestination{
		DestinationID:       destination.ID,
		DestinationRevision: destination.Revision - 1,
	})
	if !errors.Is(err, ErrBackupDestinationChanged) {
		t.Fatalf("loadBackupDestinationForRun() error = %v, want ErrBackupDestinationChanged", err)
	}
}

func TestRunScheduledBackupWritesStatus(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	createLocalDestination(t, svc, "", 7)

	enabled := true
	timeOfDay := "03:00"
	if err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
	}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}
	fixedNow := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	ownerID := serviceutil.NewBackgroundTaskOwnerID()
	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	lastStatus, err := backupsettings.GetString(nil, svc.DB, KeyLastStatus, "")
	if err != nil || lastStatus != StatusOK {
		t.Fatalf("backup last_status = %q, err = %v, want %q/nil", lastStatus, err, StatusOK)
	}
	lastError, err := backupsettings.GetString(nil, svc.DB, KeyLastError, "")
	if err != nil || lastError != "" {
		t.Fatalf("backup last_error = %q, err = %v, want empty/nil", lastError, err)
	}
	lastRunAt, err := backupsettings.GetString(nil, svc.DB, KeyLastRunAt, "")
	if err != nil || lastRunAt == "" {
		t.Fatalf("backup last_run_at = %q, err = %v, want set/nil", lastRunAt, err)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("second RunScheduledBackup() error = %v", err)
	}
	_, items, err := svc.ListLocalBackups()
	if err != nil {
		t.Fatalf("ListLocalBackups() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 backup after two same-day runs, got %d", len(items))
	}
}

func TestRunScheduledBackupFailureDoesNotBlockSameDayRetry(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	enabled := true
	timeOfDay := "03:00"

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}
	badDir := filepath.Join(blockingFile, "backups")
	destID := createLocalDestination(t, svc, badDir, 7)
	if err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
	}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	fixedNow := time.Date(2026, 6, 15, 4, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	ownerID := serviceutil.NewBackgroundTaskOwnerID()

	if err := svc.RunScheduledBackup(ownerID); err == nil {
		t.Fatal("expected RunScheduledBackup() to fail with an unwritable directory")
	}

	lastStatus, err := backupsettings.GetString(nil, svc.DB, KeyLastStatus, "")
	if err != nil || lastStatus != StatusFailed {
		t.Fatalf("backup last_status = %q, err = %v, want %q/nil", lastStatus, err, StatusFailed)
	}
	lastError, err := backupsettings.GetString(nil, svc.DB, KeyLastError, "")
	if err != nil || lastError == "" {
		t.Fatalf("backup last_error = %q, err = %v, want non-empty/nil", lastError, err)
	}
	lastRunAt, err := backupsettings.GetString(nil, svc.DB, KeyLastRunAt, "")
	if err != nil || lastRunAt != "" {
		t.Fatalf("backup last_run_at = %q, err = %v, want empty/nil", lastRunAt, err)
	}

	// The per-destination row must also carry the failure detail.
	var failed model.BackupDestination
	if err := svc.DB.First(&failed, destID).Error; err != nil {
		t.Fatalf("load destination: %v", err)
	}
	if failed.LastStatus != StatusFailed || failed.LastError == "" {
		t.Fatalf("destination status = %q err = %q, want failed/non-empty", failed.LastStatus, failed.LastError)
	}
	if failed.LastRunAt != nil {
		t.Fatalf("destination last_run_at = %v, want nil after a failed delivery", failed.LastRunAt)
	}

	goodDir := filepath.Join(t.TempDir(), "backups")
	goodConfig := fmt.Sprintf(`{"dir":%q,"retention_count":7}`, goodDir)
	var currentDestination model.BackupDestination
	if err := svc.DB.First(&currentDestination, destID).Error; err != nil {
		t.Fatalf("load destination before retry: %v", err)
	}
	if _, err := svc.UpdateDestination(destID, UpdateDestinationInput{Revision: currentDestination.Revision, Config: &goodConfig}); err != nil {
		t.Fatalf("UpdateDestination() error = %v", err)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("retry RunScheduledBackup() error = %v", err)
	}

	lastStatus, err = backupsettings.GetString(nil, svc.DB, KeyLastStatus, "")
	if err != nil || lastStatus != StatusOK {
		t.Fatalf("backup last_status = %q, err = %v, want %q/nil after retry", lastStatus, err, StatusOK)
	}
	lastError, err = backupsettings.GetString(nil, svc.DB, KeyLastError, "")
	if err != nil || lastError != "" {
		t.Fatalf("backup last_error = %q, err = %v, want empty/nil after retry", lastError, err)
	}
	lastRunAt, err = backupsettings.GetString(nil, svc.DB, KeyLastRunAt, "")
	if err != nil || lastRunAt == "" {
		t.Fatalf("backup last_run_at = %q, err = %v, want set/nil after retry", lastRunAt, err)
	}

	_, items, err := svc.ListLocalBackups()
	if err != nil {
		t.Fatalf("ListLocalBackups() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected exactly 1 backup after a failed run and a same-day retry, got %d", len(items))
	}
}

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestBackupDBPasswordProducesEncryptedZip(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	const password = "download-secret"

	tests := []struct {
		name          string
		includeAssets bool
	}{
		{name: "db only", includeAssets: false},
		{name: "with assets", includeAssets: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, err := svc.BackupDB(tc.includeAssets, password)
			if err != nil {
				t.Fatalf("BackupDB() error = %v", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

			if filepath.Ext(path) != ".zip" {
				t.Fatalf("BackupDB() path = %q, want a .zip archive", path)
			}
			if !backupArchiveIsEncrypted(path) {
				t.Fatal("BackupDB() with password produced an unencrypted archive")
			}

			reader, err := yekazip.OpenReader(path)
			if err != nil {
				t.Fatalf("yekazip.OpenReader() error = %v", err)
			}
			defer reader.Close()
			foundDB := false
			for _, entry := range reader.File {
				if entry.Name != "subdux.db" {
					continue
				}
				foundDB = true
				if !entry.IsEncrypted() {
					t.Fatal("subdux.db entry should be encrypted")
				}
				entry.SetPassword(password)
				rc, openErr := entry.Open()
				if openErr != nil {
					t.Fatalf("open encrypted db entry: %v", openErr)
				}
				header := make([]byte, len(sqliteFileHeader))
				if _, readErr := io.ReadFull(rc, header); readErr != nil {
					rc.Close()
					t.Fatalf("read decrypted db header: %v", readErr)
				}
				rc.Close()
				if string(header) != string(sqliteFileHeader) {
					t.Fatalf("decrypted db header = %q, want SQLite header", header)
				}
			}
			if !foundDB {
				t.Fatal("encrypted archive missing subdux.db entry")
			}
		})
	}
}

func TestBackupDBWhitespacePasswordKeepsPlainBehavior(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	path, err := svc.BackupDB(false, "   ")
	if err != nil {
		t.Fatalf("BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

	if filepath.Ext(path) != ".db" {
		t.Fatalf("BackupDB() path = %q, want a raw .db file", path)
	}
	if !isSQLiteFile(t, path) {
		t.Fatal("BackupDB() db-only path is not a SQLite file")
	}
}

func TestBackupDBUsesPrivateTempDirectoryAnd0600Files(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	tests := []struct {
		name          string
		includeAssets bool
	}{
		{name: "database", includeAssets: false},
		{name: "archive", includeAssets: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, err := svc.BackupDB(tc.includeAssets, "")
			if err != nil {
				t.Fatalf("BackupDB() error = %v", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

			dirInfo, err := os.Stat(filepath.Dir(path))
			if err != nil {
				t.Fatalf("stat private temp directory: %v", err)
			}
			if got := dirInfo.Mode().Perm(); got != 0o700 {
				t.Fatalf("private temp directory permissions = %04o, want 0700", got)
			}

			fileInfo, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat backup file: %v", err)
			}
			if got := fileInfo.Mode().Perm(); got != 0o600 {
				t.Fatalf("backup file permissions = %04o, want 0600", got)
			}
		})
	}
}

func TestBuildBackupArchiveUsesPrivateTempDirectoryAnd0600Files(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	archiveName, err := newBackupArchiveName()
	if err != nil {
		t.Fatalf("newBackupArchiveName() error = %v", err)
	}
	archivePath, cleanup, err := svc.buildBackupArchiveNamed(runtimeConfig{}, archiveName)
	if err != nil {
		t.Fatalf("buildBackupArchiveNamed() error = %v", err)
	}
	t.Cleanup(cleanup)

	if filepath.Base(archivePath) != archiveName {
		t.Fatalf("archive base name = %q, want %q", filepath.Base(archivePath), archiveName)
	}

	dir := filepath.Dir(archivePath)
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat private temp directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("private temp directory permissions = %04o, want 0700", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read private temp directory: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("private temp directory contains %d files, want database snapshot and archive", len(entries))
	}
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", entry.Name(), err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("%s permissions = %04o, want 0600", entry.Name(), got)
		}
	}
}

func TestBackupDBPlainZipRemainsUnencrypted(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	path, err := svc.BackupDB(true, "")
	if err != nil {
		t.Fatalf("BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

	if filepath.Ext(path) != ".zip" {
		t.Fatalf("BackupDB() path = %q, want a .zip archive", path)
	}
	if backupArchiveIsEncrypted(path) {
		t.Fatal("plain include-assets backup should not be encrypted")
	}
}

func isSQLiteFile(t *testing.T, path string) bool {
	t.Helper()
	f, err := os.Open(path) // #nosec G304 -- path is an internally-created temp backup file.
	if err != nil {
		t.Fatalf("open %q: %v", path, err)
	}
	defer f.Close()
	header := make([]byte, len(sqliteFileHeader))
	if _, err := io.ReadFull(f, header); err != nil {
		return false
	}
	return string(header) == string(sqliteFileHeader)
}
