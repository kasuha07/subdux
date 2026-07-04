package api

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	iconproxy "github.com/kasuha07/subdux/internal/service/iconproxy"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	servicesmtp "github.com/kasuha07/subdux/internal/service/smtp"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	Service     *adminservice.Service
	TaskMonitor *serviceutil.BackgroundTaskMonitor
	Reauth      *servicereauth.Service
}

var (
	// errInvalidBackup maps malformed or unsupported restore archives to HTTP
	// 400 so callers can correct the uploaded file or password and retry.
	errInvalidBackup = servicebackup.ErrInvalidBackup
	// errBackupPasswordRequired signals that an uploaded archive is encrypted
	// but no password was supplied. Maps to HTTP 400.
	errBackupPasswordRequired = servicebackup.ErrBackupPasswordRequired
	// errBackupInvalidPassword signals that the supplied password failed to
	// decrypt an encrypted archive entry. Maps to HTTP 400.
	errBackupInvalidPassword = servicebackup.ErrBackupInvalidPassword
)

// isClientBackupError reports whether err is a client-correctable restore
// failure (missing/invalid password or a malformed archive) that should map to
// HTTP 400 rather than 500. New client-facing sentinels can be added here in
// one place as the restore flow grows.
func isClientBackupError(err error) bool {
	for _, target := range []error{
		errBackupPasswordRequired,
		errBackupInvalidPassword,
		errInvalidBackup,
	} {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

const (
	maxBackupUploadSize = 32 << 20 // 32 MiB
)

func NewAdminHandler(s *adminservice.Service, taskMonitor *serviceutil.BackgroundTaskMonitor, reauth *servicereauth.Service) *AdminHandler {
	return &AdminHandler{Service: s, TaskMonitor: taskMonitor, Reauth: reauth}
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
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list users"})
	}
	return c.JSON(http.StatusOK, mapAdminUserResponses(users))
}

func (h *AdminHandler) CreateUser(c echo.Context) error {
	var input adminservice.CreateUserInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if input.Username == "" || input.Email == "" || input.Password == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "username, email and password are required"})
	}

	if len(input.Password) < 8 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password must be at least 8 characters"})
	}
	if err := serviceauth.ValidateBcryptPasswordLength(input.Password); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "password must not exceed 72 bytes"})
	}

	if operation, ok := servicereauth.OperationForCreateUserRole(input.Role); ok {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			getUserID(c),
			operation,
			reauthTicketFromRequest(c),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	user, err := h.Service.WithContext(c.Request().Context()).CreateUser(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, mapAdminUserResponse(adminservice.AdminUserListItem{User: *user}))
}

func (h *AdminHandler) ChangeUserRole(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	var input adminservice.ChangeRoleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if input.Role != "admin" && input.Role != "user" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid role"})
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		getUserID(c),
		servicereauth.ReauthOperationChangeUserRole,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserRole(uint(userID), input.Role); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "role updated"})
}

func (h *AdminHandler) ChangeUserStatus(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	var input adminservice.ChangeStatusInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := h.Service.WithContext(c.Request().Context()).ChangeUserStatus(uint(userID), input.Status); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "status updated"})
}

func (h *AdminHandler) DisableUserTOTP(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserTOTP(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "two-factor authentication disabled"})
}

func (h *AdminHandler) DisableUserPasskeys(c echo.Context) error {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user id"})
	}

	if err := h.Service.WithContext(c.Request().Context()).DisableUserPasskeys(uint(userID)); err != nil {
		return writeAdminCredentialResetError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkeys disabled"})
}

func writeAdminCredentialResetError(c echo.Context, err error) error {
	if errors.Is(err, serviceauth.ErrUserNotFound) ||
		errors.Is(err, adminservice.ErrAdminCredentialResetForbidden) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return writeInternalServerError(c, err)
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

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		currentUserID,
		servicereauth.ReauthOperationDeleteUser,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteUser(uint(userID)); err != nil {
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "user deleted"})
}

