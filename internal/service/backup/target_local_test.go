package backup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type gatedReader struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (r *gatedReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.release
	payload := []byte("new archive contents")
	copy(p, payload)
	return len(payload), io.EOF
}

type failingReader struct {
	done bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	r.done = true
	payload := []byte("partial archive")
	copy(p, payload)
	return len(payload), errors.New("source read failed")
}

func TestLocalTargetPutUses0600TempFileAndAtomicRename(t *testing.T) {
	dir := t.TempDir()
	target := &localTarget{dir: dir}
	name := "subdux-backup-20260717-030000-test.zip"
	targetPath := filepath.Join(dir, name)
	if err := os.WriteFile(targetPath, []byte("old archive contents"), 0o644); err != nil {
		t.Fatalf("write existing archive: %v", err)
	}
	if err := os.Chmod(targetPath, 0o644); err != nil {
		t.Fatalf("chmod existing archive: %v", err)
	}

	reader := &gatedReader{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- target.Put(context.Background(), name, reader, -1)
	}()

	select {
	case <-reader.started:
	case <-time.After(2 * time.Second):
		t.Fatal("Put() did not start reading the archive")
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read final path while copy is blocked: %v", err)
	}
	if string(got) != "old archive contents" {
		t.Fatalf("final path contents while copy is blocked = %q, want previous archive", got)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat final path while copy is blocked: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("final path permissions while copy is blocked = %04o, want existing 0644", got)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read target directory while copy is blocked: %v", err)
	}
	foundTemp := false
	for _, entry := range entries {
		if entry.Name() == name {
			continue
		}
		foundTemp = true
		tempInfo, infoErr := entry.Info()
		if infoErr != nil {
			t.Fatalf("stat temporary archive %q: %v", entry.Name(), infoErr)
		}
		if got := tempInfo.Mode().Perm(); got != 0o600 {
			t.Fatalf("temporary archive permissions = %04o, want 0600", got)
		}
	}
	if !foundTemp {
		t.Fatal("Put() did not create a same-directory temporary archive")
	}

	close(reader.release)
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Put() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Put() did not finish after the archive source was released")
	}

	got, err = os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read final archive: %v", err)
	}
	if string(got) != "new archive contents" {
		t.Fatalf("final archive contents = %q, want new archive", got)
	}
	info, err = os.Stat(targetPath)
	if err != nil {
		t.Fatalf("stat final archive: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("final archive permissions = %04o, want 0600", got)
	}

	entries, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read target directory after Put(): %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != name {
		t.Fatalf("target directory entries after Put() = %v, want only %s", entries, name)
	}
}

func TestLocalTargetPutRemovesTemporaryFileOnCopyFailure(t *testing.T) {
	dir := t.TempDir()
	target := &localTarget{dir: dir}
	name := "subdux-backup-20260717-030000-test.zip"

	if err := target.Put(context.Background(), name, &failingReader{}, -1); err == nil {
		t.Fatal("Put() error = nil, want source read failure")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read target directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("target directory entries after failed Put() = %v, want empty", entries)
	}
}

func TestLocalTargetRejectsUnsafeObjectName(t *testing.T) {
	dir := t.TempDir()
	target := &localTarget{dir: dir}
	if err := target.Put(context.Background(), "../outside.zip", strings.NewReader("archive"), 7); !errors.Is(err, ErrInvalidBackupObjectName) {
		t.Fatalf("Put() error = %v, want ErrInvalidBackupObjectName", err)
	}
	if err := target.Delete(context.Background(), "../outside.zip"); err != nil {
		t.Fatalf("Delete() error = %v, want ignored unsafe name", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("target directory entries = %v, want empty", entries)
	}
}

func TestLocalTargetGetRejectsUnsafeAndIrregularNames(t *testing.T) {
	const validName = "subdux-backup-20260615-030000-abc123.zip"

	tests := []struct {
		name    string
		object  string
		setup   func(t *testing.T, dir string)
		wantErr error // nil means "any non-nil error is acceptable"
	}{
		{name: "traversal", object: "../outside.zip", wantErr: ErrInvalidBackupObjectName},
		{name: "separator", object: "sub/dir.zip", wantErr: ErrInvalidBackupObjectName},
		{name: "non-archive", object: "notes.txt", wantErr: ErrInvalidBackupObjectName},
		{name: "empty", object: "", wantErr: ErrInvalidBackupObjectName},
		{name: "dot-dot", object: "..", wantErr: ErrInvalidBackupObjectName},
		{
			name:   "directory",
			object: validName,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, validName), 0o750); err != nil {
					t.Fatalf("Mkdir() error = %v", err)
				}
			},
			wantErr: ErrInvalidBackupObjectName,
		},
		{
			name:   "symlink",
			object: validName,
			setup: func(t *testing.T, dir string) {
				t.Helper()
				outside := filepath.Join(t.TempDir(), "outside.zip")
				if err := os.WriteFile(outside, []byte("archive"), 0o600); err != nil {
					t.Fatalf("write outside file: %v", err)
				}
				if err := os.Symlink(outside, filepath.Join(dir, validName)); err != nil {
					t.Fatalf("Symlink() error = %v", err)
				}
			},
			wantErr: ErrInvalidBackupObjectName,
		},
		{name: "missing", object: validName},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			target := &localTarget{dir: dir}
			if tc.setup != nil {
				tc.setup(t, dir)
			}

			reader, size, err := target.Get(context.Background(), tc.object)
			if err == nil {
				_ = reader.Close()
				t.Fatalf("Get() error = nil, size = %d, want an error", size)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestLocalTargetGetReturnsArchiveBytesAndSize(t *testing.T) {
	dir := t.TempDir()
	target := &localTarget{dir: dir}
	name := "subdux-backup-20260615-030000-abc123.zip"
	payload := []byte("archive contents")
	if err := os.WriteFile(filepath.Join(dir, name), payload, 0o600); err != nil {
		t.Fatalf("write archive: %v", err)
	}

	reader, size, err := target.Get(context.Background(), name)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if size != int64(len(payload)) {
		t.Fatalf("Get() size = %d, want %d", size, len(payload))
	}
	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("Get() contents = %q, want %q", got, payload)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}
}
