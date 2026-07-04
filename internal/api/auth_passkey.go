package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type passkeyBeginRegistrationInput struct {
	Name string `json:"name"`
}

func (h *AuthHandler) ListPasskeys(c echo.Context) error {
	userID := apimw.From(c).UserID
	passkeys, err := h.Service.WithContext(c.Request().Context()).ListPasskeys(userID)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list passkeys")
	}
	return c.JSON(http.StatusOK, passkeys)
}

func (h *AuthHandler) BeginPasskeyRegistration(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input passkeyBeginRegistrationInput
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginPasskeyRegistration(userID, input.Name, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishRegistrationInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyRegistration(c echo.Context) error {
	userID := apimw.From(c).UserID
	var input passkeyFinishRegistrationInput
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "session_id and credential are required")
	}

	// Registering a passkey is a sensitive account change: a proven-present user
	// must back it, so an attacker who steals a live session cannot silently
	// enroll their own authenticator. The single-use ticket is consumed before
	// the credential is validated/persisted.
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationAddPasskey,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid credential payload")
	}

	passkey, err := h.Service.WithContext(c.Request().Context()).FinishPasskeyRegistration(userID, input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusCreated, passkey)
}

func (h *AuthHandler) DeletePasskey(c echo.Context) error {
	userID := apimw.From(c).UserID
	passkeyID, ok := httpx.ParseUintParam(c, "id", "invalid passkey id")
	if !ok {
		return nil
	}

	// Deleting a passkey is a sensitive account change. ReauthService owns the
	// accepted factor policy; the handler only consumes the operation ticket.
	if err := h.Reauth.WithContext(c.Request().Context()).Consume(
		userID,
		servicereauth.ReauthOperationDeletePasskey,
		reauthTicketFromRequest(c),
	); err != nil {
		return writeReauthError(c, err)
	}

	if err := h.Service.WithContext(c.Request().Context()).DeletePasskey(userID, uint(passkeyID)); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "passkey deleted"})
}

func (h *AuthHandler) BeginPasskeyLogin(c echo.Context) error {
	result, err := h.Service.WithContext(c.Request().Context()).BeginPasskeyLogin(c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type passkeyFinishLoginInput struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *AuthHandler) FinishPasskeyLogin(c echo.Context) error {
	var input passkeyFinishLoginInput
	if !httpx.BindJSON(c, &input, "Invalid request body") {
		return nil
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "session_id and credential are required")
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid credential payload")
	}

	resp, err := h.Service.WithContext(c.Request().Context()).FinishPasskeyLogin(input.SessionID, parsedResponse, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme())
	if err != nil {
		apimw.ClearRefreshTokenCookie(c)
		return httpx.WriteError(c, http.StatusUnauthorized, err.Error())
	}

	return writeAuthSuccess(c, http.StatusOK, resp)
}
