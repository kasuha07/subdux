package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	yekazip "github.com/yeka/zip"
)

// onlyBackupArchiveName asserts dir holds exactly one delivered archive and
// returns its name.
func onlyBackupArchiveName(t *testing.T, dir string) string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read backup directory %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && backupFileNamePattern.MatchString(entry.Name()) {
			names = append(names, entry.Name())
		}
	}
	if len(names) != 1 {
		t.Fatalf("archives in %s = %v, want exactly one", dir, names)
	}
	return names[0]
}

// runDestinationIDs returns the destination ids staged on a run, which is the
// durable record of how the scheduler grouped that pass.
func runDestinationIDs(t *testing.T, svc *Service, runID uint) []uint {
	t.Helper()

	var runDestinations []model.BackupRunDestination
	if err := svc.DB.Where("run_id = ?", runID).Order("destination_id ASC").Find(&runDestinations).Error; err != nil {
		t.Fatalf("load run destinations for run %d: %v", runID, err)
	}
	ids := make([]uint, 0, len(runDestinations))
	for _, runDestination := range runDestinations {
		ids = append(ids, runDestination.DestinationID)
	}
	return ids
}

// TestRunScheduledBackupGroupsDestinationsSharingAnArchiveSpec is the core of
// the 3-2-1 case: destinations that come due together and agree on the archive
// contents must cost one database snapshot between them, while a destination
// whose archive differs pays for its own. One run means exactly one VACUUM INTO
// and one zip, so the run rows plus the delivered file names are the evidence.
func TestRunScheduledBackupGroupsDestinationsSharingAnArchiveSpec(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	sharedFirstDir := filepath.Join(t.TempDir(), "shared-first")
	sharedSecondDir := filepath.Join(t.TempDir(), "shared-second")
	ownDir := filepath.Join(t.TempDir(), "own")

	sharedFirstID := createLocalPlan(t, svc, localPlan{dir: sharedFirstDir, retention: 7, timeOfDay: "03:00"})
	sharedSecondID := createLocalPlan(t, svc, localPlan{dir: sharedSecondDir, retention: 7, timeOfDay: "03:00"})
	// Same firing time, different archive bytes: an encrypted archive cannot be
	// the same file as a plain one, so this plan cannot join the group above.
	ownID := createLocalPlan(t, svc, localPlan{dir: ownDir, retention: 7, timeOfDay: "03:00", password: "own-secret"})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	var runs []model.BackupRun
	if err := svc.DB.Order("id ASC").Find(&runs).Error; err != nil {
		t.Fatalf("load backup runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("backup run count = %d, want 2 (one archive per spec)", len(runs))
	}

	// Two runs alone would also be produced by one run per destination, so the
	// membership of each run is asserted rather than just the count.
	var sharedRun, ownRun model.BackupRun
	for _, run := range runs {
		switch ids := runDestinationIDs(t, svc, run.ID); {
		case len(ids) == 2 && ids[0] == sharedFirstID && ids[1] == sharedSecondID:
			sharedRun = run
		case len(ids) == 1 && ids[0] == ownID:
			ownRun = run
		default:
			t.Fatalf("run %d groups destinations %v, want either the shared pair or the encrypting plan alone", run.ID, ids)
		}
	}
	if sharedRun.ID == 0 || ownRun.ID == 0 {
		t.Fatalf("runs = %+v, want one shared run and one solo run", runs)
	}

	// Both members of the group hold the identical archive name, which is the
	// observable consequence of the single snapshot the shared run took.
	sharedName := onlyBackupArchiveName(t, sharedFirstDir)
	if got := onlyBackupArchiveName(t, sharedSecondDir); got != sharedName {
		t.Fatalf("grouped destinations hold %q and %q, want one shared archive", sharedName, got)
	}
	if sharedName != sharedRun.ArchiveName {
		t.Fatalf("delivered archive = %q, want the shared run's archive %q", sharedName, sharedRun.ArchiveName)
	}
	ownName := onlyBackupArchiveName(t, ownDir)
	if ownName != ownRun.ArchiveName || ownName == sharedName {
		t.Fatalf("solo archive = %q (run archive %q, shared archive %q), want its own", ownName, ownRun.ArchiveName, sharedName)
	}

	for _, id := range []uint{sharedFirstID, sharedSecondID, ownID} {
		if loadDestination(t, svc, id).LastScheduledRunAt == nil {
			t.Fatalf("destination %d last_scheduled_run_at is nil after a successful pass", id)
		}
	}
}

// TestRunScheduledBackupSplitsGroupsByIncludeAssets is the other half of the
// grouping key: archive contents differ even without encryption, so an
// include-assets plan cannot ride along on a database-only archive.
func TestRunScheduledBackupSplitsGroupsByIncludeAssets(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	plainDir := filepath.Join(t.TempDir(), "plain")
	assetsDir := filepath.Join(t.TempDir(), "assets")
	createLocalPlan(t, svc, localPlan{dir: plainDir, retention: 7, timeOfDay: "03:00"})
	createLocalPlan(t, svc, localPlan{dir: assetsDir, retention: 7, timeOfDay: "03:00", includeAssets: true})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	if got := countBackupRuns(t, svc); got != 2 {
		t.Fatalf("backup run count = %d, want 2 (db-only and with-assets cannot share)", got)
	}
	if plain, withAssets := onlyBackupArchiveName(t, plainDir), onlyBackupArchiveName(t, assetsDir); plain == withAssets {
		t.Fatalf("both plans received archive %q, want separate archives", plain)
	}
}

// TestRunScheduledBackupHonoursIndependentTimesOfDay covers the headline of the
// refactor: two destinations are two plans, so they fire on their own ticks and
// neither drags the other along.
func TestRunScheduledBackupHonoursIndependentTimesOfDay(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	earlyDir := filepath.Join(t.TempDir(), "early")
	lateDir := filepath.Join(t.TempDir(), "late")
	earlyID := createLocalPlan(t, svc, localPlan{dir: earlyDir, retention: 7, timeOfDay: "03:00"})
	lateID := createLocalPlan(t, svc, localPlan{dir: lateDir, retention: 7, timeOfDay: "23:00"})

	ownerID := serviceutil.NewBackgroundTaskOwnerID()

	restoreMorning := pkg.SetNowForTest(scheduleTime(4, 0))
	morningErr := svc.RunScheduledBackup(ownerID)
	restoreMorning()
	if morningErr != nil {
		t.Fatalf("morning RunScheduledBackup() error = %v", morningErr)
	}

	if got := countBackupRuns(t, svc); got != 1 {
		t.Fatalf("backup run count after the morning tick = %d, want 1", got)
	}
	if got := countBackupArchives(t, earlyDir); got != 1 {
		t.Fatalf("early destination archive count = %d, want 1", got)
	}
	if got := countBackupArchives(t, lateDir); got != 0 {
		t.Fatalf("late destination archive count = %d, want 0 before its firing time", got)
	}
	early := loadDestination(t, svc, earlyID)
	if early.LastScheduledRunAt == nil {
		t.Fatal("early destination last_scheduled_run_at is nil after its tick")
	}
	if loadDestination(t, svc, lateID).LastScheduledRunAt != nil {
		t.Fatal("late destination consumed a slot on the early destination's tick")
	}
	morningRunAt := *early.LastScheduledRunAt

	t.Cleanup(pkg.SetNowForTest(scheduleTime(23, 30)))
	if err := svc.RunScheduledBackup(ownerID); err != nil {
		t.Fatalf("evening RunScheduledBackup() error = %v", err)
	}

	if got := countBackupRuns(t, svc); got != 2 {
		t.Fatalf("backup run count after the evening tick = %d, want 2", got)
	}
	if got := countBackupArchives(t, lateDir); got != 1 {
		t.Fatalf("late destination archive count = %d, want 1 after its firing time", got)
	}
	// The early plan already ran today, so the later tick must leave it alone
	// rather than take a second archive for it.
	if got := countBackupArchives(t, earlyDir); got != 1 {
		t.Fatalf("early destination archive count after the evening tick = %d, want 1", got)
	}
	if got := *loadDestination(t, svc, earlyID).LastScheduledRunAt; !got.Equal(morningRunAt) {
		t.Fatalf("early destination last_scheduled_run_at = %v, want the morning value %v", got, morningRunAt)
	}
	if loadDestination(t, svc, lateID).LastScheduledRunAt == nil {
		t.Fatal("late destination last_scheduled_run_at is nil after its own tick")
	}
}

// TestRunDestinationBackupDoesNotConsumeTheScheduledSlot separates the two
// timestamps: an ad-hoc run proves the destination works but must not stand in
// for the day's scheduled copy.
func TestRunDestinationBackupDoesNotConsumeTheScheduledSlot(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	dir := filepath.Join(t.TempDir(), "backups")
	id := createLocalPlan(t, svc, localPlan{dir: dir, retention: 7, timeOfDay: "03:00"})

	// 02:00 — before the plan's firing time, so nothing is due yet.
	restoreManual := pkg.SetNowForTest(scheduleTime(2, 0))
	_, manualErr := svc.RunDestinationBackup(context.Background(), id)
	restoreManual()
	if manualErr != nil {
		t.Fatalf("RunDestinationBackup() error = %v", manualErr)
	}

	manual := loadDestination(t, svc, id)
	if manual.LastRunAt == nil {
		t.Fatal("manual run did not advance last_run_at")
	}
	if manual.LastScheduledRunAt != nil {
		t.Fatalf("manual run advanced last_scheduled_run_at to %v, which would swallow the day's scheduled backup", manual.LastScheduledRunAt)
	}

	// Because the manual run never claimed the day, the schedule still fires.
	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}
	if got := countBackupArchives(t, dir); got != 2 {
		t.Fatalf("archive count = %d, want 2 (the manual copy plus the scheduled one)", got)
	}
	if loadDestination(t, svc, id).LastScheduledRunAt == nil {
		t.Fatal("scheduled run did not advance last_scheduled_run_at")
	}
}

