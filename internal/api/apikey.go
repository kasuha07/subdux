package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/model"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type APIKeyHandler struct {
	Service *apikeyservice.Service
	Reauth  *servicereauth.Service
}

func NewAPIKeyHandler(s *apikeyservice.Service, reauth *servicereauth.Service) *APIKeyHandler {
	return &APIKeyHandler{Service: s, Reauth: reauth}
}

type apiKeyResponse struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	KeyKind    string     `json:"key_kind"`
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
		KeyKind:    apikeyservice.NormalizePersistedAPIKeyKind(key.KeyKind),
		Scopes:     apikeyservice.ParseAPIKeyScopes(key.Scopes),
		LastUsedAt: key.LastUsedAt,
		ExpiresAt:  key.ExpiresAt,
		CreatedAt:  key.CreatedAt,
	}
}

func (h *APIKeyHandler) Create(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input apikeyservice.CreateInput
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}

	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return httpx.WriteError(c, http.StatusBadRequest, "Name is required")
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationCreateAPIKey,
		c.Request().Header.Get(reauthTicketHeader),
	); err != nil {
		return writeReauthError(c, err)
	}

	role := apimw.From(c).Role
	resp, err := h.Service.WithContext(c.Request().Context()).Create(userID, role, input)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, createAPIKeyResponse{
		APIKey: mapAPIKeyResponse(resp.APIKey),
		Key:    resp.Key,
	})
}

func (h *APIKeyHandler) List(c echo.Context) error {
	userID := apimw.From(c).UserID
	keys, err := h.Service.WithContext(c.Request().Context()).List(userID)
	if err != nil {
		return err
	}

	responses := make([]apiKeyResponse, 0, len(keys))
	for _, key := range keys {
		responses = append(responses, mapAPIKeyResponse(key))
	}

	return c.JSON(http.StatusOK, responses)
}

func (h *APIKeyHandler) Delete(c echo.Context) error {
	userID := apimw.From(c).UserID
	keyID, ok := httpx.ParseUintParam(c, "id", "invalid api key id")
	if !ok {
		return nil
	}

	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationDeleteAPIKey,
		c.Request().Header.Get(reauthTicketHeader),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).Delete(userID, uint(keyID)); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "api key deleted"})
}
