package api

import (
	"errors"
	"net/http"
	"strconv"

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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name is required"})
	}
	category, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		if err.Error() == "category name already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "name must be 1-30 characters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusCreated, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	var input catalogservice.UpdateCategoryInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	category, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		if err.Error() == "category not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		if err.Error() == "category name already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "name must be 1-30 characters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapCategoryResponse(*category))
}

func (h *CategoryHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		if err.Error() == "category not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		if errors.Is(err, catalogservice.ErrCategoryInUse) {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CategoryHandler) Reorder(c echo.Context) error {
	userID := getUserID(c)
	var items []catalogservice.ReorderItem
	if err := c.Bind(&items); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if err := h.Service.WithContext(c.Request().Context()).Reorder(userID, items); err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reordered"})
}
