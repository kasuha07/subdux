package api

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

type AdminHandler struct {
	Service     *service.AdminService
	TaskMonitor *service.BackgroundTaskMonitor
}

var (
	sqliteFileHeader = []byte("SQLite format 3\x00")
	errInvalidBackup = errors.New("invalid backup file")
)

const (
	maxBackupUploadSize            = 32 << 20 // 32 MiB
	maxBackupDatabaseExtractedSize = maxBackupUploadSize
	maxBackupAssetsExtractedSize   = 64 << 20
	maxBackupAssetEntries          = 2048
)

type backupRestorePayload struct {
	dbFilePath       string
	assetsDirPath    string
	replaceAssetsDir bool
}

type backupRestoreLimits struct {
	maxDatabaseExtractedSize int64
	maxAssetsExtractedSize   int64
	maxAssetEntries          int
}

var defaultBackupRestoreLimits = backupRestoreLimits{
	maxDatabaseExtractedSize: maxBackupDatabaseExtractedSize,
	maxAssetsExtractedSize:   maxBackupAssetsExtractedSize,
	maxAssetEntries:          maxBackupAssetEntries,
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

func NewAdminHandler(s *service.AdminService, taskMonitor *service.BackgroundTaskMonitor) *AdminHandler {
	return &AdminHandler{Service: s, TaskMonitor: taskMonitor}
}

type adminUserResponse struct {
	ID        uint      `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func mapAdminUserResponse(user model.User) adminUserResponse {
	return adminUserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Status:    user.Status,
		CreatedAt: user.CreatedAt,
	}
}

func mapAdminUserResponses(users []model.User) []adminUserResponse {
	responses := make([]adminUserResponse, len(users))
	for i, user := range users {
		responses[i] = mapAdminUserResponse(user)
	}
	return responses
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.Service.ListUsers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list users"})
	}
	return c.JSON(http.StatusOK, mapAdminUserResponses(users))
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var input service.CreateUserInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "username, email and password are required"})
	}

	if len(input.Password) < 8 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password must be at least 8 characters"})
	}
	if len([]byte(input.Password)) > 72 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password must not exceed 72 bytes"})
	}

	user, err := h.Service.CreateUser(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, mapAdminUserResponse(*user))
}

func (h *AdminHandler) ChangeUserRole(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	var input service.ChangeRoleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := h.Service.ChangeUserRole(uint(userID), input.Role); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "role updated"})
}

func (h *AdminHandler) ChangeUserStatus(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	var input service.ChangeStatusInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := h.Service.ChangeUserStatus(uint(userID), input.Status); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "status updated"})
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	currentUserID := getUserID(c)
	if currentUserID == uint(userID) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "cannot delete yourself"})
	}

	if err := h.Service.DeleteUser(uint(userID)); err != nil {
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "user deleted"})
}

func (h *AdminHandler) GetStats(c echo.Context) error {
	stats, err := h.Service.GetStats()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get stats"})
	}
	return c.JSON(http.StatusOK, stats)
}

func (h *AdminHandler) ListBackgroundTasks(c echo.Context) error {
	return c.JSON(http.StatusOK, h.TaskMonitor.List())
}

func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.Service.GetSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get settings"})
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var input service.UpdateSettingsInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := h.Service.UpdateSettings(input); err != nil {
		if errors.Is(err, service.ErrInvalidEmailDomainWhitelist) ||
			errors.Is(err, service.ErrEmailDomainWhitelistTooLong) ||
			errors.Is(err, service.ErrInvalidIconProxyDomainWhitelist) ||
			errors.Is(err, service.ErrIconProxyDomainWhitelistTooLong) ||
			errors.Is(err, service.ErrInvalidSMTPRateLimit) ||
			errors.Is(err, service.ErrInvalidSystemProxyType) ||
			errors.Is(err, service.ErrInvalidSystemProxyURL) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "settings updated"})
}

func (h *AdminHandler) TestSMTP(c echo.Context) error {
	var input struct {
		RecipientEmail string `json:"recipient_email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	currentUserID := getUserID(c)

	if err := h.Service.SendSMTPTestEmail(currentUserID, input.RecipientEmail); err != nil {
		if errors.Is(err, service.ErrSMTPRateLimited) {
			return c.JSON(http.StatusTooManyRequests, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "test email sent"})
}

func (h *AdminHandler) BackupDB(c echo.Context) error {
	includeAssets := false
	if rawIncludeAssets := c.QueryParam("include_assets"); rawIncludeAssets != "" {
		parsedIncludeAssets, err := strconv.ParseBool(rawIncludeAssets)
		if err != nil {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid include_assets query parameter"})
		}
		includeAssets = parsedIncludeAssets
	}

	backupPath, err := h.Service.BackupDB(includeAssets)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "backup failed"})
	}
	defer os.Remove(backupPath)

	filename := filepath.Base(backupPath)
	contentType := "application/octet-stream"
	if filepath.Ext(backupPath) == ".zip" {
		contentType = "application/zip"
	}

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Response().Header().Set("Content-Type", contentType)

	return c.File(backupPath)
}

