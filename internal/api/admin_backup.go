package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

const (
	maxBackupUploadSize = 32 << 20 // 32 MiB
)

func (h *AdminHandler) BackupDB(c echo.Context) error {
	var input struct {
		IncludeAssets bool   `json:"include_assets"`
		Password      string `json:"password"`
	}
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		apimw.From(c).UserID,
		servicereauth.ReauthOperationBackup,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	backupPath, err := h.Backup.WithContext(c.Request().Context()).BackupDB(input.IncludeAssets, input.Password)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "backup failed")
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

func (h *AdminHandler) RunBackupNow(c echo.Context) error {
	backupPath, err := h.Backup.WithContext(c.Request().Context()).CreateLocalBackup()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message": "backup created",
		"file":    filepath.Base(backupPath),
	})
}

type localBackupResponse struct {
	Directory string                          `json:"directory"`
	Backups   []servicebackup.LocalBackupInfo `json:"backups"`
}

func (h *AdminHandler) ListLocalBackups(c echo.Context) error {
	directory, backups, err := h.Backup.WithContext(c.Request().Context()).ListLocalBackups()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, localBackupResponse{
		Directory: directory,
		Backups:   backups,
	})
}

func (h *AdminHandler) RestoreDB(c echo.Context) error {
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxBackupUploadSize)

	// Re-authenticate before any work: the ticket is carried in a header (not the
	// multipart body) so it can be consumed before the request body is parsed.
	// This gates the (potentially large) upload and the destructive replace, and
	// identity is verified before the server reads anything. The ticket is
	// single-use, so a failure after this point requires a fresh re-auth on retry
	// — an intentional trade-off that keeps the destructive step strictly behind a
	// proven-present admin.
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		apimw.From(c).UserID,
		servicereauth.ReauthOperationRestore,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	file, err := c.FormFile("backup")
	if err != nil {
		if httpx.IsRequestTooLargeError(err) {
			return httpx.WriteError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("backup file is too large (max %d MB)", maxBackupUploadSize>>20))
		}
		return httpx.WriteError(c, http.StatusBadRequest, "no file uploaded")
	}

	uploadedBackupPath, err := saveUploadedBackupFile(file)
	if err != nil {
		if httpx.IsRequestTooLargeError(err) {
			return httpx.WriteError(c, http.StatusRequestEntityTooLarge, fmt.Sprintf("backup file is too large (max %d MB)", maxBackupUploadSize>>20))
		}
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to save uploaded backup")
	}
	defer os.Remove(uploadedBackupPath)

	password := strings.TrimSpace(c.FormValue("password"))

	result, err := h.Backup.WithContext(c.Request().Context()).RestoreBackup(uploadedBackupPath, password)
	if err != nil {
		return writeRestoreBackupError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"message":             "backup restored - please restart server",
		"skipped_asset_count": result.SkippedAssetCount,
	})
}

func writeRestoreBackupError(c echo.Context, err error) error {
	if _, ok := serviceerr.KindOf(err); ok {
		return err
	}
	return httpx.WriteError(c, http.StatusInternalServerError, "failed to restore backup")
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
