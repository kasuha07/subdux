package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/contract"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/labstack/echo/v4"
)

type CategoryHandler struct {
	Service *catalogservice.CategoryService
}

type categoryResponse = contract.CategoryResponse

func mapCategoryResponse(category model.Category) categoryResponse {
	return contract.MapCategoryResponse(category)
}

func mapCategoryResponses(categories []model.Category) []categoryResponse {
	return contract.MapCategoryResponses(categories)
}

func NewCategoryHandler(s *catalogservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{Service: s}
}

func (h *CategoryHandler) List(c echo.Context) error {
	userID := apimw.From(c).UserID
	categories, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapCategoryResponses(categories))
}

func (h *CategoryHandler) Create(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input catalogservice.CreateCategoryInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	if input.Name == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "name_is_required")
	}
	category, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Update(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}
	var input catalogservice.UpdateCategoryInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	category, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Delete(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}
	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return err
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CategoryHandler) Reorder(c echo.Context) error {
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
