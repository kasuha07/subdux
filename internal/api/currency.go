package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	"github.com/labstack/echo/v4"
)

type CurrencyHandler struct {
	Service   *catalogservice.CurrencyService
	ERService *exchangerate.Service
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

func NewCurrencyHandler(s *catalogservice.CurrencyService, er *exchangerate.Service) *CurrencyHandler {
	return &CurrencyHandler{Service: s, ERService: er}
}

func (h *CurrencyHandler) List(c echo.Context) error {
	userID := getUserID(c)
	currencies, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponses(currencies))
}

func (h *CurrencyHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input catalogservice.CreateCurrencyInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}
	if input.Code == "" {
		return writeError(c, http.StatusBadRequest, "code is required")
	}
	currency, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusConflict, "currency code already exists"),
			serviceErrorMessage(http.StatusBadRequest, "code must be 1-10 characters", "code must contain only uppercase letters"),
		)
	}
	return c.JSON(http.StatusCreated, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}
	var input catalogservice.UpdateCurrencyInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}
	currency, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "currency not found"),
		)
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}
	ctx := c.Request().Context()
	svc := h.Service.WithContext(ctx)
	erService := h.ERService.WithContext(ctx)
	pref, err := erService.GetUserPreference(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	if err := svc.Delete(userID, uint(id), pref.PreferredCurrency); err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "currency not found"),
			serviceErrorMessage(http.StatusBadRequest, "cannot delete your preferred currency"),
			serviceError(http.StatusBadRequest, catalogservice.ErrCurrencyInUse),
		)
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CurrencyHandler) Reorder(c echo.Context) error {
	userID := getUserID(c)
	var items []catalogservice.ReorderItem
	if !bindJSON(c, &items, "invalid request body") {
		return nil
	}
	if err := h.Service.WithContext(c.Request().Context()).Reorder(userID, items); err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reordered"})
}
