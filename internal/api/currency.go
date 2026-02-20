package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

type CurrencyHandler struct {
	Service   *service.CurrencyService
	ERService *service.ExchangeRateService
}

type userCurrencyResponse struct {
	ID        uint   `json:"id"`
	Code      string `json:"code"`
	Symbol    string `json:"symbol"`
	Alias     string `json:"alias"`
	SortOrder int    `json:"sort_order"`
}

func mapUserCurrencyResponse(currency model.UserCurrency) userCurrencyResponse {
	return userCurrencyResponse{
		ID:        currency.ID,
		Code:      currency.Code,
		Symbol:    currency.Symbol,
		Alias:     currency.Alias,
		SortOrder: currency.SortOrder,
	}
}

func mapUserCurrencyResponses(currencies []model.UserCurrency) []userCurrencyResponse {
	responses := make([]userCurrencyResponse, len(currencies))
	for i, currency := range currencies {
		responses[i] = mapUserCurrencyResponse(currency)
	}
	return responses
}

func NewCurrencyHandler(s *service.CurrencyService, er *service.ExchangeRateService) *CurrencyHandler {
	return &CurrencyHandler{Service: s, ERService: er}
}

func (h *CurrencyHandler) List(c echo.Context) error {
	userID := getUserID(c)
	currencies, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponses(currencies))
}

func (h *CurrencyHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateCurrencyInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if input.Code == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "code is required"})
	}
	currency, err := h.Service.Create(userID, input)
	if err != nil {
		if err.Error() == "currency code already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "code must be 1-10 characters" || err.Error() == "code must contain only uppercase letters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	var input service.UpdateCurrencyInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	currency, err := h.Service.Update(userID, uint(id), input)
	if err != nil {
		if err.Error() == "currency not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	pref, err := h.ERService.GetUserPreference(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	if err := h.Service.Delete(userID, uint(id), pref.PreferredCurrency); err != nil {
		if err.Error() == "currency not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		if err.Error() == "cannot delete your preferred currency" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CurrencyHandler) Reorder(c echo.Context) error {
	userID := getUserID(c)
	var items []service.ReorderItem
	if err := c.Bind(&items); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if err := h.Service.Reorder(userID, items); err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reordered"})
}
