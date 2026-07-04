package api

import (
	"net/http"

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
	userID := getUserID(c)
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req importer.WallosImportRequest
	if !bindLimitedJSON(c, &req, "import file is too large", "invalid JSON") {
		return nil
	}
	if req.Confirm {
		if getAuthType(c) == pkg.AuthTypeAPIKey {
			return writeError(c, http.StatusForbidden, "human session required")
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			servicereauth.ReauthOperationImportWallos,
			c.Request().Header.Get(reauthTicketHeader),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromWallos(userID, req.Data, req.Confirm)
	if err != nil {
		return writeServiceError(c, err,
			serviceError(http.StatusBadRequest, importer.ErrWallosImportTooLarge),
		)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *ImportHandler) ImportSubdux(c echo.Context) error {
	userID := getUserID(c)
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req importer.SubduxImportRequest
	if !bindLimitedJSON(c, &req, "import file is too large", "invalid JSON") {
		return nil
	}
	if req.Confirm {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			servicereauth.ReauthOperationImportSubdux,
			c.Request().Header.Get(reauthTicketHeader),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromSubdux(userID, req.Data, req.Confirm)
	if err != nil {
		return writeServiceError(c, err,
			serviceError(http.StatusBadRequest, importer.ErrInvalidSubduxImportFormat, importer.ErrSubduxImportTooLarge),
		)
	}

	return c.JSON(http.StatusOK, result)
}
