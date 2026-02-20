package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type ExchangeRateHandler struct {
	Service *service.ExchangeRateService
}

func NewExchangeRateHandler(s *service.ExchangeRateService) *ExchangeRateHandler {
	return &ExchangeRateHandler{Service: s}
}

func (h *ExchangeRateHandler) ListRates(c echo.Context) error {
	base := c.QueryParam("base")
	rates, err := h.Service.ListRates(base)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, rates)
}

func (h *ExchangeRateHandler) GetRate(c echo.Context) error {
	base := c.Param("base")
	target := c.Param("target")

	if base == "" || target == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "base and target currencies are required"})
	}

	rate, ok := h.Service.GetRate(base, target)
	if !ok {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "exchange rate not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"base_currency":   base,
		"target_currency": target,
		"rate":            rate,
	})
}

func (h *ExchangeRateHandler) GetStatus(c echo.Context) error {
	status, err := h.Service.GetStatus()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, status)
}

func (h *ExchangeRateHandler) RefreshRates(c echo.Context) error {
	if err := h.Service.RefreshRates(); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "rates refreshed"})
}

func (h *ExchangeRateHandler) GetPreference(c echo.Context) error {
	userID := getUserID(c)
	pref, err := h.Service.GetUserPreference(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, pref)
}

func (h *ExchangeRateHandler) UpdatePreference(c echo.Context) error {
	userID := getUserID(c)
	var input service.UpdatePreferenceInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	if input.PreferredCurrency == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "preferred_currency is required"})
	}

	pref, err := h.Service.UpdateUserPreference(userID, input)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, pref)
}
