package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type ImportHandler struct {
	Service *service.ImportService
}

func NewImportHandler(s *service.ImportService) *ImportHandler {
	return &ImportHandler{Service: s}
}

func (h *ImportHandler) ImportWallos(c echo.Context) error {
	userID := getUserID(c)

	var data []service.WallosSubscription
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid JSON"})
	}

	result, err := h.Service.ImportFromWallos(userID, data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}
