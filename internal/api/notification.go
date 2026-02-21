package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type NotificationHandler struct {
	Service *service.NotificationService
}

func NewNotificationHandler(s *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{Service: s}
}

func (h *NotificationHandler) ListChannels(c echo.Context) error {
	userID := getUserID(c)
	channels, err := h.Service.ListChannels(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, channels)
}

func (h *NotificationHandler) CreateChannel(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateChannelInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	if input.Type == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "type is required"})
	}

	channel, err := h.Service.CreateChannel(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, channel)
}

func (h *NotificationHandler) UpdateChannel(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	var input service.UpdateChannelInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	channel, err := h.Service.UpdateChannel(userID, uint(id), input)
	if err != nil {
		if err.Error() == "channel not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, channel)
}

func (h *NotificationHandler) DeleteChannel(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	if err := h.Service.DeleteChannel(userID, uint(id)); err != nil {
		if err.Error() == "channel not found" {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *NotificationHandler) TestChannel(c echo.Context) error {
	userID := getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid id"})
	}

	if err := h.Service.TestChannel(userID, uint(id)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, echo.Map{"message": "test notification sent"})
}

func (h *NotificationHandler) GetPolicy(c echo.Context) error {
	userID := getUserID(c)
	policy, err := h.Service.GetPolicy(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, policy)
}

func (h *NotificationHandler) UpdatePolicy(c echo.Context) error {
	userID := getUserID(c)
	var input service.UpdatePolicyInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}

	policy, err := h.Service.UpdatePolicy(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, policy)
}

func (h *NotificationHandler) ListLogs(c echo.Context) error {
	userID := getUserID(c)
	limit := 50
	if q := c.QueryParam("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			limit = v
		}
	}

	logs, err := h.Service.ListLogs(userID, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, logs)
}
