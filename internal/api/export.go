package api

import (
	"encoding/json"
	"fmt"
	"github.com/kasuha07/subdux/internal/pkg"
	"net/http"

	"github.com/kasuha07/subdux/internal/service"
	"github.com/labstack/echo/v4"
)

type ExportHandler struct {
	Service *service.ExportService
	Reauth  *service.ReauthService
}

func NewExportHandler(s *service.ExportService, reauth *service.ReauthService) *ExportHandler {
	return &ExportHandler{Service: s, Reauth: reauth}
}

func (h *ExportHandler) Export(c echo.Context) error {
	userID := getUserID(c)
	includeSecrets := c.QueryParam("include_secrets") == "1"
	if includeSecrets && c.QueryParam("confirm") != "include_secrets" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "exporting notification secrets requires confirmation"})
	}
	operation := service.ReauthOperationExportRedacted
	if includeSecrets {
		operation = service.ReauthOperationExportSecrets
	}
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		operation,
		c.Request().Header.Get(reauthTicketHeader),
	); err != nil {
		return writeReauthError(c, err)
	}

	data, err := h.Service.WithContext(c.Request().Context()).ExportUserData(userID, includeSecrets)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to encode export"})
	}

	date := pkg.NowUTC().Format("2006-01-02")
	filename := fmt.Sprintf("subdux-export-%s-%s.json", data.User.Username, date)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Blob(http.StatusOK, "application/json", out)
}
