package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apicontract"
	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/labstack/echo/v4"
)

type CategoryHandler struct {
	Service *catalogservice.CategoryService
}

type categoryResponse = apicontract.CategoryResponse

func mapCategoryResponse(category model.Category) categoryResponse {
	return apicontract.MapCategoryResponse(category)
}

func mapCategoryResponses(categories []model.Category) []categoryResponse {
	return apicontract.MapCategoryResponses(categories)
}

func NewCategoryHandler(s *catalogservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{Service: s}
}

func (h *CategoryHandler) List(c echo.Context) error {
	userID := getUserID(c)
	categories, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapCategoryResponses(categories))
}

func (h *CategoryHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input catalogservice.CreateCategoryInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}
	if input.Name == "" {
		return writeError(c, http.StatusBadRequest, "name is required")
	}
	category, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusConflict, "category name already exists"),
			serviceErrorMessage(http.StatusBadRequest, "name must be 1-30 characters"),
		)
	}
	return c.JSON(http.StatusCreated, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}
	var input catalogservice.UpdateCategoryInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}
	category, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "category not found"),
			serviceErrorMessage(http.StatusConflict, "category name already exists"),
			serviceErrorMessage(http.StatusBadRequest, "name must be 1-30 characters"),
		)
	}
	return c.JSON(http.StatusOK, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}
	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "category not found"),
			serviceError(http.StatusBadRequest, catalogservice.ErrCategoryInUse),
		)
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CategoryHandler) Reorder(c echo.Context) error {
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
