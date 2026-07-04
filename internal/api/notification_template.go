package api

import (
	"net/http"

	notificationservice "github.com/kasuha07/subdux/internal/service/notification"
	"github.com/labstack/echo/v4"
)

type NotificationTemplateHandler struct {
	Service *notificationservice.NotificationTemplateService
}

func NewNotificationTemplateHandler(s *notificationservice.NotificationTemplateService) *NotificationTemplateHandler {
	return &NotificationTemplateHandler{Service: s}
}

func (h *NotificationTemplateHandler) ListTemplates(c echo.Context) error {
	userID := getUserID(c)
	templates, err := h.Service.WithContext(c.Request().Context()).ListTemplates(userID)
	if err != nil {
		return writeInternalServerError(c, err)
	}
	return c.JSON(http.StatusOK, templates)
}

func (h *NotificationTemplateHandler) GetTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).GetTemplate(userID, uint(id))
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "template not found"),
		)
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) CreateTemplate(c echo.Context) error {
	userID := getUserID(c)
	var input notificationservice.CreateTemplateInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).CreateTemplate(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(error) bool { return true }),
		)
	}
	return c.JSON(http.StatusCreated, template)
}

func (h *NotificationTemplateHandler) UpdateTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	var input notificationservice.UpdateTemplateInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).UpdateTemplate(userID, uint(id), input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "template not found"),
			serviceErrorFunc(http.StatusBadRequest, func(error) bool { return true }),
		)
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) DeleteTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, ok := parseUintParam(c, "id", "invalid id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteTemplate(userID, uint(id)); err != nil {
		return writeServiceError(c, err,
			serviceErrorMessage(http.StatusNotFound, "template not found"),
		)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationTemplateHandler) PreviewTemplate(c echo.Context) error {
	userID := getUserID(c)
	var input notificationservice.CreateTemplateInput
	if !bindJSON(c, &input, "invalid request body") {
		return nil
	}

	preview, err := h.Service.WithContext(c.Request().Context()).PreviewTemplate(userID, input)
	if err != nil {
		return writeServiceError(c, err,
			serviceErrorFunc(http.StatusBadRequest, func(error) bool { return true }),
		)
	}
	return c.JSON(http.StatusOK, echo.Map{"preview": preview})
}
