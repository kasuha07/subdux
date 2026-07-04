package api

import (
	"errors"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

type oidcSessionResponse struct {
	Purpose     string                          `json:"purpose"`
	Token       string                          `json:"token,omitempty"`
	AccessToken string                          `json:"access_token,omitempty"`
	User        *authUserResponse               `json:"user,omitempty"`
	Connected   bool                            `json:"connected,omitempty"`
	Connection  *serviceauth.OIDCConnectionInfo `json:"connection,omitempty"`
	Error       string                          `json:"error,omitempty"`
}

func mapOIDCSessionResponse(result *serviceauth.OIDCSessionResult) oidcSessionResponse {
	var user *authUserResponse
	if result.User != nil {
		mapped := mapAuthUserResponse(*result.User)
		user = &mapped
	}

	return oidcSessionResponse{
		Purpose:     result.Purpose,
		Token:       result.Token,
		AccessToken: result.Token,
		User:        user,
		Connected:   result.Connected,
		Connection:  result.Connection,
		Error:       result.Error,
	}
}

func writeOIDCSessionSuccess(c echo.Context, status int, result *serviceauth.OIDCSessionResult) error {
	apimw.SetRefreshTokenCookie(c, result.RefreshToken)
	return c.JSON(status, mapOIDCSessionResponse(result))
}

func (h *AuthHandler) GetOIDCConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Service.WithContext(c.Request().Context()).GetOIDCPublicConfig())
}

func (h *AuthHandler) BeginOIDCLogin(c echo.Context) error {
	result, err := h.Service.WithContext(c.Request().Context()).BeginOIDCLogin()
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) BeginOIDCConnect(c echo.Context) error {
	userID := apimw.From(c).UserID

	authService := h.Service.WithContext(c.Request().Context())
	if err := h.Reauth.WithContext(c.Request().Context()).ConsumeOIDCConnect(
		userID,
		apimw.ReauthTicketFromRequest(c),
	); err != nil {
		if !errors.Is(err, servicereauth.ErrReauthRequired) {
			return httpx.WriteError(c, http.StatusInternalServerError, "failed to load oidc connections")
		}
		return apimw.WriteReauthError(c, err)
	}

	result, err := authService.BeginOIDCConnect(userID)
	if err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) OIDCCallback(c echo.Context) error {
	callbackResult, err := h.Service.WithContext(c.Request().Context()).HandleOIDCCallback(
		c.QueryParam("state"),
		c.QueryParam("code"),
		c.QueryParam("error"),
		c.QueryParam("error_description"),
	)
	if err != nil {
		apimw.ClearOIDCSessionCookie(c)
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to process oidc callback")
	}
	if callbackResult.SessionID == "" {
		apimw.ClearOIDCSessionCookie(c)
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to finalize oidc callback")
	}

	// Reauth ("step-up") uses its own path-scoped cookie and lands on a dedicated,
	// operation-agnostic popup route that posts the outcome back to the opener (the
	// page that started the step-up, still open in the background). Login and
	// connect keep the ordinary session cookie and full-page redirect.
	if callbackResult.Purpose == "reauth" {
		apimw.SetOIDCReauthSessionCookie(c, callbackResult.SessionID)
		return c.Redirect(http.StatusFound, "/oidc/reauth?oidc_action=reauth")
	}

	apimw.SetOIDCSessionCookie(c, callbackResult.SessionID)

	redirectPath := "/login"
	if callbackResult.Purpose == "connect" {
		redirectPath = "/settings"
	}

	return c.Redirect(http.StatusFound, redirectPath+"?oidc_action="+callbackResult.Purpose)
}

func (h *AuthHandler) GetOIDCSession(c echo.Context) error {
	sessionID := apimw.GetCookieValue(c, apimw.OIDCSessionCookieName)
	if sessionID == "" {
		apimw.ClearOIDCSessionCookie(c)
		return httpx.WriteError(c, http.StatusBadRequest, "oidc session cookie is required")
	}

	result, err := h.Service.WithContext(c.Request().Context()).ConsumeOIDCSessionResult(sessionID)
	apimw.ClearOIDCSessionCookie(c)
	if err != nil {
		return httpx.WriteError(c, http.StatusNotFound, err.Error())
	}

	return writeOIDCSessionSuccess(c, http.StatusOK, result)
}

func (h *AuthHandler) ListOIDCConnections(c echo.Context) error {
	userID := apimw.From(c).UserID
	connections, err := h.Service.WithContext(c.Request().Context()).ListOIDCConnections(userID)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed to list oidc connections")
	}

	return c.JSON(http.StatusOK, connections)
}

func (h *AuthHandler) DeleteOIDCConnection(c echo.Context) error {
	userID := apimw.From(c).UserID
	connectionID, ok := httpx.ParseUintParam(c, "id", "invalid oidc connection id")
	if !ok {
		return nil
	}

	if err := h.Service.WithContext(c.Request().Context()).DeleteOIDCConnection(userID, uint(connectionID)); err != nil {
		return httpx.WriteError(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "oidc connection deleted"})
}
