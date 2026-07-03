package backup

import (
	"archive/zip"
	"errors"
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
	svc, _ := newBackupTestDB(t)
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

	if err := svc.applyLocalBackupRetention(dir, 2); err != nil {
		t.Fatalf("applyLocalBackupRetention() error = %v", err)
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

	enabled := true
	timeOfDay := "02:30"
	includeAssets := true
	retention := int64(14)
	localDir := filepath.Join(t.TempDir(), "backups")

	if err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
		IncludeAssets:   &includeAssets,
		RetentionCount:  &retention,
		LocalDir:        &localDir,
	}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
	}

	gotEnabled, err := backupsettings.GetBool(nil, svc.DB, KeyScheduleEnabled, false)
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
	gotRetention, err := backupsettings.GetInt(nil, svc.DB, KeyRetentionCount, 0)
	if err != nil || gotRetention != int(retention) {
		t.Fatalf("backup retention = %d, err = %v, want %d/nil", gotRetention, err, retention)
	}
	gotLocalDir, err := backupsettings.GetString(nil, svc.DB, KeyLocalDir, "")
	if err != nil || gotLocalDir != filepath.Clean(localDir) {
		t.Fatalf("backup local_dir = %q, err = %v, want %q/nil", gotLocalDir, err, filepath.Clean(localDir))
	}
	gotPassword, err := backupsettings.GetString(nil, svc.DB, KeyEncryptionPassword, "")
	if err != nil || gotPassword != "" {
		t.Fatalf("backup password = %q, err = %v, want empty/nil", gotPassword, err)
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
			name:    "retention too low",
			input:   UpdateSettingsInput{RetentionCount: int64Ptr(0)},
			wantErr: ErrInvalidBackupRetentionCount,
		},
		{
			name:    "retention too high",
			input:   UpdateSettingsInput{RetentionCount: int64Ptr(1001)},
			wantErr: ErrInvalidBackupRetentionCount,
		},
		{
			name:    "relative dir with parent segment",
			input:   UpdateSettingsInput{LocalDir: strPtr("../evil")},
			wantErr: ErrInvalidBackupLocalDir,
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

func TestCreateLocalBackupAndList(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)

	fixedNow := time.Date(2026, 6, 15, 3, 0, 0, 0, time.UTC)
	restore := pkg.SetNowForTest(fixedNow)
	t.Cleanup(restore)

	path, err := svc.CreateLocalBackup()
	if err != nil {
		t.Fatalf("CreateLocalBackup() error = %v", err)
	}

	wantDir := filepath.Join(dataDir, "backups")
	if filepath.Dir(path) != wantDir {
		t.Fatalf("backup dir = %q, want %q", filepath.Dir(path), wantDir)
	}
	if _, err := os.Stat(path); err != nil {
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
	if items[0].Name != filepath.Base(path) {
		t.Fatalf("listed name = %q, want %q", items[0].Name, filepath.Base(path))
	}
	if items[0].Encrypted {
		t.Fatal("plain backup should not be reported as encrypted")
	}
}

func TestRunScheduledBackupWritesStatus(t *testing.T) {
	svc, _ := newBackupTestDB(t)

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
	if err := applySettings(t, svc, UpdateSettingsInput{
		ScheduleEnabled: &enabled,
		TimeOfDay:       &timeOfDay,
		LocalDir:        &badDir,
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

	goodDir := filepath.Join(t.TempDir(), "backups")
	if err := applySettings(t, svc, UpdateSettingsInput{LocalDir: &goodDir}); err != nil {
		t.Fatalf("ApplySettings() error = %v", err)
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
func int64Ptr(i int64) *int64 { return &i }

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
			t.Cleanup(func() { _ = os.Remove(path) })

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
	t.Cleanup(func() { _ = os.Remove(path) })

	if filepath.Ext(path) != ".db" {
		t.Fatalf("BackupDB() path = %q, want a raw .db file", path)
	}
	if !isSQLiteFile(t, path) {
		t.Fatal("BackupDB() db-only path is not a SQLite file")
	}
}

func TestBackupDBPlainZipRemainsUnencrypted(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	path, err := svc.BackupDB(true, "")
	if err != nil {
		t.Fatalf("BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

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
