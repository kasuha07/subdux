package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
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
		return err
	}
	return c.JSON(http.StatusOK, rates)
}

func (h *ExchangeRateHandler) GetRate(c echo.Context) error {
	base := c.Param("base")
	target := c.Param("target")

	if base == "" || target == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "base_and_target_currencies_are_required")
	}

	rate, ok := h.Service.WithContext(c.Request().Context()).GetRate(base, target)
	if !ok {
		return httpx.WriteError(c, http.StatusNotFound, "exchange_rate_not_found")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"rate": rate,
	})
}

func (h *ExchangeRateHandler) GetStatus(c echo.Context) error {
	status, err := h.Service.WithContext(c.Request().Context()).GetStatus()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, status)
}

func (h *ExchangeRateHandler) RefreshRates(c echo.Context) error {
	if err := h.Service.WithContext(c.Request().Context()).RefreshRates(); err != nil {
		return err
	}
	return httpx.WriteMessage(c, http.StatusOK, "rates_refreshed")
}

func (h *ExchangeRateHandler) GetPreference(c echo.Context) error {
	userID := apimw.From(c).UserID
	pref, err := h.Service.WithContext(c.Request().Context()).GetUserPreference(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapUserPreferenceResponse(*pref))
}

func (h *ExchangeRateHandler) UpdatePreference(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input exchangerate.UpdatePreferenceInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	if input.PreferredCurrency == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "preferred_currency_is_required")
	}

	pref, err := h.Service.WithContext(c.Request().Context()).UpdateUserPreference(userID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapUserPreferenceResponse(*pref))
}
