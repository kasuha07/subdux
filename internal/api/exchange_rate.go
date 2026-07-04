package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/model"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	"github.com/labstack/echo/v4"
)

type ExchangeRateHandler struct {
	Service *exchangerate.Service
}

type userPreferenceResponse struct {
	PreferredCurrency string `json:"preferred_currency"`
}

func mapUserPreferenceResponse(pref model.UserPreference) userPreferenceResponse {
	return userPreferenceResponse{
		PreferredCurrency: pref.PreferredCurrency,
	}
}

func NewExchangeRateHandler(s *exchangerate.Service) *ExchangeRateHandler {
	return &ExchangeRateHandler{Service: s}
}

func (h *ExchangeRateHandler) ListRates(c echo.Context) error {
	rates, err := h.Service.WithContext(c.Request().Context()).ListRates()
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, rates)
}

func (h *ExchangeRateHandler) GetRate(c echo.Context) error {
	base := c.Param("base")
	target := c.Param("target")

	if base == "" || target == "" {
		return writeError(c, http.StatusBadRequest, "base and target currencies are required")
	}

	rate, ok := h.Service.WithContext(c.Request().Context()).GetRate(base, target)
	if !ok {
		return writeError(c, http.StatusNotFound, "exchange rate not found")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"rate": rate,
	})
}

func (h *ExchangeRateHandler) GetStatus(c echo.Context) error {
	status, err := h.Service.WithContext(c.Request().Context()).GetStatus()
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, status)
}

func (h *ExchangeRateHandler) RefreshRates(c echo.Context) error {
	if err := h.Service.WithContext(c.Request().Context()).RefreshRates(); err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "rates refreshed"})
}

func (h *ExchangeRateHandler) GetPreference(c echo.Context) error {
	userID := getUserID(c)
	pref, err := h.Service.WithContext(c.Request().Context()).GetUserPreference(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapUserPreferenceResponse(*pref))
}

func (h *ExchangeRateHandler) UpdatePreference(c echo.Context) error {
	userID := getUserID(c)
	var input exchangerate.UpdatePreferenceInput
	if !bindJSON(c, &input, "Invalid request body") {
		return nil
	}

	if input.PreferredCurrency == "" {
		return writeError(c, http.StatusBadRequest, "preferred_currency is required")
	}

	pref, err := h.Service.WithContext(c.Request().Context()).UpdateUserPreference(userID, input)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapUserPreferenceResponse(*pref))
}
