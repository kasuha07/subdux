package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shiroha/subdux/internal/pkg"
)

// BackupDB produces an on-demand backup and returns the path of the file to
// serve. When password is empty the historical behavior is preserved: a raw
// SQLite .db file when includeAssets is false, or a plain .zip when true.
// When password is non-empty encryption requires a zip container, so the DB is
// always bundled into a WinZip AES-256 .zip (with assets honored) regardless of
// includeAssets. The password is trimmed before deciding, so all-whitespace is
// treated as empty.
func (s *AdminService) BackupDB(includeAssets bool, password string) (string, error) {
	password = strings.TrimSpace(password)

	timestamp := pkg.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	if err := s.DB.Exec("VACUUM INTO ?", backupPath).Error; err != nil {
		return "", err
	}

	if !includeAssets && password == "" {
		return backupPath, nil
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.zip", timestamp))
	if err := writeBackupZipFromDB(archivePath, backupPath, includeAssets, password); err != nil {
		_ = os.Remove(backupPath)
		_ = os.Remove(archivePath)
		return "", err
	}

	_ = os.Remove(backupPath)

	return archivePath, nil
}
