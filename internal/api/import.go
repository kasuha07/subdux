package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

const maxImportRequestBodyBytes int64 = 2 * 1024 * 1024

type ImportHandler struct {
	Service *service.ImportService
	Reauth  *service.ReauthService
}

func NewImportHandler(s *service.ImportService, reauth *service.ReauthService) *ImportHandler {
	return &ImportHandler{Service: s, Reauth: reauth}
}

func (h *ImportHandler) ImportWallos(c echo.Context) error {
	userID := getUserID(c)
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req service.WallosImportRequest
	if err := c.Bind(&req); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return c.JSON(http.StatusRequestEntityTooLarge, echo.Map{"error": "import file is too large"})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid JSON"})
	}
	if req.Confirm {
		if getAuthType(c) == pkg.AuthTypeAPIKey {
			return c.JSON(http.StatusForbidden, echo.Map{"error": "human session required"})
		}
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			service.ReauthOperationImportWallos,
			c.Request().Header.Get(reauthTicketHeader),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromWallos(userID, req.Data, req.Confirm)
	if err != nil {
		if errors.Is(err, service.ErrWallosImportTooLarge) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}

func (h *ImportHandler) ImportSubdux(c echo.Context) error {
	userID := getUserID(c)
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, maxImportRequestBodyBytes)

	var req service.SubduxImportRequest
	if err := c.Bind(&req); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			return c.JSON(http.StatusRequestEntityTooLarge, echo.Map{"error": "import file is too large"})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid JSON"})
	}
	if req.Confirm {
		if err := h.Reauth.WithContext(c.Request().Context()).Consume(
			userID,
			service.ReauthOperationImportSubdux,
			c.Request().Header.Get(reauthTicketHeader),
		); err != nil {
			return writeReauthError(c, err)
		}
	}

	result, err := h.Service.WithContext(c.Request().Context()).ImportFromSubdux(userID, req.Data, req.Confirm)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSubduxImportFormat) || errors.Is(err, service.ErrSubduxImportTooLarge) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}

	return c.JSON(http.StatusOK, result)
}
