package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type CategoryHandler struct {
	Service *service.CategoryService
}

func NewCategoryHandler(s *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{Service: s}
}

func (h *CategoryHandler) List(c echo.Context) error {
	userID := getUserID(c)
	categories, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, categories)
}

func (h *CategoryHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateCategoryInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name is required"})
	}
	category, err := h.Service.Create(userID, input)
	if err != nil {
		if err.Error() == "category name already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "name must be 1-30 characters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, category)
}

func (h *CategoryHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	var input service.UpdateCategoryInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	category, err := h.Service.Update(userID, uint(id), input)
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
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, category)
}

func (h *CategoryHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}
	if err := h.Service.Delete(userID, uint(id)); err != nil {
		if err.Error() == "category not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusNoContent, nil)
}

func (h *CategoryHandler) Reorder(c echo.Context) error {
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
