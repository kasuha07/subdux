package backup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
)

// TestBackupDBRecordsDownloadRun pins the requirement that a download is a
// backup like any other: it must land in the run history with its own source,
// archive metadata, and no destination rows.
func TestBackupDBRecordsDownloadRun(t *testing.T) {
	tests := []struct {
		name          string
		includeAssets bool
		password      string
		wantExt       string
		wantEncrypted bool
	}{
		{name: "plain db", wantExt: ".db"},
		{name: "with assets", includeAssets: true, wantExt: ".zip"},
		{name: "encrypted", password: "download-secret", wantExt: ".zip", wantEncrypted: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newBackupTestDB(t)

			path, err := svc.BackupDB(tc.includeAssets, tc.password)
			if err != nil {
				t.Fatalf("BackupDB() error = %v", err)
			}
			t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

			records, listErr := svc.ListBackupRuns(0)
			if listErr != nil {
				t.Fatalf("ListBackupRuns() error = %v", listErr)
			}
			if len(records) != 1 {
				t.Fatalf("ListBackupRuns() records = %d, want 1", len(records))
			}
			record := records[0]
			if record.Source != backupRunSourceDownload {
				t.Fatalf("run source = %q, want %q", record.Source, backupRunSourceDownload)
			}
			if record.Status != StatusOK {
				t.Fatalf("run status = %q, want success", record.Status)
			}
			if record.ArchiveName != filepath.Base(path) {
				t.Fatalf("recorded name = %q, want served file %q", record.ArchiveName, filepath.Base(path))
			}
			if !strings.HasSuffix(record.ArchiveName, tc.wantExt) {
				t.Fatalf("recorded name = %q, want %s archive", record.ArchiveName, tc.wantExt)
			}
			if record.Encrypted != tc.wantEncrypted {
				t.Fatalf("recorded encrypted = %v, want %v", record.Encrypted, tc.wantEncrypted)
			}
			if record.SizeBytes <= 0 {
				t.Fatalf("recorded size = %d, want > 0", record.SizeBytes)
			}
			if record.FinishedAt == nil {
				t.Fatal("recorded run has no finished_at")
			}
			if len(record.Destinations) != 0 {
				t.Fatalf("download run destinations = %+v, want none", record.Destinations)
			}

			var run model.BackupRun
			if err := svc.DB.First(&run, record.ID).Error; err != nil {
				t.Fatalf("reload run row: %v", err)
			}
			if run.GlobalBookkeepingStatus != StatusOK || run.BookkeepingStatus != StatusOK {
				t.Fatalf("run bookkeeping = %q/%q, want success/success", run.BookkeepingStatus, run.GlobalBookkeepingStatus)
			}
		})
	}
}

// TestBackupDBSameSecondDownloadsGetDistinctNames guards the archive_name
// unique index: two downloads in the same clock second must not collide now
// that downloads are recorded as runs.
func TestBackupDBSameSecondDownloadsGetDistinctNames(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	t.Cleanup(pkg.SetNowForTest(scheduleTime(3, 0)))

	first, err := svc.BackupDB(false, "")
	if err != nil {
		t.Fatalf("first BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(first)) })

	second, err := svc.BackupDB(false, "")
	if err != nil {
		t.Fatalf("second BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(second)) })

	if filepath.Base(first) == filepath.Base(second) {
		t.Fatalf("same-second downloads share the name %q", filepath.Base(first))
	}
	if got := countBackupRuns(t, svc); got != 2 {
		t.Fatalf("backup run count = %d, want 2", got)
	}
}

// TestBackupDBRecordsFailedDownloadRun makes sure a download that never
// produced an archive still shows up in the history as a failed backup rather
// than vanishing.
func TestBackupDBRecordsFailedDownloadRun(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("file permissions do not apply to root")
	}
	svc, dataDir := newBackupTestDB(t)

	// An unreadable asset makes the assets walk fail after the run row exists.
	assetsDir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	blocked := filepath.Join(assetsDir, "blocked.bin")
	if err := os.WriteFile(blocked, []byte("x"), 0o000); err != nil {
		t.Fatalf("create unreadable asset: %v", err)
	}

	if _, err := svc.BackupDB(true, ""); err == nil {
		t.Fatal("BackupDB() succeeded with an unreadable asset")
	}

	records, err := svc.ListBackupRuns(0)
	if err != nil {
		t.Fatalf("ListBackupRuns() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListBackupRuns() records = %d, want 1 failed run", len(records))
	}
	record := records[0]
	if record.Source != backupRunSourceDownload || record.Status != StatusFailed {
		t.Fatalf("run source/status = %q/%q, want download/failed", record.Source, record.Status)
	}
	if record.Error == "" {
		t.Fatal("failed run has no error message")
	}
	if record.SizeBytes != 0 {
		t.Fatalf("failed run size = %d, want 0", record.SizeBytes)
	}
}

// TestListBackupRunsMixesSourcesNewestFirst covers the whole point of the
// unified history: a manual destination run and a download both appear, newest
// first, and the limit caps the page.
func TestListBackupRunsMixesSourcesNewestFirst(t *testing.T) {
	svc, _ := newBackupTestDB(t)
	destID := createLocalDestination(t, svc, "", 7)

	if _, err := svc.RunDestinationBackup(context.Background(), destID); err != nil {
		t.Fatalf("RunDestinationBackup() error = %v", err)
	}
	path, err := svc.BackupDB(false, "")
	if err != nil {
		t.Fatalf("BackupDB() error = %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(path)) })

	records, err := svc.ListBackupRuns(0)
	if err != nil {
		t.Fatalf("ListBackupRuns() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("ListBackupRuns() records = %d, want 2", len(records))
	}
	if records[0].Source != backupRunSourceDownload || records[1].Source != backupRunSourceManual {
		t.Fatalf("record order = %q, %q; want download first, manual second", records[0].Source, records[1].Source)
	}
	if len(records[1].Destinations) != 1 || records[1].Destinations[0].DestinationID != destID {
		t.Fatalf("manual run destinations = %+v, want destination %d", records[1].Destinations, destID)
	}

	limited, err := svc.ListBackupRuns(1)
	if err != nil {
		t.Fatalf("ListBackupRuns(1) error = %v", err)
	}
	if len(limited) != 1 || limited[0].Source != backupRunSourceDownload {
		t.Fatalf("ListBackupRuns(1) = %+v, want only the newest (download) run", limited)
	}
}
