package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shiroha/subdux/internal/pkg"
)

func (s *AdminService) BackupDB(includeAssets bool) (string, error) {
	timestamp := pkg.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	if err := s.DB.Exec("VACUUM INTO ?", backupPath).Error; err != nil {
		return "", err
	}

	if !includeAssets {
		return backupPath, nil
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.zip", timestamp))
	if err := writeBackupZipFromDB(archivePath, backupPath, true, ""); err != nil {
		_ = os.Remove(backupPath)
		_ = os.Remove(archivePath)
		return "", err
	}

	_ = os.Remove(backupPath)

	return archivePath, nil
}
