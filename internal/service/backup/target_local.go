package backup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidBackupObjectName = errors.New("invalid backup object name")

// localTarget delivers backups to a directory on the server's own filesystem.
// It preserves the exact behavior of the original single-destination local
// backup: the same default directory resolution, the same filename pattern
// gate for listing/retention, and the same AES detection for the listing UI.
type localTarget struct {
	dir            string
	retentionCount int
}

type localDestinationConfig struct {
	Dir            string `json:"dir"`
	RetentionCount int    `json:"retention_count"`
}

// newLocalTarget builds a local target from a decrypted config map. The "dir"
// field is normalized the same way the legacy backup_local_dir setting was
// (empty means the default <DATA_PATH>/backups); "retention_count" uses the
// default only when absent and is rejected when out of range.
func newLocalTarget(config map[string]any) (*localTarget, error) {
	var parsed localDestinationConfig
	if err := decodeDestinationConfigStrict(config, "local", &parsed); err != nil {
		return nil, err
	}

	dir, err := resolveBackupDir(parsed.Dir)
	if err != nil {
		return nil, err
	}

	retention, err := retentionCountFromConfig(config, parsed.RetentionCount)
	if err != nil {
		return nil, err
	}

	return &localTarget{dir: dir, retentionCount: retention}, nil
}

func (t *localTarget) Type() string { return "local" }

func (t *localTarget) RetentionCount() int { return t.retentionCount }

func (t *localTarget) Put(_ context.Context, name string, r io.Reader, _ int64) error {
	if !isSafeBackupFileName(name) {
		return ErrInvalidBackupObjectName
	}
	if err := os.MkdirAll(t.dir, 0o750); err != nil {
		return err
	}

	targetPath := filepath.Join(t.dir, name)
	file, err := os.CreateTemp(t.dir, ".subdux-backup-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	removeTemp := func() {
		_ = file.Close()
		_ = os.Remove(tempPath)
	}
	if err := file.Chmod(0o600); err != nil {
		removeTemp()
		return err
	}

	if _, err := io.Copy(file, r); err != nil {
		removeTemp()
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func (t *localTarget) List(_ context.Context) ([]BackupObject, error) {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []BackupObject{}, nil
		}
		return nil, err
	}

	items := make([]BackupObject, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !backupFileNamePattern.MatchString(entry.Name()) {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}
		fullPath := filepath.Join(t.dir, entry.Name())
		items = append(items, BackupObject{
			Name:       entry.Name(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			Encrypted:  backupArchiveIsEncrypted(fullPath),
		})
	}
	return items, nil
}

func (t *localTarget) Delete(_ context.Context, name string) error {
	if !isSafeBackupFileName(name) {
		// Defensive: retention only ever passes names it read back from List,
		// which are already pattern-filtered. Refuse anything else so a bad
		// caller can never delete an unrelated file in the backup directory.
		return nil
	}
	err := os.Remove(filepath.Join(t.dir, name))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func isSafeBackupFileName(name string) bool {
	return name != "" &&
		!strings.ContainsAny(name, `/\\`) &&
		name != "." && name != ".." &&
		backupFileNamePattern.MatchString(name)
}
