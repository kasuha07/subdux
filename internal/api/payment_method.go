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

type PaymentMethodHandler struct {
	Service *catalogservice.PaymentMethodService
}

type paymentMethodResponse = contract.PaymentMethodResponse

func mapPaymentMethodResponse(method model.PaymentMethod) paymentMethodResponse {
	return contract.MapPaymentMethodResponse(method)
}

func mapPaymentMethodResponses(methods []model.PaymentMethod) []paymentMethodResponse {
	return contract.MapPaymentMethodResponses(methods)
}

func NewPaymentMethodHandler(s *catalogservice.PaymentMethodService) *PaymentMethodHandler {
	return &PaymentMethodHandler{Service: s}
}

func (h *PaymentMethodHandler) List(c echo.Context) error {
	userID := apimw.From(c).UserID
	methods, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, mapPaymentMethodResponses(methods))
}

func (h *PaymentMethodHandler) Create(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input catalogservice.CreatePaymentMethodInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Name == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "name is required")
	}
	if !validateIcon(input.Icon) {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid icon value")
	}

	method, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, mapPaymentMethodResponse(*method))
}

func (h *PaymentMethodHandler) Update(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	var input catalogservice.UpdatePaymentMethodInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	if input.Icon != nil && !validateIcon(*input.Icon) {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid icon value")
	}

	method, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, mapPaymentMethodResponse(*method))
}

func (h *PaymentMethodHandler) Delete(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PaymentMethodHandler) Reorder(c echo.Context) error {
	userID := apimw.From(c).UserID
	var items []catalogservice.ReorderItem
	if !httpx.BindJSON(c, &items, "invalid request body") {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).Reorder(userID, items); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "reordered"})
}

func (h *PaymentMethodHandler) UploadIcon(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "no file provided")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	svc := h.Service.WithContext(c.Request().Context())
	maxSize := svc.GetMaxIconFileSize()
	iconPath, err := svc.UploadPaymentMethodIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}
