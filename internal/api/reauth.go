package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

// reauthTicketHeader carries the single-use step-up ticket. It lives in a
// header (rather than a request body) so sensitive endpoints can consume it
// before parsing/buffering the body — e.g. the restore upload is gated behind a
// proven-present admin before any multipart data is read.
const reauthTicketHeader = "X-Reauth-Ticket"

// ReauthHandler exposes step-up re-authentication endpoints. A client verifies
// one factor (password or passkey) for a named operation and receives a
// short-lived, single-use ticket, which it then presents to the sensitive
// endpoint (e.g. backup download / restore).
type ReauthHandler struct {
	Service *service.ReauthService
}

func NewReauthHandler(s *service.ReauthService) *ReauthHandler {
	return &ReauthHandler{Service: s}
}

// writeReauthError maps reauth service errors to HTTP 400 with the error
// message. It never returns 401, so a failed re-auth attempt does not trip the
// frontend's token-refresh/logout flow.
func writeReauthError(c echo.Context, err error) error {
	return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
}

// validateReauthOperation extracts and validates the operation identifier.
func validateReauthOperation(operation string) (string, error) {
	switch operation {
	case service.ReauthOperationBackup, service.ReauthOperationRestore:
		return operation, nil
	default:
		return "", service.ErrInvalidReauthOperation
	}
}

func (h *ReauthHandler) Methods(c echo.Context) error {
	if _, err := validateReauthOperation(c.QueryParam("operation")); err != nil {
		return writeReauthError(c, err)
	}

	methods, err := h.Service.WithContext(c.Request().Context()).AvailableMethods(getUserID(c))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to load re-authentication methods"})
	}
	return c.JSON(http.StatusOK, methods)
}

type reauthPasswordInput struct {
	Operation string `json:"operation"`
	Password  string `json:"password"`
}

func (h *ReauthHandler) VerifyPassword(c echo.Context) error {
	var input reauthPasswordInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return writeReauthError(c, err)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyPassword(getUserID(c), operation, input.Password)
	if err != nil {
		return writeReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthPasskeyStartInput struct {
	Operation string `json:"operation"`
}

func (h *ReauthHandler) BeginPasskey(c echo.Context) error {
	var input reauthPasskeyStartInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return writeReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginPasskey(
		getUserID(c), operation, c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return writeReauthError(c, err)
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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return writeReauthError(c, err)
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "session_id and credential are required"})
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid credential payload"})
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).FinishPasskey(
		getUserID(c), operation, input.SessionID, parsedResponse,
		c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return writeReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthOIDCStartInput struct {
	Operation string `json:"operation"`
}

// BeginOIDC starts an OIDC step-up for the operation and returns the provider
// authorization URL. The client opens it in a popup; the callback lands on the
// admin page (see OIDCCallback) which posts the result back to the opener.
func (h *ReauthHandler) BeginOIDC(c echo.Context) error {
	var input reauthOIDCStartInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return writeReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginOIDC(getUserID(c), operation)
	if err != nil {
		return writeReauthError(c, err)
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
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request body"})
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return writeReauthError(c, err)
	}

	sessionID := getCookieValue(c, oidcReauthSessionCookieName)
	clearOIDCReauthSessionCookie(c)
	if sessionID == "" {
		return writeReauthError(c, service.ErrReauthRequired)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyOIDC(getUserID(c), operation, sessionID)
	if err != nil {
		return writeReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}
