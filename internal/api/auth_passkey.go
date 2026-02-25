package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
)

type passkeyBeginRegistrationInput struct {
	Name string `json:"name"`
}

func (h *AuthHandler) ListPasskeys(c echo.Context) error {
	userID := getUserID(c)
	passkeys, err := h.Service.ListPasskeys(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list passkeys"})
	}
	return c.JSON(http.StatusOK, passkeys)
}

func (h *AuthHandler) BeginPasskeyRegistration(c echo.Context) error {
	userID := getUserID(c)
	var input passkeyBeginRegistrationInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}

	result, err := h.Service.BeginPasskeyRegistration(userID, input.Name, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishRegistrationInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyRegistration(c echo.Context) error {
	userID := getUserID(c)
	var input passkeyFinishRegistrationInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and credential are required"})
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid credential payload"})
	}

	passkey, err := h.Service.FinishPasskeyRegistration(userID, input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, passkey)
}

func (h *AuthHandler) DeletePasskey(c echo.Context) error {
	userID := getUserID(c)
	passkeyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid passkey id"})
	}

	if err := h.Service.DeletePasskey(userID, uint(passkeyID)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkey deleted"})
}

func (h *AuthHandler) BeginPasskeyLogin(c echo.Context) error {
	result, err := h.Service.BeginPasskeyLogin(c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishLoginInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyLogin(c echo.Context) error {
	var input passkeyFinishLoginInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid request body"})
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and credential are required"})
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid credential payload"})
	}

	resp, err := h.Service.FinishPasskeyLogin(input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, mapAuthResponse(resp))
}
