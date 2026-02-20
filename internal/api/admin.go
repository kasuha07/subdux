package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type AdminHandler struct {
	Service *service.AdminService
}

func NewAdminHandler(s *service.AdminService) *AdminHandler {
	return &AdminHandler{Service: s}
}

func (h *AdminHandler) ListUsers(c echo.Context) error {
	users, err := h.Service.ListUsers()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list users"})
	}
	return c.JSON(http.StatusOK, users)
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
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
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
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "settings updated"})
}

func (h *AdminHandler) BackupDB(c echo.Context) error {
	backupPath, err := h.Service.BackupDB()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "backup failed"})
	}
	defer os.Remove(backupPath)

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("subdux-backup-%s.db", timestamp)

	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Response().Header().Set("Content-Type", "application/octet-stream")

	return c.File(backupPath)
}

func (h *AdminHandler) RestoreDB(c echo.Context) error {
	file, err := c.FormFile("backup")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no file uploaded"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read file"})
	}
	defer src.Close()

	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("subdux-restore-%d.db", time.Now().Unix()))
	dst, err := os.Create(tempPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create temp file"})
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tempPath)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to save file"})
	}
	dst.Close()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/subdux.db"
	}

	sqlDB, err := h.Service.DB.DB()
	if err != nil {
		os.Remove(tempPath)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to access database"})
	}
	sqlDB.Close()

	if err := os.Rename(tempPath, dbPath); err != nil {
		os.Remove(tempPath)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to restore database"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "database restored - please restart server"})
}
