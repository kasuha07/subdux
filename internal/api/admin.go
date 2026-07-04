package api

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	Service     *adminservice.Service
	TaskMonitor *serviceutil.BackgroundTaskMonitor
	Reauth      *servicereauth.Service
	Backup      *servicebackup.Service
}

const (
	maxBackupUploadSize = 32 << 20 // 32 MiB
)

func NewAdminHandler(s *adminservice.Service, taskMonitor *serviceutil.BackgroundTaskMonitor, reauth *servicereauth.Service, backup *servicebackup.Service) *AdminHandler {
	return &AdminHandler{Service: s, TaskMonitor: taskMonitor, Reauth: reauth, Backup: backup}
}

type adminUserResponse struct {
	ID                uint      `json:"id"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Role              string    `json:"role"`
	Status            string    `json:"status"`
	TotpEnabled       bool      `json:"totp_enabled"`
	PasskeyCount      int64     `json:"passkey_count"`
	CreatedAt         time.Time `json:"created_at"`
	SubscriptionCount int64     `json:"subscription_count"`
}

func mapAdminUserResponse(user adminservice.AdminUserListItem) adminUserResponse {
	return adminUserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Role:              user.Role,
		Status:            user.Status,
		TotpEnabled:       user.TotpEnabled,
		PasskeyCount:      user.PasskeyCount,
		CreatedAt:         user.CreatedAt,
		SubscriptionCount: user.SubscriptionCount,
	}
}

func mapAdminUserResponses(users []adminservice.AdminUserListItem) []adminUserResponse {
	responses := make([]adminUserResponse, len(users))
	for i, user := range users {
		responses[i] = mapAdminUserResponse(user)
	}
	return responses
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.Service.WithContext(c.Request().Context()).ListUsers()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list users")
	}
	return c.JSON(http.StatusOK, mapAdminUserResponses(users))
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var input adminservice.CreateUserInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "username, email and password are required")
	}

	if len(input.Password) < 8 {
		return httpx.WriteError(c, http.StatusBadRequest, "password must be at least 8 characters")
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.Password); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "password must not exceed 72 bytes")
	}

	if operation, ok := servicereauth.OperationForCreateUserRole(input.Role); ok {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			operation,
			reauthTicketFromRequest(c),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	user, err := h.Service.WithContext(c.Request().Context()).CreateUser(input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, mapAdminUserResponse(adminservice.AdminUserListItem{User: *user}))
}

func (h *AdminHandler) ChangeUserRole(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	var input adminservice.ChangeRoleInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Role != "admin" && input.Role != "user" {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid role")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		apimw.From(c).UserID,
		servicereauth.ReauthOperationChangeUserRole,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserRole(uint(userID), input.Role); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "role updated"})
}

func (h *AdminHandler) ChangeUserStatus(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	var input adminservice.ChangeStatusInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserStatus(uint(userID), input.Status); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "status updated"})
}

func (h *AdminHandler) DisableUserTOTP(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserTOTP(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "two-factor authentication disabled"})
}

func (h *AdminHandler) DisableUserPasskeys(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserPasskeys(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkeys disabled"})
}

func writeAdminCredentialResetError(c echo.Context, err error) error {
	return err
}

func (h *AdminHandler) DeleteUser(c echo.Context) error {
	userID, ok := httpx.ParseUintParam(c, "id", "invalid user id")
	if !ok {
		return nil
	}

	currentUserID := apimw.From(c).UserID
	if currentUserID == uint(userID) {
		return httpx.WriteError(c, http.StatusBadRequest, "cannot delete yourself")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		currentUserID,
		servicereauth.ReauthOperationDeleteUser,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteUser(uint(userID)); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "user deleted"})
}

func (h *AdminHandler) ListBackgroundTasks(c echo.Context) error {
	return c.JSON(http.StatusOK, h.TaskMonitor.List())
}

func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.Service.WithContext(c.Request().Context()).GetSettings()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to get settings")
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var input adminservice.UpdateSettingsInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if operation, ok := servicereauth.OperationForAdminSettingsUpdate(servicereauth.AdminSettingsUpdateInput{
		BackupScheduleEnabled:    input.BackupScheduleEnabled,
		BackupTimeOfDay:          input.BackupTimeOfDay,
		BackupIncludeAssets:      input.BackupIncludeAssets,
		BackupEncryptEnabled:     input.BackupEncryptEnabled,
		BackupEncryptionPassword: input.BackupEncryptionPassword,
		BackupLocalDir:           input.BackupLocalDir,
		BackupRetentionCount:     input.BackupRetentionCount,
	}); ok {
		if h.Reauth == nil {
			return httpx.WriteError(c, http.StatusInternalServerError, "reauthentication service is not configured")
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			operation,
			reauthTicketFromRequest(c),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	if err := h.Service.WithContext(c.Request().Context()).UpdateSettings(input); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "settings updated"})
}

func (h *AdminHandler) TestSSRF(c echo.Context) error {
	var input adminservice.SSRFTestInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	result, err := h.Service.WithContext(c.Request().Context()).TestSSRF(input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) TestSMTP(c echo.Context) error {
	var input struct {
		RecipientEmail string `json:"recipient_email"`
	}
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	currentUserID := apimw.From(c).UserID

	if err := h.Service.WithContext(c.Request().Context()).SendSMTPTestEmail(currentUserID, input.RecipientEmail); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "test email sent"})
}

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
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
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
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
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

	if err := h.Backup.WithContext(c.Request().Context()).RestoreBackup(uploadedBackupPath, password); err != nil {
		return writeRestoreBackupError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "backup restored - please restart server"})
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
