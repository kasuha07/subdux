package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

// fakeArchiveTarget is a BackupTarget test double that lets a test control
// exactly what List and Get report, independently of each other — this is
// what makes it possible to exercise the streamed-size cap against a target
// that misreports its own object sizes.
type fakeArchiveTarget struct {
	objects  []BackupObject
	payloads map[string][]byte
	sizes    map[string]int64 // reported size per name; falls back to len(payloads[name])
	getErr   error
}

func (t *fakeArchiveTarget) Type() string { return "fake" }

func (t *fakeArchiveTarget) Put(context.Context, string, io.Reader, int64) error { return nil }

func (t *fakeArchiveTarget) List(context.Context) ([]BackupObject, error) {
	return t.objects, nil
}

func (t *fakeArchiveTarget) Delete(context.Context, string) error { return nil }

func (t *fakeArchiveTarget) RetentionCount() int { return 0 }

func (t *fakeArchiveTarget) Get(_ context.Context, name string) (io.ReadCloser, int64, error) {
	if t.getErr != nil {
		return nil, 0, t.getErr
	}
	payload := t.payloads[name]
	size, ok := t.sizes[name]
	if !ok {
		size = int64(len(payload))
	}
	return io.NopCloser(bytes.NewReader(payload)), size, nil
}

func TestDownloadArchiveToTempEnforcesReportedSizeCap(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	target := &fakeArchiveTarget{
		payloads: map[string][]byte{"a.zip": []byte("small payload")},
		sizes:    map[string]int64{"a.zip": 2 << 20},
	}

	_, _, err := downloadArchiveToTemp(context.Background(), target, "a.zip", 1<<20)
	var svcErr *serviceerr.Error
	if !errors.As(err, &svcErr) || svcErr.Code != "backup_file_is_too_large_max_mb" {
		t.Fatalf("downloadArchiveToTemp() error = %v, want backup_file_is_too_large_max_mb", err)
	}
	if got := svcErr.Params["max_mb"]; got != int64(1) {
		t.Fatalf("max_mb param = %v, want 1", got)
	}

	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		t.Fatalf("ReadDir(TMPDIR) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("TMPDIR entries = %v, want none (no temp file left behind)", entries)
	}
}

func TestDownloadArchiveToTempEnforcesStreamedSizeCapWhenSizeIsUnderreported(t *testing.T) {
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	payload := bytes.Repeat([]byte("x"), 2<<20)
	target := &fakeArchiveTarget{
		payloads: map[string][]byte{"a.zip": payload},
		sizes:    map[string]int64{"a.zip": -1},
	}

	_, _, err := downloadArchiveToTemp(context.Background(), target, "a.zip", 1<<20)
	var svcErr *serviceerr.Error
	if !errors.As(err, &svcErr) || svcErr.Code != "backup_file_is_too_large_max_mb" {
		t.Fatalf("downloadArchiveToTemp() error = %v, want backup_file_is_too_large_max_mb", err)
	}

	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		t.Fatalf("ReadDir(TMPDIR) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("TMPDIR entries = %v, want none (staging directory removed)", entries)
	}
}

func TestDownloadArchiveToTempAcceptsExactLimit(t *testing.T) {
	const maxBytes = 1 << 20
	payload := bytes.Repeat([]byte("y"), maxBytes)
	target := &fakeArchiveTarget{
		payloads: map[string][]byte{"a.zip": payload},
		sizes:    map[string]int64{"a.zip": maxBytes},
	}

	path, cleanup, err := downloadArchiveToTemp(context.Background(), target, "a.zip", maxBytes)
	if err != nil {
		t.Fatalf("downloadArchiveToTemp() error = %v, want nil", err)
	}
	defer cleanup()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read staged archive: %v", err)
	}
	if len(got) != maxBytes {
		t.Fatalf("staged archive size = %d, want %d", len(got), maxBytes)
	}
}

func TestDownloadArchiveToTempWrapsTransportFailure(t *testing.T) {
	target := &fakeArchiveTarget{getErr: errors.New("connection reset")}

	_, _, err := downloadArchiveToTemp(context.Background(), target, "a.zip", 1<<20)
	if !errors.Is(err, ErrBackupArchiveDownloadFailed) {
		t.Fatalf("downloadArchiveToTemp() error = %v, want ErrBackupArchiveDownloadFailed", err)
	}
}

func TestRestoreDestinationBackupRejectsUnknownDestination(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	_, err := svc.RestoreDestinationBackup(context.Background(), 9999, "subdux-backup-20260615-030000-abc123.zip", "")
	if !errors.Is(err, ErrBackupDestinationNotFound) {
		t.Fatalf("RestoreDestinationBackup() error = %v, want ErrBackupDestinationNotFound", err)
	}
}

func TestRestoreDestinationBackupRejectsArchiveNotInListing(t *testing.T) {
	svc, _ := newBackupTestDB(t)

	archiveDir := t.TempDir()
	tmpRoot := t.TempDir()
	t.Setenv("TMPDIR", tmpRoot)

	destination, err := svc.CreateDestination(CreateDestinationInput{
		Type:    "local",
		Enabled: true,
		Config:  fmt.Sprintf(`{"dir":%q}`, archiveDir),
	})
	if err != nil {
		t.Fatalf("CreateDestination() error = %v", err)
	}

	_, err = svc.RestoreDestinationBackup(context.Background(), destination.ID, "subdux-backup-20260615-030000-missing.zip", "")
	if !errors.Is(err, ErrBackupArchiveNotFound) {
		t.Fatalf("RestoreDestinationBackup() error = %v, want ErrBackupArchiveNotFound", err)
	}

	// The listing-membership gate must refuse before anything is fetched, so no
	// staging directory is created under TMPDIR.
	entries, err := os.ReadDir(tmpRoot)
	if err != nil {
		t.Fatalf("ReadDir(TMPDIR) error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("TMPDIR entries after rejection = %v, want none (nothing fetched)", entries)
	}
}
