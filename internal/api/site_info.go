package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/httpx"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"github.com/labstack/echo/v4"
)

type SiteInfoHandler struct {
	Service *systemsettings.Service
}

func NewSiteInfoHandler(s *systemsettings.Service) *SiteInfoHandler {
	return &SiteInfoHandler{Service: s}
}

func (h *SiteInfoHandler) Get(c echo.Context) error {
	siteInfo, err := h.Service.WithContext(c.Request().Context()).GetSiteInfo()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to get site info")
	}
	return c.JSON(http.StatusOK, siteInfo)
}
