package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type PaymentMethodHandler struct {
	Service *service.PaymentMethodService
}

func NewPaymentMethodHandler(s *service.PaymentMethodService) *PaymentMethodHandler {
	return &PaymentMethodHandler{Service: s}
}

func (h *PaymentMethodHandler) List(c echo.Context) error {
	userID := getUserID(c)
	methods, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, methods)
}

func (h *PaymentMethodHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreatePaymentMethodInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name is required"})
	}
	if !validateIcon(input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid icon value"})
	}

	method, err := h.Service.Create(userID, input)
	if err != nil {
		if err.Error() == "payment method name already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "name must be 1-50 characters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, method)
}

func (h *PaymentMethodHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	var input service.UpdatePaymentMethodInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if input.Icon != nil && !validateIcon(*input.Icon) {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid icon value"})
	}

	method, err := h.Service.Update(userID, uint(id), input)
	if err != nil {
		if err.Error() == "payment method not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		if err.Error() == "payment method name already exists" {
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		}
		if err.Error() == "name must be 1-50 characters" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, method)
}

func (h *PaymentMethodHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	if err := h.Service.Delete(userID, uint(id)); err != nil {
		if err.Error() == "payment method not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PaymentMethodHandler) Reorder(c echo.Context) error {
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

func (h *PaymentMethodHandler) UploadIcon(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "no file provided"})
	}

	src, err := fileHeader.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to read file"})
	}
	defer src.Close()

	maxSize := h.Service.GetMaxIconFileSize()
	iconPath, err := h.Service.UploadPaymentMethodIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		msg := err.Error()
		if msg == "only PNG and JPG images are supported" || msg == "file size exceeds limit" || msg == "payment method not found" {
			return c.JSON(http.StatusBadRequest, echo.Map{"error": msg})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": msg})
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}
