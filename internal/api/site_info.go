package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type SiteInfoHandler struct {
	Service *service.SystemSettingsService
}

func NewSiteInfoHandler(s *service.SystemSettingsService) *SiteInfoHandler {
	return &SiteInfoHandler{Service: s}
}

func (h *SiteInfoHandler) Get(c echo.Context) error {
	siteInfo, err := h.Service.WithContext(c.Request().Context()).GetSiteInfo()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to get site info"})
	}
	return c.JSON(http.StatusOK, siteInfo)
}
