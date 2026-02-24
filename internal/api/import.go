package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

const maxImportRequestBodyBytes int64 = 2 * 1024 * 1024

type ImportHandler struct {
	Service *service.ImportService
}

func NewImportHandler(s *service.ImportService) *ImportHandler {
	return &ImportHandler{Service: s}
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

	result, err := h.Service.ImportFromWallos(userID, req.Data, req.Confirm)
	if err != nil {
		if errors.Is(err, service.ErrWallosImportTooLarge) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
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

	result, err := h.Service.ImportFromSubdux(userID, req.Data, req.Confirm)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSubduxImportFormat) || errors.Is(err, service.ErrSubduxImportTooLarge) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
