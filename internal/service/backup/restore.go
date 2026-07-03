package backup

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/yeka/zip"
)

var (
	sqliteFileHeader          = []byte("SQLite format 3\x00")
	ErrInvalidBackup          = errors.New("invalid backup file")
	ErrBackupPasswordRequired = errors.New("backup is encrypted; a password is required")
	ErrBackupInvalidPassword  = errors.New("invalid backup password")
)

type restorePayload struct {
	dbFilePath       string
	assetsDirPath    string
	replaceAssetsDir bool
}

type restoreLimits struct {
	maxDatabaseExtractedSize int64
	maxAssetsExtractedSize   int64
	maxAssetEntries          int
}

var defaultRestoreLimits = restoreLimits{
	maxDatabaseExtractedSize: 32 << 20,
	maxAssetsExtractedSize:   64 << 20,
	maxAssetEntries:          2048,
}

func isRestorableAssetPath(relativePath string) bool {
	if relativePath == "" {
		return false
	}
	parts := strings.Split(relativePath, "/")
	if len(parts) != 2 || parts[0] != "icons" {
		return false
	}
	filename := parts[1]
	if filename == "" || path.Base(filename) != filename {
		return false
	}
	ext := strings.ToLower(path.Ext(filename))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".ico"
}

func (s *Service) RestoreBackup(uploadedBackupPath string, password string) error {
	password = strings.TrimSpace(password)

	restorePayload, err := prepareRestorePayload(uploadedBackupPath, password)
	if err != nil {
		return err
	}
	if restorePayload.dbFilePath != "" && restorePayload.dbFilePath != uploadedBackupPath {
		defer os.Remove(restorePayload.dbFilePath)
	}
	if restorePayload.assetsDirPath != "" {
		defer os.RemoveAll(restorePayload.assetsDirPath)
	}

	dbPath := filepath.Join(pkg.GetDataPath(), "subdux.db")

	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	if err := sqlDB.Close(); err != nil {
		return err
	}

	if err := replaceDatabaseFile(restorePayload.dbFilePath, dbPath); err != nil {
		return err
	}
	if restorePayload.replaceAssetsDir {
		if err := replaceAssetsDirectory(restorePayload.assetsDirPath); err != nil {
			return err
		}
	}

	return nil
}

func prepareRestorePayload(uploadedBackupPath string, password string) (*restorePayload, error) {
	if isSQLiteBackupFile(uploadedBackupPath) {
		return &restorePayload{
			dbFilePath: uploadedBackupPath,
		}, nil
	}

	if !isZipBackupFile(uploadedBackupPath) {
		return nil, invalidBackupError("unsupported format, please upload a .db or .zip backup")
	}

	return prepareRestorePayloadFromZip(uploadedBackupPath, password)
}

func prepareRestorePayloadFromZip(zipPath string, password string) (*restorePayload, error) {
	return prepareRestorePayloadFromZipWithLimits(zipPath, password, defaultRestoreLimits)
}

func prepareRestorePayloadFromZipWithLimits(zipPath string, password string, limits restoreLimits) (*restorePayload, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, invalidBackupError("invalid zip archive")
	}
	defer zipReader.Close()

	dbEntry, err := findDatabaseBackupEntry(zipReader.File)
	if err != nil {
		return nil, err
	}

	// An encrypted archive requires a password. Detection keys off the database
	// entry; every entry is decrypted with the same password below.
	encrypted := dbEntry.IsEncrypted()
	if encrypted && password == "" {
		return nil, ErrBackupPasswordRequired
	}
	if encrypted {
		for _, entry := range zipReader.File {
			entry.SetPassword(password)
		}
	}

	tempDBFile, err := os.CreateTemp("", "subdux-restore-db-*.db")
	if err != nil {
		return nil, err
	}
	tempDBPath := tempDBFile.Name()
	if err = tempDBFile.Close(); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(tempDBPath)
		}
	}()

	if err = validateZipFileEntrySize(dbEntry, limits.maxDatabaseExtractedSize, "zip backup database exceeds extracted size limit"); err != nil {
		return nil, err
	}
	if _, err = extractZipFileEntryLimited(dbEntry, tempDBPath, limits.maxDatabaseExtractedSize); err != nil {
		if isZipPasswordError(err) {
			return nil, ErrBackupInvalidPassword
		}
		return nil, invalidBackupError("failed to extract database from zip backup")
	}
	if !isSQLiteBackupFile(tempDBPath) {
		return nil, invalidBackupError("zip backup database is invalid")
	}

	replaceAssetsDir, assetsDirPath, err := extractAssetsFromZip(zipReader.File, limits)
	if err != nil {
		return nil, err
	}

	return &restorePayload{
		dbFilePath:       tempDBPath,
		assetsDirPath:    assetsDirPath,
		replaceAssetsDir: replaceAssetsDir,
	}, nil
}

