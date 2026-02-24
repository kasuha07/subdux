package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type ExportHandler struct {
	Service *service.ExportService
}

func NewExportHandler(s *service.ExportService) *ExportHandler {
	return &ExportHandler{Service: s}
}

func (h *ExportHandler) Export(c echo.Context) error {
	userID := getUserID(c)

	data, err := h.Service.ExportUserData(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to encode export"})
	}

	date := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("subdux-export-%s-%s.json", data.User.Username, date)
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Blob(http.StatusOK, "application/json", out)
}
