package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg"
	exporter "github.com/kasuha07/subdux/internal/service/exporter"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type ExportHandler struct {
	Service *exporter.Service
	Reauth  *servicereauth.Service
}

func NewExportHandler(s *exporter.Service, reauth *servicereauth.Service) *ExportHandler {
	return &ExportHandler{Service: s, Reauth: reauth}
}

func (h *ExportHandler) Export(c echo.Context) error {
	userID := apimw.From(c).UserID
	includeSecrets := c.QueryParam("include_secrets") == "1"
	if includeSecrets && c.QueryParam("confirm") != "include_secrets" {
		return httpx.WriteError(c, http.StatusBadRequest, "exporting notification secrets requires confirmation")
	}
	operation := servicereauth.OperationForExport(includeSecrets)
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		operation,
		c.Request().Header.Get(apimw.ReauthTicketHeader),
	); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	data, err := h.Service.WithContext(c.Request().Context()).ExportUserData(userID, includeSecrets)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to encode export")
	}

	date := pkg.NowUTC().Format("2006-01-02")
	filename := fmt.Sprintf("subdux-export-%s-%s.json", data.User.Username, date)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Blob(http.StatusOK, "application/json", out)
}
