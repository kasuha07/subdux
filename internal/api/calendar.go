package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type CalendarHandler struct {
	Service *service.CalendarService
}

func NewCalendarHandler(s *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{Service: s}
}

func (h *CalendarHandler) ListTokens(c echo.Context) error {
	userID := getUserID(c)
	tokens, err := h.Service.ListTokens(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Name is required"})
	}
	if len(input.Name) > 100 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Name must be 100 characters or less"})
	}

	existing, err := h.Service.ListTokens(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if len(existing) >= 5 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Maximum of 5 calendar links reached"})
	}

	token, err := h.Service.GenerateToken(userID, input.Name)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, token)
}

func (h *CalendarHandler) DeleteToken(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid ID"})
	}

	if err := h.Service.DeleteToken(userID, uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *CalendarHandler) GetCalendarFeed(c echo.Context) error {
	tokenStr := c.QueryParam("token")
	if tokenStr == "" {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "token is required"})
	}

	userID, err := h.Service.ValidateToken(tokenStr)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid or expired token"})
	}

	ics, err := h.Service.GenerateICalFeed(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	c.Response().Header().Set("Content-Type", "text/calendar; charset=utf-8")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="subdux.ics"`)
	return c.String(http.StatusOK, ics)
}
