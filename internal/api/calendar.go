package api

import (
	"net/http"
	"strings"

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
	userID := getUserID(c)
	tokens, err := h.Service.WithContext(c.Request().Context()).ListTokens(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	for i := range tokens {
		tokens[i].MaskToken()
	}
	return c.JSON(http.StatusOK, tokens)
}

func (h *CalendarHandler) CreateToken(c echo.Context) error {
	userID := getUserID(c)
	var input struct {
		Name string `json:"name"`
	}
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return writeError(c, http.StatusBadRequest, "Name is required")
	}
	if len(input.Name) > 100 {
		return writeError(c, http.StatusBadRequest, "Name must be 100 characters or less")
	}

	svc := h.Service.WithContext(c.Request().Context())
	existing, err := svc.ListTokens(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	if len(existing) >= 5 {
		return writeError(c, http.StatusBadRequest, "Maximum of 5 calendar links reached")
	}

	token, err := svc.GenerateToken(userID, input.Name)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusCreated, token)
}

func (h *CalendarHandler) DeleteToken(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "Invalid ID")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteToken(userID, uint(id)); err != nil {
		return writeInternalServerError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *CalendarHandler) GetCalendarFeed(c echo.Context) error {
	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return writeError(c, http.StatusUnauthorized, "token is required")
	}

	svc := h.Service.WithContext(c.Request().Context())
	userID, err := svc.ValidateToken(tokenStr)
	if err != nil {
		return writeError(c, http.StatusUnauthorized, "invalid or expired token")
	}

	ics, err := svc.GenerateICalFeed(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}

	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="subdux.ics"`)
	return c.String(http.StatusOK, ics)
}
