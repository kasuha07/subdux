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

// ReauthHandler exposes step-up re-authentication endpoints. A client verifies
// one reauth method for a named operation and receives a short-lived,
// single-use ticket, which it then presents to the sensitive endpoint (e.g.
// backup download / restore). The password method may itself require a TOTP
// code when the account has two-factor authentication enabled.
type ReauthHandler struct {
	Service *servicereauth.Service
}

func NewReauthHandler(s *servicereauth.Service) *ReauthHandler {
	return &ReauthHandler{Service: s}
}

// validateReauthOperation extracts and validates the operation identifier,
// delegating the set of valid operations to reauth.IsValidReauthOperation so
// there is a single source of truth.
func validateReauthOperation(operation string) (string, error) {
	if !servicereauth.IsValidReauthOperation(operation) {
		return "", servicereauth.ErrInvalidReauthOperation
	}
	return operation, nil
}

func (h *ReauthHandler) Methods(c echo.Context) error {
	operation, err := validateReauthOperation(c.QueryParam("operation"))
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}

	methods, err := h.Service.WithContext(c.Request().Context()).AvailableMethods(apimw.From(c).UserID, operation)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to load re-authentication methods")
	}
	return c.JSON(http.StatusOK, methods)
}

type reauthPasswordInput struct {
	Operation string `json:"operation"`
	Password  string `json:"password"`
	Code      string `json:"code"`
}

func (h *ReauthHandler) VerifyPassword(c echo.Context) error {
	var input reauthPasswordInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyPassword(
		apimw.From(c).UserID, operation, input.Password, input.Code,
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthPasskeyStartInput struct {
	Operation string `json:"operation"`
}

func (h *ReauthHandler) BeginPasskey(c echo.Context) error {
	var input reauthPasskeyStartInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginPasskey(
		apimw.From(c).UserID, operation, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

type reauthPasskeyFinishInput struct {
	Operation  string          `json:"operation"`
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *ReauthHandler) FinishPasskey(c echo.Context) error {
	var input reauthPasskeyFinishInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "session_id and credential are required")
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid credential payload")
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).FinishPasskey(
		apimw.From(c).UserID, operation, input.SessionID, parsedResponse,
		c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthOIDCStartInput struct {
	Operation string `json:"operation"`
}

// BeginOIDC starts an OIDC step-up for the operation and returns the provider
// authorization URL. The client opens it in a popup; the callback lands on the
// dedicated, operation-agnostic reauth route (see OIDCCallback) which posts the
// result back to the opener.
func (h *ReauthHandler) BeginOIDC(c echo.Context) error {
	var input reauthOIDCStartInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginOIDC(apimw.From(c).UserID, operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

type reauthOIDCFinishInput struct {
	Operation string `json:"operation"`
}

// FinishOIDC completes an OIDC step-up: it reads the reauth-scoped session cookie
// set by the callback, spends it for this user and operation, and mints a ticket.
// The cookie is always cleared, so a failed attempt cannot be replayed.
func (h *ReauthHandler) FinishOIDC(c echo.Context) error {
	var input reauthOIDCFinishInput
	if !httpx.BindJSON(c, &input, "invalid request body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}

	sessionID := apimw.GetCookieValue(c, apimw.OIDCReauthSessionCookieName)
	apimw.ClearOIDCReauthSessionCookie(c)
	if sessionID == "" {
		return apimw.WriteReauthError(c, servicereauth.ErrReauthRequired)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyOIDC(apimw.From(c).UserID, operation, sessionID)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}
