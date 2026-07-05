package api

import (
	"net/http"
	"strings"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	calendarservice "github.com/kasuha07/subdux/internal/service/calendar"
	"github.com/labstack/echo/v4"
)

type CalendarHandler struct {
	Service *calendarservice.Service
}

func NewCalendarHandler(s *calendarservice.Service) *CalendarHandler {
	return &CalendarHandler{Service: s}
}

func (h *CalendarHandler) ListTokens(c echo.Context) error {
	userID := apimw.From(c).UserID
	tokens, err := h.Service.WithContext(c.Request().Context()).ListTokens(userID)
	if err != nil {
		return err
	}
	for i := range tokens {
		tokens[i].MaskToken()
	}
	return c.JSON(http.StatusOK, tokens)
}

func (h *CalendarHandler) CreateToken(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input struct {
		Name string `json:"name"`
	}
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "name_is_required")
	}
	if len(input.Name) > 100 {
		return httpx.WriteError(c, http.StatusBadRequest, "name_must_be_100_characters_or_less")
	}

	svc := h.Service.WithContext(c.Request().Context())
	existing, err := svc.ListTokens(userID)
	if err != nil {
		return err
	}
	if len(existing) >= 5 {
		return httpx.WriteError(c, http.StatusBadRequest, "maximum_of_5_calendar_links_reached")
	}

	token, err := svc.GenerateToken(userID, input.Name)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, token)
}

func (h *CalendarHandler) DeleteToken(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteToken(userID, uint(id)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *CalendarHandler) GetCalendarFeed(c echo.Context) error {
	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return httpx.WriteError(c, http.StatusUnauthorized, "token_is_required")
	}

	svc := h.Service.WithContext(c.Request().Context())
	userID, err := svc.ValidateToken(tokenStr)
	if err != nil {
		return httpx.WriteError(c, http.StatusUnauthorized, "invalid_or_expired_token")
	}

	ics, err := svc.GenerateICalFeed(userID)
	if err != nil {
		return err
	}

	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="subdux.ics"`)
	return c.String(http.StatusOK, ics)
}
