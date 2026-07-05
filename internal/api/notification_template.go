package api

import (
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
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
	userID := apimw.From(c).UserID
	templates, err := h.Service.WithContext(c.Request().Context()).ListTemplates(userID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, templates)
}

func (h *NotificationTemplateHandler) GetTemplate(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).GetTemplate(userID, uint(id))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) CreateTemplate(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input notificationservice.CreateTemplateInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).CreateTemplate(userID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, template)
}

func (h *NotificationTemplateHandler) UpdateTemplate(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	var input notificationservice.UpdateTemplateInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	template, err := h.Service.WithContext(c.Request().Context()).UpdateTemplate(userID, uint(id), input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) DeleteTemplate(c echo.Context) error {
	userID := apimw.From(c).UserID
	id, ok := httpx.ParseUintParam(c, "id", "invalid_id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteTemplate(userID, uint(id)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationTemplateHandler) PreviewTemplate(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input notificationservice.CreateTemplateInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}

	preview, err := h.Service.WithContext(c.Request().Context()).PreviewTemplate(userID, input)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, echo.Map{"preview": preview})
}
