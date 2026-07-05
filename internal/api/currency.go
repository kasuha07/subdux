package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
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
	userID := apimw.From(c).UserID
	currencies, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponses(currencies))
}

func (h *CurrencyHandler) Create(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input catalogservice.CreateCurrencyInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	if input.Code == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "code_is_required")
	}
	currency, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Update(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}
	var input catalogservice.UpdateCurrencyInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	currency, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapUserCurrencyResponse(*currency))
}

func (h *CurrencyHandler) Delete(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}
	ctx := c.Request().Context()
	svc := h.Service.WithContext(ctx)
	erService := h.ERService.WithContext(ctx)
	pref, err := erService.GetUserPreference(userID)
	if err != nil {
		return err
	}
	if err := svc.Delete(userID, uint(id), pref.PreferredCurrency); err != nil {
		return err
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CurrencyHandler) Reorder(c echo.Context) error {
	userID := apimw.From(c).UserID
	var items []catalogservice.ReorderItem
	if !httpx.BindJSON(c, &items, "invalid_request_body") {
		return nil
	}
	if err := h.Service.WithContext(c.Request().Context()).Reorder(userID, items); err != nil {
		return err
	}
	return httpx.WriteMessage(c, http.StatusOK, "reordered")
}