func findDatabaseBackupEntry(entries []*zip.File) (*zip.File, error) {
	var fallback *zip.File
	var preferred *zip.File

	for _, entry := range entries {
		cleanPath, ok := normalizeZipEntryPath(entry.Name)
		if !ok || cleanPath == "assets" || strings.HasPrefix(cleanPath, "assets/") {
			continue
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		if !entry.Mode().IsRegular() {
			continue
		}

		lowerCleanPath := strings.ToLower(cleanPath)
		if lowerCleanPath == "subdux.db" {
			return entry, nil
		}
		if preferred == nil && strings.EqualFold(path.Base(cleanPath), "subdux.db") {
			preferred = entry
			continue
		}
		if fallback == nil && strings.EqualFold(path.Ext(cleanPath), ".db") {
			fallback = entry
		}
	}

	if preferred != nil {
		return preferred, nil
	}
	if fallback != nil {
		return fallback, nil
	}

	return nil, invalidBackupError("zip backup does not contain a database file")
}

func extractAssetsFromZip(entries []*zip.File, limits restoreLimits) (bool, string, error) {
	shouldRestoreAssets := false
	for _, entry := range entries {
		cleanPath, ok := normalizeZipEntryPath(entry.Name)
		if !ok {
			continue
		}
		if cleanPath == "assets" || strings.HasPrefix(cleanPath, "assets/") {
			shouldRestoreAssets = true
			break
		}
	}

	if !shouldRestoreAssets {
		return false, "", nil
	}

	dataDir := pkg.GetDataPath()
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return false, "", err
	}

	tempAssetsDir, err := os.MkdirTemp(dataDir, ".subdux-restore-assets-*")
	if err != nil {
		return false, "", err
	}
	shouldCleanup := true
	defer func() {
		if shouldCleanup {
			_ = os.RemoveAll(tempAssetsDir)
		}
	}()

	var extractedSize int64
	assetEntries := 0

	for _, entry := range entries {
		cleanPath, ok := normalizeZipEntryPath(entry.Name)
		if !ok {
			return false, "", invalidBackupError("zip backup contains unsafe paths")
		}
		if cleanPath == "assets" || !strings.HasPrefix(cleanPath, "assets/") {
			continue
		}

		relativePath := strings.TrimPrefix(cleanPath, "assets/")
		if relativePath == "" {
			continue
		}
		if entry.FileInfo().IsDir() {
			if relativePath == "icons" {
				continue
			}
			return false, "", invalidBackupError("zip backup contains unsupported assets entry")
		}
		if !isRestorableAssetPath(relativePath) {
			return false, "", invalidBackupError("zip backup contains unsupported assets entry")
		}

		mode := entry.Mode()
		if !mode.IsRegular() {
			return false, "", invalidBackupError("zip backup contains unsupported assets entry")
		}

		assetEntries++
		if assetEntries > limits.maxAssetEntries {
			return false, "", invalidBackupError("zip backup contains too many assets")
		}

		remainingSize := limits.maxAssetsExtractedSize - extractedSize
		if remainingSize < 0 {
			remainingSize = 0
		}
		if err := validateZipFileEntrySize(entry, remainingSize, "zip backup assets exceed extracted size limit"); err != nil {
			return false, "", err
		}

		sanitized, sourceSize, err := sanitizeRestoreAsset(entry, path.Base(relativePath), remainingSize)
		if err != nil {
			return false, "", err
		}
		targetPath := filepath.Join(tempAssetsDir, filepath.FromSlash(relativePath))
		if !isSubPath(tempAssetsDir, targetPath) {
			return false, "", invalidBackupError("zip backup contains invalid assets path")
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
			return false, "", err
		}
		if err := os.WriteFile(targetPath, sanitized, 0o600); err != nil {
			return false, "", invalidBackupError("failed to extract assets from zip backup")
		}
		extractedSize += sourceSize
	}

	shouldCleanup = false
	return true, tempAssetsDir, nil
}

