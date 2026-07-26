package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
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

	// No admin setting requires step-up re-authentication any more: the backup
	// schedule moved onto individual destinations, whose own endpoints carry the
	// destination-bound reauth tickets.
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
