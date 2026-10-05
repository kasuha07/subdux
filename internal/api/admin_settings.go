package api

import (
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
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
	if input.Revisions == nil {
		return serviceutil.ErrRevisionRequired
	}

	// Settings that decide which identity provider is trusted, how strong its
	// logins count, where OIDC credentials are sent, or how outbound requests
	// are filtered need step-up: a stolen session alone must not be able to
	// point OIDC at an attacker-controlled issuer and sign in as linked users.
	// The service decides whether the update changes one of them and calls back
	// only then, after validation, so re-sent saved values and invalid input
	// never spend a ticket.
	authorizeSecurityChange := func() error {
		if h.Reauth == nil {
			return serviceerr.New(serviceerr.KindInternal, "reauthentication_service_is_not_configured", "reauthentication service is not configured")
		}
		return h.Reauth.WithContext(c.Request().Context()).Consume(
			apimw.From(c).UserID,
			servicereauth.ReauthOperationAdminSecuritySettings,
			apimw.ReauthTicketFromRequest(c),
		)
	}
	if err := h.Service.WithContext(c.Request().Context()).UpdateSettings(input, authorizeSecurityChange); err != nil {
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
