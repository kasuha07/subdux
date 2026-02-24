package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type APIKeyHandler struct {
	Service *service.APIKeyService
}

func NewAPIKeyHandler(s *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{Service: s}
}

func (h *APIKeyHandler) Create(c echo.Context) error {
	userID := getUserID(c)
	var input service.CreateAPIKeyInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Name is required"})
	}

	role := getUserRole(c)
	resp, err := h.Service.Create(userID, role, input)
	if err != nil {
		switch err {
		case service.ErrAPIKeyNameRequired, service.ErrAPIKeyNameTooLong:
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		case service.ErrAPIKeyLimitReached:
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create api key"})
		}
	}

	return c.JSON(http.StatusCreated, resp)
}

func (h *APIKeyHandler) List(c echo.Context) error {
	userID := getUserID(c)
	keys, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list api keys"})
	}
	return c.JSON(http.StatusOK, keys)
}

func (h *APIKeyHandler) Delete(c echo.Context) error {
	userID := getUserID(c)
	keyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid api key id"})
	}

	if err := h.Service.Delete(userID, uint(keyID)); err != nil {
		if err == service.ErrAPIKeyNotFound {
			return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to delete api key"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "api key deleted"})
}
