package backup

import (
	"archive/zip"
	"context"
	"encoding/json"
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
	yekazip "github.com/yeka/zip"
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

// localPlan describes a local destination's whole backup plan. The schedule now
// lives in the same config blob as the transport settings, so tests build both
// halves in one place. The zero value is a plain, unencrypted destination that
// fires at the default time of day.
type localPlan struct {
	dir           string
	retention     int
	timeOfDay     string
	includeAssets bool
	password      string // a non-empty password also turns encryption on
	disabled      bool
}

func (p localPlan) configJSON(t *testing.T) string {
	t.Helper()

	// Fields are omitted rather than zero-filled so each test exercises the same
	// defaulting the admin UI relies on when it leaves a plan field untouched.
	config := map[string]any{"dir": p.dir}
	if p.retention > 0 {
		config["retention_count"] = p.retention
	}
	if p.timeOfDay != "" {
		config["time_of_day"] = p.timeOfDay
	}
	if p.includeAssets {
		config["include_assets"] = true
	}
	if p.password != "" {
		config["encrypt_enabled"] = true
		config["encryption_password"] = p.password
	}

	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("marshal local destination config: %v", err)
	}
	return string(encoded)
}

// createLocalPlan creates a local backup destination from plan and returns its
// ID. An empty dir means the default <DATA_PATH>/backups.
func createLocalPlan(t *testing.T, svc *Service, plan localPlan) uint {
	t.Helper()

	destination, err := svc.CreateDestination(CreateDestinationInput{
		Type:    "local",
		Enabled: !plan.disabled,
		Config:  plan.configJSON(t),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}
	return destination.ID
}

// createLocalDestination creates and enables a local backup destination with
// the given directory and retention count, returning its ID.
func createLocalDestination(t *testing.T, svc *Service, dir string, retention int) uint {
	t.Helper()

	return createLocalPlan(t, svc, localPlan{dir: dir, retention: retention})
}

// loadDestination reloads a persisted destination so tests can assert on the
// stored plan and on the two independent run timestamps.
func loadDestination(t *testing.T, svc *Service, id uint) model.BackupDestination {
	t.Helper()

	var destination model.BackupDestination
	if err := svc.DB.First(&destination, id).Error; err != nil {
		t.Fatalf("load destination %d: %v", id, err)
	}
	return destination
}

// loadDestinations returns the rows for ids in the order given so a test can
// drive runBackup's fan-out directly, the way the scheduler does once it has
// grouped destinations onto one shared archive.
func loadDestinations(t *testing.T, svc *Service, ids ...uint) []model.BackupDestination {
	t.Helper()

	destinations := make([]model.BackupDestination, 0, len(ids))
	for _, id := range ids {
		destinations = append(destinations, loadDestination(t, svc, id))
	}
	return destinations
}

// scheduleTime builds an instant on a fixed test day in the timezone the
// scheduler evaluates plans in. Times of day are compared in that zone, so a
// literal UTC instant would make every due-check test depend on where the suite
// happens to run.
func scheduleTime(hour, minute int) time.Time {
	return time.Date(2026, 6, 15, hour, minute, 0, 0, pkg.GetSystemTimezone())
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

// TestDestinationScheduleFromStoredReadsPersistedPlan is the per-destination
// successor to the old global-settings round trip: the plan an admin saves must
// come back to the runtime exactly as written, including the archive password
// that the masked DestinationView deliberately blanks.
func TestDestinationScheduleFromStoredReadsPersistedPlan(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	id := createLocalPlan(t, svc, localPlan{
		retention:     7,
		timeOfDay:     "02:30",
		includeAssets: true,
		password:      "s3cr3t-backup",
	})

	schedule, err := destinationScheduleFromStored(loadDestination(t, svc, id).Config)
	if err != nil {
		t.Fatalf("destinationScheduleFromStored() error = %v", err)
	}
	if schedule.TimeOfDay != "02:30" || !schedule.IncludeAssets {
		t.Fatalf("stored schedule = %+v, want 02:30 with assets", schedule)
	}
	if !schedule.EncryptEnabled || schedule.EncryptPassword != "s3cr3t-backup" {
		t.Fatalf("stored encryption = %v/%q, want enabled with the saved password", schedule.EncryptEnabled, schedule.EncryptPassword)
	}
}

// TestDestinationScheduleDefaultsWhenPlanFieldsAreOmitted covers the other half
// of the old settings round trip: a destination saved without plan fields is
// still a complete plan rather than one that never fires.
func TestDestinationScheduleDefaultsWhenPlanFieldsAreOmitted(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	id := createLocalDestination(t, svc, "", 7)

	schedule, err := destinationScheduleFromStored(loadDestination(t, svc, id).Config)
	if err != nil {
		t.Fatalf("destinationScheduleFromStored() error = %v", err)
	}
	if schedule.TimeOfDay != defaultBackupTimeOfDay {
		t.Fatalf("time of day = %q, want default %q", schedule.TimeOfDay, defaultBackupTimeOfDay)
	}
	if schedule.IncludeAssets || schedule.EncryptEnabled || schedule.EncryptPassword != "" {
		t.Fatalf("schedule = %+v, want a plain db-only plan", schedule)
	}
}

// TestRunScheduledBackupWithoutDestinationsIsANoOp replaces the old "enabling
// the schedule without a destination is an error" guard. There is no global
// schedule left to enable, so a pass with nothing configured must simply do
// nothing rather than fail the background task.
func TestRunScheduledBackupWithoutDestinationsIsANoOp(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v, want nil with no destinations", err)
	}
	if got := countBackupRuns(t, svc); got != 0 {
		t.Fatalf("backup run count = %d, want 0", got)
	}
}

func TestRunDestinationBackupAndList(t *testing.T) {
	svc, dataDir := newBackupTestDB(t)

	t.Cleanup(pkg.SetNowForTest(scheduleTime(3, 0)))

	wantDir := filepath.Join(dataDir, "backups")
	destID := createLocalDestination(t, svc, "", 7)

	result, err := svc.RunDestinationBackup(context.Background(), destID)
	if err != nil {
		t.Fatalf("RunDestinationBackup() error = %v", err)
	}
	if len(result.Results) != 1 || !result.Results[0].Success {
		t.Fatalf("RunDestinationBackup() results = %+v, want one successful delivery", result.Results)
	}
	if result.Results[0].DestinationID != destID {
		t.Fatalf("RunDestinationBackup() destination id = %d, want %d", result.Results[0].DestinationID, destID)
	}

	if _, err := os.Stat(filepath.Join(wantDir, result.ArchiveName)); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	// The manual run must land in the unified backup history with its archive
	// metadata and the destination it delivered to.
	records, err := svc.ListBackupRuns(0)
	if err != nil {
		t.Fatalf("ListBackupRuns() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListBackupRuns() records = %d, want 1", len(records))
	}
	record := records[0]
	if record.Source != backupRunSourceManual || record.Status != StatusOK {
		t.Fatalf("recorded run source/status = %q/%q, want manual/success", record.Source, record.Status)
	}
	if record.ArchiveName != result.ArchiveName {
		t.Fatalf("recorded name = %q, want %q", record.ArchiveName, result.ArchiveName)
	}
	if record.SizeBytes <= 0 {
		t.Fatalf("recorded size = %d, want > 0", record.SizeBytes)
	}
	if record.Encrypted {
		t.Fatal("plain backup should not be reported as encrypted")
	}
	if len(record.Destinations) != 1 || record.Destinations[0].DestinationID != destID || record.Destinations[0].DeliveryStatus != StatusOK {
		t.Fatalf("recorded destinations = %+v, want one successful delivery to %d", record.Destinations, destID)
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

// TestRunBackupWithoutDestinationFails confirms a run with no destination is a
// hard error rather than a silent no-op that would create an empty run row and
// an archive nobody receives.
func TestRunBackupWithoutDestinationFails(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	_, err := svc.runBackup(context.Background(), backupRunSourceManual, nil, archiveSpec{}, nil)
	if !errors.Is(err, ErrNoEnabledBackupDestination) {
		t.Fatalf("runBackup() error = %v, want ErrNoEnabledBackupDestination", err)
	}
	if got := countBackupRuns(t, svc); got != 0 {
		t.Fatalf("backup run count = %d, want 0", got)
	}
}

// TestRunDestinationBackupRejectsMissingDestination pins the manual-run lookup:
// an unknown id is a not-found error, not a run over zero destinations.
func TestRunDestinationBackupRejectsMissingDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	if _, err := svc.RunDestinationBackup(context.Background(), 4242); !errors.Is(err, ErrBackupDestinationNotFound) {
		t.Fatalf("RunDestinationBackup() error = %v, want ErrBackupDestinationNotFound", err)
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
	states, err := svc.findResumableScheduledRuns()
	if err != nil {
		t.Fatalf("findResumableScheduledRuns() error = %v", err)
	}
	if len(states) != 1 || states[0].run.ID != run.ID {
		t.Fatalf("findResumableScheduledRuns() = %+v, want only cleanup-pending run %d", states, run.ID)
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

func TestFindResumableScheduledRunsSupersedesMissingSpool(t *testing.T) {
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

	states, err := svc.findResumableScheduledRuns()
	if err != nil {
		t.Fatalf("findResumableScheduledRuns() error = %v", err)
	}
	if len(states) != 0 {
		t.Fatal("findResumableScheduledRuns() returned a run with a missing spool")
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
	destID := createLocalPlan(t, svc, localPlan{retention: 7, timeOfDay: "03:00"})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))

	ownerID := serviceutil.NewBackgroundTaskOwnerID()
	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	// The run outcome now lives on the destination itself. LastScheduledRunAt is
	// what the due check reads, so it is the row that stops a same-day repeat.
	destination := loadDestination(t, svc, destID)
	if destination.LastStatus != StatusOK || destination.LastError != "" {
		t.Fatalf("destination status = %q err = %q, want success/empty", destination.LastStatus, destination.LastError)
	}
	if destination.LastRunAt == nil || destination.LastScheduledRunAt == nil {
		t.Fatalf("destination last_run_at = %v last_scheduled_run_at = %v, want both set", destination.LastRunAt, destination.LastScheduledRunAt)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("second RunScheduledBackup() error = %v", err)
	}
	objects, err := svc.ListDestinationBackups(context.Background(), destID)
	if err != nil {
		t.Fatalf("ListDestinationBackups() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected exactly 1 backup after two same-day runs, got %d", len(objects))
	}
}

func TestRunScheduledBackupFailureDoesNotBlockSameDayRetry(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}
	badDir := filepath.Join(blockingFile, "backups")
	destID := createLocalPlan(t, svc, localPlan{dir: badDir, retention: 7, timeOfDay: "03:00"})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))

	ownerID := serviceutil.NewBackgroundTaskOwnerID()

	if err := svc.RunScheduledBackup(ownerID); err == nil {
		t.Fatal("expected RunScheduledBackup() to fail with an unwritable directory")
	}

	failed := loadDestination(t, svc, destID)
	if failed.LastStatus != StatusFailed || failed.LastError == "" {
		t.Fatalf("destination status = %q err = %q, want failed/non-empty", failed.LastStatus, failed.LastError)
	}
	if failed.LastRunAt != nil {
		t.Fatalf("destination last_run_at = %v, want nil after a failed delivery", failed.LastRunAt)
	}
	// A failed delivery must not consume the destination's schedule slot, which
	// is exactly what leaves it due for the same-day retry below.
	if failed.LastScheduledRunAt != nil {
		t.Fatalf("destination last_scheduled_run_at = %v, want nil after a failed delivery", failed.LastScheduledRunAt)
	}

	goodDir := filepath.Join(t.TempDir(), "backups")
	goodConfig := localPlan{dir: goodDir, retention: 7, timeOfDay: "03:00"}.configJSON(t)
	if _, err := svc.UpdateDestination(destID, UpdateDestinationInput{Revision: failed.Revision, Config: &goodConfig}); err != nil {
		t.Fatalf("UpdateDestination() error = %v", err)
	}

	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("retry RunScheduledBackup() error = %v", err)
	}

	repaired := loadDestination(t, svc, destID)
	if repaired.LastStatus != StatusOK || repaired.LastError != "" {
		t.Fatalf("destination status = %q err = %q after retry, want success/empty", repaired.LastStatus, repaired.LastError)
	}
	if repaired.LastScheduledRunAt == nil {
		t.Fatal("destination last_scheduled_run_at is nil after a successful retry")
	}

	objects, err := svc.ListDestinationBackups(context.Background(), destID)
	if err != nil {
		t.Fatalf("ListDestinationBackups() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected exactly 1 backup after a failed run and a same-day retry, got %d", len(objects))
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
	archivePath, cleanup, err := svc.buildBackupArchiveNamed(archiveSpec{}, archiveName)
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