// TestRunScheduledBackupEncryptsPerDestination confirms encryption follows the
// destination rather than a global switch: in one pass, one plan produces an
// AES-encrypted archive and the other a plain one.
func TestRunScheduledBackupEncryptsPerDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	plainDir := filepath.Join(t.TempDir(), "plain")
	encryptedDir := filepath.Join(t.TempDir(), "encrypted")
	createLocalPlan(t, svc, localPlan{dir: plainDir, retention: 7, timeOfDay: "03:00"})
	createLocalPlan(t, svc, localPlan{dir: encryptedDir, retention: 7, timeOfDay: "03:00", password: "per-destination-secret"})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	if backupArchiveIsEncrypted(filepath.Join(plainDir, onlyBackupArchiveName(t, plainDir))) {
		t.Fatal("the plain destination received an encrypted archive")
	}
	if !backupArchiveIsEncrypted(filepath.Join(encryptedDir, onlyBackupArchiveName(t, encryptedDir))) {
		t.Fatal("the encrypting destination received a plain archive")
	}
}

// TestRunScheduledBackupLeavesFailedDestinationDue pins the per-destination due
// gate. Only a delivery that actually landed may consume a destination's slot;
// otherwise a failing target would go quiet until the next day, which is exactly
// when an operator most needs it to keep retrying.
func TestRunScheduledBackupLeavesFailedDestinationDue(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	goodDir := filepath.Join(t.TempDir(), "good")
	goodID := createLocalPlan(t, svc, localPlan{dir: goodDir, retention: 7, timeOfDay: "03:00"})
	blockingFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(blockingFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	badID := createLocalPlan(t, svc, localPlan{dir: filepath.Join(blockingFile, "backups"), retention: 7, timeOfDay: "03:00"})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v, want nil for a partial pass", err)
	}

	if loadDestination(t, svc, goodID).LastScheduledRunAt == nil {
		t.Fatal("delivered destination last_scheduled_run_at is nil")
	}
	if scheduled := loadDestination(t, svc, badID).LastScheduledRunAt; scheduled != nil {
		t.Fatalf("failed destination last_scheduled_run_at = %v, want nil", scheduled)
	}

	// Asking with no claims reflects the schedule alone rather than run
	// ownership: the failed plan is still due, the delivered one is not.
	groups, err := svc.dueScheduledDestinations(nil)
	if err != nil {
		t.Fatalf("dueScheduledDestinations() error = %v", err)
	}
	if len(groups) != 1 || len(groups[0].destinations) != 1 || groups[0].destinations[0].ID != badID {
		t.Fatalf("due destinations = %+v, want only the failed destination %d", groups, badID)
	}
}

