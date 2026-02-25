package service

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/shiroha/subdux/internal/pkg"
)

func (s *AdminService) BackupDB(includeAssets bool) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.db", timestamp))

	if err := s.DB.Exec("VACUUM INTO ?", backupPath).Error; err != nil {
		return "", err
	}

	if !includeAssets {
		return backupPath, nil
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-backup-%s.zip", timestamp))
	if err := createBackupZip(archivePath, backupPath); err != nil {
		_ = os.Remove(backupPath)
		_ = os.Remove(archivePath)
		return "", err
	}

	_ = os.Remove(backupPath)

	return archivePath, nil
}

func createBackupZip(archivePath string, dbPath string) error {
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)

	if err := addFileToBackupZip(zipWriter, dbPath, "subdux.db"); err != nil {
		_ = zipWriter.Close()
		return err
	}

	if err := addAssetsToBackupZip(zipWriter); err != nil {
		_ = zipWriter.Close()
		return err
	}

	return zipWriter.Close()
}

func addFileToBackupZip(zipWriter *zip.Writer, sourcePath string, archivePath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	targetFile, err := zipWriter.Create(archivePath)
	if err != nil {
		return err
	}

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}

	return nil
}

func addAssetsToBackupZip(zipWriter *zip.Writer) error {
	assetsRoot := filepath.Join(pkg.GetDataPath(), "assets")
	if err := addDirectoryToBackupZip(zipWriter, "assets/"); err != nil {
		return err
	}

	info, err := os.Stat(assetsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.Walk(assetsRoot, func(path string, fileInfo os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if fileInfo.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(assetsRoot, path)
		if err != nil {
			return err
		}

		archivePath := filepath.ToSlash(filepath.Join("assets", relativePath))
		return addFileToBackupZip(zipWriter, path, archivePath)
	})
}

func addDirectoryToBackupZip(zipWriter *zip.Writer, archivePath string) error {
	header := &zip.FileHeader{
		Name: archivePath,
	}
	header.SetMode(os.ModeDir | 0o755)

	_, err := zipWriter.CreateHeader(header)
	return err
}
