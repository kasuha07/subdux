package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/service"
)

type APIKeyHandler struct {
	Service *service.APIKeyService
}

func NewAPIKeyHandler(s *service.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{Service: s}
}

type apiKeyResponse struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	Scopes     []string   `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type createAPIKeyResponse struct {
	APIKey apiKeyResponse `json:"api_key"`
	Key    string         `json:"key"`
}

func mapAPIKeyResponse(key model.APIKey) apiKeyResponse {
	return apiKeyResponse{
		ID:         key.ID,
		Name:       key.Name,
		Prefix:     key.Prefix,
		Scopes:     service.ParseAPIKeyScopes(key.Scopes),
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		CreatedAt:  key.CreatedAt,
	}
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
		case service.ErrAPIKeyNameRequired, service.ErrAPIKeyNameTooLong, service.ErrAPIKeyScopeInvalid:
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		case service.ErrAPIKeyLimitReached:
			return c.JSON(http.StatusConflict, echo.Map{"error": err.Error()})
		default:
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create api key"})
		}
	}

	return c.JSON(http.StatusCreated, createAPIKeyResponse{
		APIKey: mapAPIKeyResponse(resp.APIKey),
		Key:    resp.Key,
	})
}

func (h *APIKeyHandler) List(c echo.Context) error {
	userID := getUserID(c)
	keys, err := h.Service.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list api keys"})
	}

	responses := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		responses = append(responses, mapAPIKeyResponse(key))
	}

	return c.JSON(http.StatusOK, responses)
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
