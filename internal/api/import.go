package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg"
	importer "github.com/kasuha07/subdux/internal/service/importer"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

const maxImportRequestBodyBytes int64 = 2 * 1024 * 1024

type ImportHandler struct {
	Service *importer.Service
	Reauth  *servicereauth.Service
}

func NewImportHandler(s *importer.Service, reauth *servicereauth.Service) *ImportHandler {
	return &ImportHandler{Service: s, Reauth: reauth}
}

func (h *ImportHandler) ImportWallos(c echo.Context) error {
	userID := apimw.From(c).UserID
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req importer.WallosImportRequest
	if !httpx.BindLimitedJSON(c, &req, "import_file_is_too_large", "invalid_json") {
		return nil
	}
	if req.Confirm {
		if apimw.From(c).AuthType == pkg.AuthTypeAPIKey {
			return httpx.WriteError(c, http.StatusForbidden, "human_session_required")
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			servicereauth.ReauthOperationImportWallos,
			c.Request().Header.Get(apimw.ReauthTicketHeader),
		); err != nil {
			return apimw.WriteReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromWallos(userID, req.Data, req.Confirm)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (h *ImportHandler) ImportSubdux(c echo.Context) error {
	userID := apimw.From(c).UserID
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req importer.SubduxImportRequest
	if !httpx.BindLimitedJSON(c, &req, "import_file_is_too_large", "invalid_json") {
		return nil
	}
	if req.Confirm {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			servicereauth.ReauthOperationImportSubdux,
			c.Request().Header.Get(apimw.ReauthTicketHeader),
		); err != nil {
			return apimw.WriteReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromSubdux(userID, req.Data, req.Confirm)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}