func (h *AdminHandler) RestoreDB(c echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxBackupUploadSize)

	file, err := c.FormFile("backup")
	if err != nil {
		if isRequestTooLargeError(err) {
			return c.JSON(http.StatusRequestEntityTooLarge, echo.Map{
				"error": fmt.Sprintf("backup file is too large (max %d MB)", maxBackupUploadSize>>20),
			})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no file uploaded"})
	}

	uploadedBackupPath, err := saveUploadedBackupFile(file)
	if err != nil {
		if isRequestTooLargeError(err) {
			return c.JSON(http.StatusRequestEntityTooLarge, echo.Map{
				"error": fmt.Sprintf("backup file is too large (max %d MB)", maxBackupUploadSize>>20),
			})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to save uploaded backup"})
	}
	defer os.Remove(uploadedBackupPath)

	restorePayload, err := prepareRestorePayload(uploadedBackupPath)
	if err != nil {
		if errors.Is(err, errInvalidBackup) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to process backup file"})
	}
	if restorePayload.dbFilePath != "" && restorePayload.dbFilePath != uploadedBackupPath {
		defer os.Remove(restorePayload.dbFilePath)
	}
	if restorePayload.assetsDirPath != "" {
		defer os.RemoveAll(restorePayload.assetsDirPath)
	}

	dbPath := filepath.Join(pkg.GetDataPath(), "subdux.db")

	sqlDB, err := h.Service.DB.DB()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to access database"})
	}
	if err := sqlDB.Close(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to close database before restore"})
	}

	if err := replaceDatabaseFile(restorePayload.dbFilePath, dbPath); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to restore database"})
	}
	if restorePayload.replaceAssetsDir {
		if err := replaceAssetsDirectory(restorePayload.assetsDirPath); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to restore assets"})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "backup restored - please restart server"})
}

func saveUploadedBackupFile(fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	tempFile, err := os.CreateTemp("", "subdux-restore-upload-*"+ext)
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, src); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func prepareRestorePayload(uploadedBackupPath string) (*backupRestorePayload, error) {
	if isSQLiteBackupFile(uploadedBackupPath) {
		return &backupRestorePayload{
			dbFilePath: uploadedBackupPath,
		}, nil
	}

	if !isZipBackupFile(uploadedBackupPath) {
		return nil, invalidBackupError("unsupported format, please upload a .db or .zip backup")
	}

	return prepareRestorePayloadFromZip(uploadedBackupPath)
}

func prepareRestorePayloadFromZip(zipPath string) (*backupRestorePayload, error) {
	return prepareRestorePayloadFromZipWithLimits(zipPath, defaultBackupRestoreLimits)
}

func prepareRestorePayloadFromZipWithLimits(zipPath string, limits backupRestoreLimits) (*backupRestorePayload, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, invalidBackupError("invalid zip archive")
	}
	defer zipReader.Close()

	dbEntry, err := findDatabaseBackupEntry(zipReader.File)
	if err != nil {
		return nil, err
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
		return nil, invalidBackupError("failed to extract database from zip backup")
	}
	if !isSQLiteBackupFile(tempDBPath) {
		return nil, invalidBackupError("zip backup database is invalid")
	}

	replaceAssetsDir, assetsDirPath, err := extractAssetsFromZip(zipReader.File, limits)
	if err != nil {
		return nil, err
	}

	return &backupRestorePayload{
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

func extractAssetsFromZip(entries []*zip.File, limits backupRestoreLimits) (bool, string, error) {
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
		return nil, 0, invalidBackupError("failed to extract assets from zip backup")
	}
	defer source.Close()

	countingSource := &countingReader{reader: source}
	sanitized, _, err := service.SanitizeIconFile(countingSource, filename, maxBytes)
	if err != nil {
		return nil, 0, invalidBackupError("zip backup contains invalid asset image")
	}
	return sanitized, countingSource.bytesRead, nil
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
	return fmt.Errorf("%w: %s", errInvalidBackup, message)
}

func isSubPath(basePath string, targetPath string) bool {
	relativePath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}

	return relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(filepath.Separator))
}

func isRequestTooLargeError(err error) bool {
	if err == nil {
		return false
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "request body too large")
}
