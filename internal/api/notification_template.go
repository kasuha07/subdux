package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type NotificationTemplateHandler struct {
	Service *service.NotificationTemplateService
}

func NewNotificationTemplateHandler(s *service.NotificationTemplateService) *NotificationTemplateHandler {
	return &NotificationTemplateHandler{Service: s}
}

func (h *NotificationTemplateHandler) ListTemplates(c echo.Context) error {
	userID := getUserID(c)
	templates, err := h.Service.ListTemplates(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, templates)
}

func (h *NotificationTemplateHandler) GetTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	template, err := h.Service.GetTemplate(userID, uint(id))
	if err != nil {
		if err.Error() == "template not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) CreateTemplate(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateTemplateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	template, err := h.Service.CreateTemplate(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, template)
}

func (h *NotificationTemplateHandler) UpdateTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	var input service.UpdateTemplateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	template, err := h.Service.UpdateTemplate(userID, uint(id), input)
	if err != nil {
		if err.Error() == "template not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, template)
}

func (h *NotificationTemplateHandler) DeleteTemplate(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	if err := h.Service.DeleteTemplate(userID, uint(id)); err != nil {
		if err.Error() == "template not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationTemplateHandler) PreviewTemplate(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateTemplateInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	preview, err := h.Service.PreviewTemplate(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"preview": preview})
}
