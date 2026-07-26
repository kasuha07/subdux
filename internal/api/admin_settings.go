package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func (h *AdminHandler) ListBackgroundTasks(c echo.Context) error {
	return c.JSON(http.StatusOK, h.TaskMonitor.List())
}

func (h *AdminHandler) GetSettings(c echo.Context) error {
	settings, err := h.Service.WithContext(c.Request().Context()).GetSettings()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_get_settings")
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *AdminHandler) UpdateSettings(c echo.Context) error {
	var input adminservice.UpdateSettingsInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if operation, ok := servicereauth.OperationForAdminSettingsUpdate(servicereauth.AdminSettingsUpdateInput{
		BackupScheduleEnabled:    input.BackupScheduleEnabled,
		BackupTimeOfDay:          input.BackupTimeOfDay,
		BackupIncludeAssets:      input.BackupIncludeAssets,
		BackupEncryptEnabled:     input.BackupEncryptEnabled,
		BackupEncryptionPassword: input.BackupEncryptionPassword,
	}); ok {
		if h.Reauth == nil {
			return httpx.WriteError(c, http.StatusInternalServerError, "reauthentication_service_is_not_configured")
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			operation,
			apimw.ReauthTicketFromRequest(c),
		); err != nil {
			return apimw.WriteReauthError(c, err)
		}
	}

	if err := h.Service.WithContext(c.Request().Context()).UpdateSettings(input); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "settings_updated")
}

func (h *AdminHandler) TestSSRF(c echo.Context) error {
	var input adminservice.SSRFTestInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
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
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	currentUserID := apimw.From(c).UserID

	if err := h.Service.WithContext(c.Request().Context()).SendSMTPTestEmail(currentUserID, input.RecipientEmail); err != nil {
		return err
	}

	return httpx.WriteMessage(c, http.StatusOK, "test_email_sent")
}