// TestRunScheduledBackupReportsUnreadableScheduleWithoutBlockingOthers covers
// the fail-loud contract for a destination whose stored plan cannot be read: it
// is configured to be backed up and is not being backed up, so the operator must
// see it — but it must not take the rest of the pass down with it.
func TestRunScheduledBackupReportsUnreadableScheduleWithoutBlockingOthers(t *testing.T) {
	tests := []struct {
		name         string
		brokenConfig string
		wantSentinel error
	}{
		{
			name:         "invalid time of day",
			brokenConfig: `{"dir":"backups","time_of_day":"nope"}`,
			wantSentinel: ErrInvalidBackupTimeOfDay,
		},
		{
			name:         "unparseable config",
			brokenConfig: `{"dir":`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newBackupTestDB(t)

			healthyDir := filepath.Join(t.TempDir(), "healthy")
			healthyID := createLocalPlan(t, svc, localPlan{dir: healthyDir, retention: 7, timeOfDay: "03:00"})
			brokenID := createLocalPlan(t, svc, localPlan{dir: filepath.Join(t.TempDir(), "broken"), retention: 7, timeOfDay: "03:00"})

			// Stored configs without the encryption envelope are read back verbatim,
			// so writing plaintext here reproduces a row hand-edited or left behind
			// by an older schema.
			if err := svc.DB.Model(&model.BackupDestination{}).
				Where("id = ?", brokenID).
				Update("config", tc.brokenConfig).Error; err != nil {
				t.Fatalf("corrupt destination config: %v", err)
			}

			t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
			err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID())
			if err == nil {
				t.Fatal("RunScheduledBackup() error = nil, want the unreadable destination reported")
			}
			if tc.wantSentinel != nil && !errors.Is(err, tc.wantSentinel) {
				t.Fatalf("RunScheduledBackup() error = %v, want %v", err, tc.wantSentinel)
			}
			// The message must name the destination, because "some schedule is
			// invalid" is not actionable when several are configured.
			if want := fmt.Sprintf("backup destination %d schedule", brokenID); !strings.Contains(err.Error(), want) {
				t.Fatalf("RunScheduledBackup() error = %v, want it to name %q", err, want)
			}

			if got := countBackupArchives(t, healthyDir); got != 1 {
				t.Fatalf("healthy destination archive count = %d, want 1", got)
			}
			if loadDestination(t, svc, healthyID).LastScheduledRunAt == nil {
				t.Fatal("healthy destination last_scheduled_run_at is nil, want its run to have completed")
			}
		})
	}
}

