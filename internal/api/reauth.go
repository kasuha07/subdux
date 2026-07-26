package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
	if err := validateReauthScope(operation, reauthScopeInput{
		Operation:           operation,
		DestinationID:       parseReauthUintQuery(c.QueryParam("destination_id")),
		DestinationRevision: parseReauthUint64Query(c.QueryParam("destination_revision")),
	}); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	methods, err := h.Service.WithContext(c.Request().Context()).AvailableMethods(apimw.From(c).UserID, operation)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_load_re_authentication_methods")
	}
	return c.JSON(http.StatusOK, methods)
}

type reauthScopeInput struct {
	Operation           string `json:"operation"`
	DestinationID       uint   `json:"destination_id,omitempty"`
	DestinationRevision uint64 `json:"destination_revision,omitempty"`
}

func (input reauthScopeInput) binding() *servicereauth.TicketBinding {
	if input.DestinationID == 0 && input.DestinationRevision == 0 {
		return nil
	}
	return &servicereauth.TicketBinding{
		DestinationID:       input.DestinationID,
		DestinationRevision: input.DestinationRevision,
	}
}

func validateReauthScope(operation string, input reauthScopeInput) error {
	if operation != input.Operation {
		return servicereauth.ErrInvalidReauthOperation
	}
	return servicereauth.ValidateTicketBinding(operation, input.binding())
}

func parseReauthUintQuery(raw string) uint {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, strconv.IntSize)
	if err != nil {
		return 0
	}
	return uint(value)
}

func parseReauthUint64Query(raw string) uint64 {
	value, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

type reauthPasswordInput struct {
	reauthScopeInput
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (h *ReauthHandler) VerifyPassword(c echo.Context) error {
	var input reauthPasswordInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if err := validateReauthScope(operation, input.reauthScopeInput); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyPasswordWithBinding(
		apimw.From(c).UserID, operation, input.Password, input.Code, input.binding(),
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthPasskeyStartInput struct{ reauthScopeInput }

func (h *ReauthHandler) BeginPasskey(c echo.Context) error {
	var input reauthPasskeyStartInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if err := validateReauthScope(operation, input.reauthScopeInput); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginPasskeyWithBinding(
		apimw.From(c).UserID, operation, input.binding(), c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

type reauthPasskeyFinishInput struct {
	reauthScopeInput
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (h *ReauthHandler) FinishPasskey(c echo.Context) error {
	var input reauthPasskeyFinishInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if err := validateReauthScope(operation, input.reauthScopeInput); err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if input.SessionID == "" || len(input.Credential) == 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "session_id_and_credential_are_required")
	}

	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(input.Credential))
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, "invalid_credential_payload")
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).FinishPasskeyWithBinding(
		apimw.From(c).UserID, operation, input.binding(), input.SessionID, parsedResponse,
		c.Request().Header.Get("Origin"), c.Request().Host, c.Scheme(),
	)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}

type reauthOIDCStartInput struct{ reauthScopeInput }

// BeginOIDC starts an OIDC step-up for the operation and returns the provider
// authorization URL. The client opens it in a popup; the callback lands on the
// dedicated, operation-agnostic reauth route (see OIDCCallback) which posts the
// result back to the opener.
func (h *ReauthHandler) BeginOIDC(c echo.Context) error {
	var input reauthOIDCStartInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if err := validateReauthScope(operation, input.reauthScopeInput); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	result, err := h.Service.WithContext(c.Request().Context()).BeginOIDCWithBinding(apimw.From(c).UserID, operation, input.binding())
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

type reauthOIDCFinishInput struct{ reauthScopeInput }

// FinishOIDC completes an OIDC step-up: it reads the reauth-scoped session cookie
// set by the callback, spends it for this user and operation, and mints a ticket.
// The cookie is always cleared, so a failed attempt cannot be replayed.
func (h *ReauthHandler) FinishOIDC(c echo.Context) error {
	var input reauthOIDCFinishInput
	if !httpx.BindJSON(c, &input, "invalid_request_body") {
		return nil
	}
	operation, err := validateReauthOperation(input.Operation)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	if err := validateReauthScope(operation, input.reauthScopeInput); err != nil {
		return apimw.WriteReauthError(c, err)
	}

	sessionID := apimw.GetCookieValue(c, apimw.OIDCReauthSessionCookieName)
	apimw.ClearOIDCReauthSessionCookie(c)
	if sessionID == "" {
		return apimw.WriteReauthError(c, servicereauth.ErrReauthRequired)
	}

	ticket, err := h.Service.WithContext(c.Request().Context()).VerifyOIDCWithBinding(apimw.From(c).UserID, operation, input.binding(), sessionID)
	if err != nil {
		return apimw.WriteReauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"ticket": ticket})
}