func (h *AdminHandler) ListBackgroundTasks(c echo.Context) error {
	return c.JSON(http.StatusOK, h.TaskMonitor.List())
}

func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.Service.WithContext(c.Request().Context()).GetSettings()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get settings"})
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var input adminservice.UpdateSettingsInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
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
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "reauthentication service is not configured"})
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			getUserID(c),
			operation,
			reauthTicketFromRequest(c),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	if err := h.Service.WithContext(c.Request().Context()).UpdateSettings(input); err != nil {
		if errors.Is(err, serviceauth.ErrInvalidEmailDomainWhitelist) ||
			errors.Is(err, serviceauth.ErrEmailDomainWhitelistTooLong) ||
			errors.Is(err, iconproxy.ErrInvalidIconProxyDomainWhitelist) ||
			errors.Is(err, iconproxy.ErrIconProxyDomainWhitelistTooLong) ||
			errors.Is(err, servicesmtp.ErrInvalidSMTPRateLimit) ||
			errors.Is(err, serviceoutbound.ErrInvalidSystemProxyType) ||
			errors.Is(err, serviceoutbound.ErrInvalidSystemProxyURL) ||
			errors.Is(err, serviceoutbound.ErrInvalidSSRFFilterMode) ||
			errors.Is(err, serviceoutbound.ErrInvalidSSRFDomainFilterList) ||
			errors.Is(err, serviceoutbound.ErrSSRFDomainFilterListTooLong) ||
			errors.Is(err, serviceoutbound.ErrInvalidSSRFIPFilterList) ||
			errors.Is(err, serviceoutbound.ErrSSRFIPFilterListTooLong) ||
			errors.Is(err, servicebackup.ErrInvalidBackupTimeOfDay) ||
			errors.Is(err, servicebackup.ErrInvalidBackupRetentionCount) ||
			errors.Is(err, servicebackup.ErrInvalidBackupLocalDir) ||
			errors.Is(err, servicebackup.ErrBackupEncryptionPasswordRequired) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "settings updated"})
}

func (h *AdminHandler) TestSSRF(c echo.Context) error {
	var input adminservice.SSRFTestInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	result, err := h.Service.WithContext(c.Request().Context()).TestSSRF(input)
	if err != nil {
		if errors.Is(err, serviceoutbound.ErrInvalidSSRFTestTarget) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AdminHandler) TestSMTP(c echo.Context) error {
	var input struct {
		RecipientEmail string `json:"recipient_email"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	currentUserID := getUserID(c)

	if err := h.Service.WithContext(c.Request().Context()).SendSMTPTestEmail(currentUserID, input.RecipientEmail); err != nil {
		if errors.Is(err, servicesmtp.ErrSMTPRateLimited) {
			return c.JSON(http.StatusTooManyRequests, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "test email sent"})
}

func (h *AdminHandler) BackupDB(c echo.Context) error {
	var input struct {
		IncludeAssets bool   `json:"include_assets"`
		Password      string `json:"password"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		getUserID(c),
		servicereauth.ReauthOperationBackup,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	backupPath, err := servicebackup.NewService(h.Service.DB.WithContext(c.Request().Context())).BackupDB(input.IncludeAssets, input.Password)
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

func (h *AdminHandler) RunBackupNow(c echo.Context) error {
	backupPath, err := servicebackup.NewService(h.Service.DB.WithContext(c.Request().Context())).CreateLocalBackup()
	if err != nil {
		if errors.Is(err, servicebackup.ErrBackupEncryptionPasswordRequired) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
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
	directory, backups, err := servicebackup.NewService(h.Service.DB.WithContext(c.Request().Context())).ListLocalBackups()
	if err != nil {
		return writeInternalServerError(c, err)
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
		getUserID(c),
		servicereauth.ReauthOperationRestore,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

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

	password := strings.TrimSpace(c.FormValue("password"))

	if err := servicebackup.NewService(h.Service.DB.WithContext(c.Request().Context())).RestoreBackup(uploadedBackupPath, password); err != nil {
		if isClientBackupError(err) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to restore backup"})
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