// TestScheduledArchiveOpensWithItsDestinationPassword goes past "the archive is
// encrypted" to "it is encrypted with this destination's own password". Moving
// the password onto the destination is only meaningful if the bytes delivered
// there are actually openable with it.
func TestScheduledArchiveOpensWithItsDestinationPassword(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	const password = "per-destination-secret"
	encryptedDir := filepath.Join(t.TempDir(), "encrypted")
	createLocalPlan(t, svc, localPlan{dir: encryptedDir, retention: 7, timeOfDay: "03:00", password: password})

	t.Cleanup(pkg.SetNowForTest(scheduleTime(4, 0)))
	if err := svc.RunScheduledBackup(serviceutil.NewBackgroundTaskOwnerID()); err != nil {
		t.Fatalf("RunScheduledBackup() error = %v", err)
	}

	archivePath := filepath.Join(encryptedDir, onlyBackupArchiveName(t, encryptedDir))
	if !backupArchiveIsEncrypted(archivePath) {
		t.Fatal("the encrypting destination received a plain archive")
	}

	reader, err := yekazip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("yekazip.OpenReader() error = %v", err)
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.Name != "subdux.db" {
			continue
		}
		if !entry.IsEncrypted() {
			t.Fatal("subdux.db entry is not encrypted")
		}
		entry.SetPassword(password)
		rc, openErr := entry.Open()
		if openErr != nil {
			t.Fatalf("open encrypted db entry: %v", openErr)
		}
		header := make([]byte, len(sqliteFileHeader))
		_, readErr := io.ReadFull(rc, header)
		rc.Close()
		if readErr != nil {
			t.Fatalf("read decrypted db header: %v", readErr)
		}
		if string(header) != string(sqliteFileHeader) {
			t.Fatalf("decrypted db header = %q, want the SQLite header", header)
		}
		return
	}
	t.Fatal("archive is missing its subdux.db entry")
}
