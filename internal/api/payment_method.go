package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apicontract"
	"github.com/kasuha07/subdux/internal/model"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/labstack/echo/v4"
)

type PaymentMethodHandler struct {
	Service *catalogservice.PaymentMethodService
}

type paymentMethodResponse = apicontract.PaymentMethodResponse

func mapPaymentMethodResponse(method model.PaymentMethod) paymentMethodResponse {
	return apicontract.MapPaymentMethodResponse(method)
}

func mapPaymentMethodResponses(methods []model.PaymentMethod) []paymentMethodResponse {
	return apicontract.MapPaymentMethodResponses(methods)
}

func NewPaymentMethodHandler(s *catalogservice.PaymentMethodService) *PaymentMethodHandler {
	return &PaymentMethodHandler{Service: s}
}

func (h *PaymentMethodHandler) List(c echo.Context) error {
	userID := getUserID(c)
	methods, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, mapPaymentMethodResponses(methods))
}

func (h *PaymentMethodHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input catalogservice.CreatePaymentMethodInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}

	if input.Name == "" {
		return writeError(c, http.StatusBadRequest, "name is required")
	}
	if !validateIcon(input.Icon) {
		return writeError(c, http.StatusBadRequest, "invalid icon value")
	}

	method, err := h.Service.WithContext(c.Request().Context()).Create(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusConflict, "payment method name already exists"),
			serviceErrorMessage(http.StatusBadRequest, "name must be 1-50 characters"),
		)
	}

	return c.JSON(http.StatusCreated, mapPaymentMethodResponse(*method))
}

func (h *PaymentMethodHandler) Update(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	var input catalogservice.UpdatePaymentMethodInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}
	if input.Icon != nil && !validateIcon(*input.Icon) {
		return writeError(c, http.StatusBadRequest, "invalid icon value")
	}

	method, err := h.Service.WithContext(c.Request().Context()).Update(userID, uint(id), input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "payment method not found"),
			serviceErrorMessage(http.StatusConflict, "payment method name already exists"),
			serviceErrorMessage(http.StatusBadRequest, "name must be 1-50 characters"),
		)
	}

	return c.JSON(http.StatusOK, mapPaymentMethodResponse(*method))
}

func (h *PaymentMethodHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(id)); err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "payment method not found"),
			serviceError(http.StatusBadRequest, catalogservice.ErrPaymentMethodInUse),
		)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *PaymentMethodHandler) Reorder(c echo.Context) error {
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

func (h *PaymentMethodHandler) UploadIcon(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	fileHeader, err := c.FormFile("icon")
	if err != nil {
		return writeError(c, http.StatusBadRequest, "no file provided")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return writeError(c, http.StatusInternalServerError, "failed to read file")
	}
	defer src.Close()

	svc := h.Service.WithContext(c.Request().Context())
	maxSize := svc.GetMaxIconFileSize()
	iconPath, err := svc.UploadPaymentMethodIcon(userID, uint(id), src, fileHeader.Filename, maxSize)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusForbidden, isIconUploadForbiddenError),
			serviceErrorFunc(http.StatusBadRequest, isIconUploadBadRequestError),
		)
	}

	return c.JSON(http.StatusOK, echo.Map{"icon": iconPath})
}