func sanitizeRestoreAsset(entry *zip.File, filename string, maxBytes int64) ([]byte, int64, error) {
	source, err := entry.Open()
	if err != nil {
		if isZipPasswordError(err) {
			return nil, 0, ErrBackupInvalidPassword
		}
		return nil, 0, invalidBackupError("failed to extract assets from zip backup")
	}
	defer source.Close()

	countingSource := &countingReader{reader: source}
	sanitized, _, err := serviceutil.SanitizeIconFile(countingSource, filename, maxBytes)
	if err != nil {
		if isZipPasswordError(err) {
			return nil, 0, ErrBackupInvalidPassword
		}
		return nil, 0, invalidBackupError("zip backup contains invalid asset image")
	}
	return sanitized, countingSource.bytesRead, nil
}

// isZipPasswordError reports whether err is one of the yeka/zip decryption
// failures raised when an encrypted entry is read with a wrong or missing
// password. yeka surfaces these on Open or first Read/io.Copy.
func isZipPasswordError(err error) bool {
	return errors.Is(err, zip.ErrPassword) ||
		errors.Is(err, zip.ErrAuthentication) ||
		errors.Is(err, zip.ErrDecryption)
}

type countingReader struct {
	reader    io.Reader
	bytesRead int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += int64(n)
	return n, err
}

func validateZipFileEntrySize(entry *zip.File, maxBytes int64, message string) error {
	if maxBytes < 0 || entry.UncompressedSize64 > uint64(maxBytes) {
		return invalidBackupError(message)
	}
	return nil
}

func extractZipFileEntryLimited(entry *zip.File, targetPath string, maxBytes int64) (int64, error) {
	if maxBytes < 0 {
		return 0, invalidBackupError("zip backup entry exceeds extracted size limit")
	}

	source, err := entry.Open()
	if err != nil {
		return 0, err
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600) // #nosec G304 -- targetPath is an internal temporary restore path.
	if err != nil {
		return 0, err
	}
	defer target.Close()

	limited := &io.LimitedReader{R: source, N: maxBytes + 1}
	written, err := io.Copy(target, limited)
	if err != nil {
		return written, err
	}
	if written > maxBytes {
		return written, invalidBackupError("zip backup entry exceeds extracted size limit")
	}

	return written, nil
}

func replaceDatabaseFile(sourcePath string, dbPath string) error {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0o750); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dbDir, ".subdux-restore-db-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	defer func() {
		_ = os.Remove(tempPath)
	}()

	source, err := os.Open(sourcePath) // #nosec G304 -- sourcePath is an internally-created and validated temporary restore DB path.
	if err != nil {
		_ = tempFile.Close()
		return err
	}
	defer source.Close()

	if _, err := io.Copy(tempFile, source); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempPath, dbPath)
}

func replaceAssetsDirectory(sourceAssetsDir string) error {
	dataDir := pkg.GetDataPath()
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return err
	}

	targetAssetsDir := filepath.Join(dataDir, "assets")
	previousAssetsDir := filepath.Join(dataDir, fmt.Sprintf(".subdux-restore-assets-prev-%d", pkg.Now().UnixNano()))
	previousAssetsExists := false

	if _, err := os.Stat(targetAssetsDir); err == nil {
		previousAssetsExists = true
		if err := os.Rename(targetAssetsDir, previousAssetsDir); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.Rename(sourceAssetsDir, targetAssetsDir); err != nil {
		if previousAssetsExists {
			_ = os.Rename(previousAssetsDir, targetAssetsDir)
		}
		return err
	}

	if previousAssetsExists {
		_ = os.RemoveAll(previousAssetsDir)
	}

	return nil
}

func normalizeZipEntryPath(entryName string) (string, bool) {
	sanitized := strings.TrimSpace(strings.ReplaceAll(entryName, "\\", "/"))
	if sanitized == "" {
		return "", false
	}
	if strings.HasPrefix(sanitized, "/") {
		return "", false
	}

	cleanPath := path.Clean(sanitized)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", false
	}

	return cleanPath, true
}

func isSQLiteBackupFile(filePath string) bool {
	file, err := os.Open(filePath) // #nosec G304 -- filePath is an internally-created upload/extraction temp file, not a client-chosen path.
	if err != nil {
		return false
	}
	defer file.Close()

	header := make([]byte, len(sqliteFileHeader))
	if _, err := io.ReadFull(file, header); err != nil {
		return false
	}

	return bytes.Equal(header, sqliteFileHeader)
}

func isZipBackupFile(filePath string) bool {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return false
	}
	defer reader.Close()

	return true
}

func invalidBackupError(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidBackup, message)
}

func isSubPath(basePath string, targetPath string) bool {
	relativePath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}

	return relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}
