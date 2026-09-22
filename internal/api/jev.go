package api

import (
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/service/jev"
	"github.com/labstack/echo/v4"
)

type JevHandler struct{ Service *jev.Service }

func (h *JevHandler) RegisterRoutes(g RouteGroups) {
	limit := apimw.AuthAccountRateLimit(30, time.Minute, apimw.AuthenticatedUserAccountKey)
	testLimit := apimw.AuthAccountRateLimit(5, time.Minute, apimw.AuthenticatedUserAccountKey)
	bodyLimit := apimw.RequestBodyLimitMiddleware(8<<10, nil)
	g.HumanProtected.GET("/jev/settings", h.GetSettings)
	g.HumanProtected.PUT("/jev/settings", h.UpdateSettings, bodyLimit)
	g.HumanProtected.POST("/jev/test-connection", h.TestConnection, testLimit)
	g.HumanProtected.POST("/jev/category-suggestion", h.Suggest, bodyLimit, limit)
}

func (h *JevHandler) GetSettings(c echo.Context) error {
	settings, err := h.Service.GetSettings(c.Request().Context(), apimw.From(c).UserID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *JevHandler) UpdateSettings(c echo.Context) error {
	var input jev.UpdateInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	settings, err := h.Service.UpdateSettings(c.Request().Context(), apimw.From(c).UserID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *JevHandler) TestConnection(c echo.Context) error {
	settings, err := h.Service.TestConnection(c.Request().Context(), apimw.From(c).UserID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, settings)
}

func (h *JevHandler) Suggest(c echo.Context) error {
	var input jev.SuggestInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	suggestion, err := h.Service.Suggest(c.Request().Context(), apimw.From(c).UserID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, suggestion)
}
