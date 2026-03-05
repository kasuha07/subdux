package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type oidcSessionResponse struct {
	Purpose      string                      `json:"purpose"`
	Token        string                      `json:"token,omitempty"`
	AccessToken  string                      `json:"access_token,omitempty"`
	RefreshToken string                      `json:"refresh_token,omitempty"`
	User         *authUserResponse           `json:"user,omitempty"`
	Connected    bool                        `json:"connected,omitempty"`
	Connection   *service.OIDCConnectionInfo `json:"connection,omitempty"`
	Error        string                      `json:"error,omitempty"`
}

func mapOIDCSessionResponse(result *service.OIDCSessionResult) oidcSessionResponse {
	var user *authUserResponse
	if result.User != nil {
		mapped := mapAuthUserResponse(*result.User)
		user = &mapped
	}

	return oidcSessionResponse{
		Purpose:      result.Purpose,
		Token:        result.Token,
		AccessToken:  result.Token,
		RefreshToken: result.RefreshToken,
		User:         user,
		Connected:    result.Connected,
		Connection:   result.Connection,
		Error:        result.Error,
	}
}

func writeOIDCSessionSuccess(c echo.Context, status int, result *service.OIDCSessionResult) error {
	setRefreshTokenCookie(c, result.RefreshToken)
	return c.JSON(status, mapOIDCSessionResponse(result))
}

func (h *AuthHandler) GetOIDCConfig(c echo.Context) error {
	return c.JSON(http.StatusOK, h.Service.GetOIDCPublicConfig())
}

func (h *AuthHandler) BeginOIDCLogin(c echo.Context) error {
	result, err := h.Service.BeginOIDCLogin()
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) BeginOIDCConnect(c echo.Context) error {
	userID := getUserID(c)
	result, err := h.Service.BeginOIDCConnect(userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) OIDCCallback(c echo.Context) error {
	callbackResult, err := h.Service.HandleOIDCCallback(
		c.QueryParam("state"),
		c.QueryParam("code"),
		c.QueryParam("error"),
		c.QueryParam("error_description"),
	)
	if err != nil {
		clearOIDCSessionCookie(c)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to process oidc callback"})
	}
	if callbackResult.SessionID == "" {
		clearOIDCSessionCookie(c)
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to finalize oidc callback"})
	}

	setOIDCSessionCookie(c, callbackResult.SessionID)

	redirectPath := "/login"
	if callbackResult.Purpose == "connect" {
		redirectPath = "/settings"
	}

	return c.Redirect(http.StatusFound, redirectPath+"?oidc_action="+callbackResult.Purpose)
}

func (h *AuthHandler) GetOIDCSession(c echo.Context) error {
	sessionID := getCookieValue(c, oidcSessionCookieName)
	if sessionID == "" {
		clearOIDCSessionCookie(c)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "oidc session cookie is required"})
	}

	result, err := h.Service.ConsumeOIDCSessionResult(sessionID)
	clearOIDCSessionCookie(c)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return writeOIDCSessionSuccess(c, http.StatusOK, result)
}

func (h *AuthHandler) ListOIDCConnections(c echo.Context) error {
	userID := getUserID(c)
	connections, err := h.Service.ListOIDCConnections(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to list oidc connections"})
	}

	return c.JSON(http.StatusOK, connections)
}

func (h *AuthHandler) DeleteOIDCConnection(c echo.Context) error {
	userID := getUserID(c)
	connectionID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid oidc connection id"})
	}

	if err := h.Service.DeleteOIDCConnection(userID, uint(connectionID)); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "oidc connection deleted"})
}
