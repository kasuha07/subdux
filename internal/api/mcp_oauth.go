package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/service/mcpoauth"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	"github.com/labstack/echo/v4"
)

type MCPOAuthHandler struct {
	Service  *mcpoauth.Service
	Settings *systemsettings.Service
}

func oauthHeaders(c echo.Context) {
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Pragma", "no-cache")
	c.Response().Header().Set("Referrer-Policy", "no-referrer")
}

func oauthError(c echo.Context, err error) error {
	oauthHeaders(c)
	var protocol *mcpoauth.ProtocolError
	if errors.As(err, &protocol) {
		status := http.StatusBadRequest
		if protocol.Code == "temporarily_unavailable" {
			status = http.StatusServiceUnavailable
		}
		return c.JSON(status, echo.Map{"error": protocol.Code, "error_description": protocol.Description})
	}
	return c.JSON(http.StatusInternalServerError, echo.Map{"error": "server_error", "error_description": "unable to process OAuth request"})
}

func uniqueOAuthParams(params url.Values) error {
	for _, values := range params {
		if len(values) != 1 {
			return &mcpoauth.ProtocolError{Code: "invalid_request", Description: "duplicate OAuth parameters are not accepted"}
		}
	}
	return nil
}

func oauthForm(c echo.Context) (url.Values, error) {
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != "application/x-www-form-urlencoded" {
		return nil, &mcpoauth.ProtocolError{Code: "invalid_request", Description: "content type must be application/x-www-form-urlencoded"}
	}
	if c.Request().Header.Get(echo.HeaderAuthorization) != "" {
		return nil, &mcpoauth.ProtocolError{Code: "invalid_client", Description: "use a public client ID and PKCE without client authentication"}
	}
	if err := c.Request().ParseForm(); err != nil {
		return nil, &mcpoauth.ProtocolError{Code: "invalid_request", Description: "invalid form body"}
	}
	params := c.Request().PostForm
	return params, uniqueOAuthParams(params)
}

func (h *MCPOAuthHandler) Mount(e *echo.Echo, settings *systemsettings.Service) {
	gate := apimw.MCPEnabledMiddleware(settings)
	body := apimw.RequestBodyLimitMiddleware(64<<10, nil)
	e.GET("/.well-known/oauth-protected-resource", h.ResourceMetadata, gate)
	e.GET("/.well-known/oauth-protected-resource/mcp", h.ResourceMetadata, gate)
	e.GET("/.well-known/oauth-authorization-server", h.ServerMetadata, gate)
	e.GET("/oauth/authorize", h.Authorize, gate, apimw.AuthIPRateLimit(30, time.Minute))
	e.POST("/oauth/register", h.Register, gate, body, apimw.AuthIPRateLimit(10, time.Minute))
	e.POST("/oauth/token", h.Token, gate, body, apimw.AuthIPRateLimit(120, time.Minute))
	e.POST("/oauth/revoke", h.RevokeToken, gate, body, apimw.AuthIPRateLimit(60, time.Minute))
}

func (h *MCPOAuthHandler) RegisterRoutes(g RouteGroups) {
	gate := apimw.MCPEnabledMiddleware(h.Settings)
	g.HumanProtected.GET("/mcp/oauth/info", h.Info)
	g.HumanProtected.GET("/mcp/oauth/requests/:request", h.Consent, gate)
	g.HumanProtected.POST("/mcp/oauth/requests/:request", h.Decide, gate)
	g.HumanProtected.GET("/mcp/oauth/grants", h.Grants)
	g.HumanProtected.DELETE("/mcp/oauth/grants/:id", h.RevokeGrant)
}

func (h *MCPOAuthHandler) Info(c echo.Context) error {
	oauthHeaders(c)
	enabled, err := h.Settings.IsMCPEnabled()
	if err != nil {
		return oauthHumanError(c, err)
	}
	issuer, err := h.Service.WithContext(c.Request().Context()).Issuer()
	endpoint := ""
	if err == nil {
		endpoint = issuer + "/mcp"
	}
	return c.JSON(http.StatusOK, echo.Map{"enabled": enabled, "configured": err == nil, "url": endpoint})
}

func (h *MCPOAuthHandler) ResourceMetadata(c echo.Context) error {
	oauthHeaders(c)
	issuer, err := h.Service.WithContext(c.Request().Context()).Issuer()
	if err != nil {
		return oauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{
		"resource": issuer + "/mcp", "authorization_servers": []string{issuer},
		"scopes_supported": []string{"read"}, "bearer_methods_supported": []string{"header"},
		"resource_name": "Subdux MCP",
	})
}

func (h *MCPOAuthHandler) ServerMetadata(c echo.Context) error {
	oauthHeaders(c)
	issuer, err := h.Service.WithContext(c.Request().Context()).Issuer()
	if err != nil {
		return oauthError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{
		"issuer": issuer, "authorization_endpoint": issuer + "/oauth/authorize", "token_endpoint": issuer + "/oauth/token",
		"registration_endpoint": issuer + "/oauth/register", "revocation_endpoint": issuer + "/oauth/revoke",
		"response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"none"},
		"revocation_endpoint_auth_methods_supported": []string{"none"}, "response_modes_supported": []string{"query"},
		"scopes_supported":                      []string{"read", "write", "offline_access"},
		"client_id_metadata_document_supported": true, "authorization_response_iss_parameter_supported": true,
	})
}

func (h *MCPOAuthHandler) Register(c echo.Context) error {
	oauthHeaders(c)
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return oauthError(c, &mcpoauth.ProtocolError{Code: "invalid_client_metadata", Description: "content type must be application/json"})
	}
	var metadata mcpoauth.ClientMetadata
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&metadata); err != nil {
		return oauthError(c, &mcpoauth.ProtocolError{Code: "invalid_client_metadata", Description: "invalid client metadata JSON"})
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return oauthError(c, &mcpoauth.ProtocolError{Code: "invalid_client_metadata", Description: "invalid client metadata JSON"})
	}
	result, err := h.Service.WithContext(c.Request().Context()).Register(metadata)
	if err != nil {
		return oauthError(c, err)
	}
	return c.JSON(http.StatusCreated, result)
}

func (h *MCPOAuthHandler) Authorize(c echo.Context) error {
	oauthHeaders(c)
	params := c.QueryParams()
	if err := uniqueOAuthParams(params); err != nil {
		return oauthError(c, err)
	}
	handle, err := h.Service.WithContext(c.Request().Context()).Begin(params)
	if err != nil {
		return oauthError(c, err)
	}
	return c.Redirect(http.StatusFound, "/connect/mcp?request="+url.QueryEscape(handle))
}

func (h *MCPOAuthHandler) Token(c echo.Context) error {
	oauthHeaders(c)
	params, err := oauthForm(c)
	if err != nil {
		return oauthError(c, err)
	}
	result, err := h.Service.WithContext(c.Request().Context()).Exchange(params)
	if err != nil {
		return oauthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *MCPOAuthHandler) RevokeToken(c echo.Context) error {
	oauthHeaders(c)
	params, err := oauthForm(c)
	if err != nil {
		return oauthError(c, err)
	}
	if err := h.Service.WithContext(c.Request().Context()).RevokeToken(params.Get("token"), params.Get("client_id")); err != nil {
		return oauthError(c, err)
	}
	return c.NoContent(http.StatusOK)
}

func oauthHumanError(c echo.Context, err error) error {
	var protocol *mcpoauth.ProtocolError
	if !errors.As(err, &protocol) {
		return httpx.WriteError(c, http.StatusInternalServerError, "mcp_oauth_failed")
	}
	return httpx.WriteError(c, http.StatusBadRequest, "mcp_oauth_request_invalid")
}

func (h *MCPOAuthHandler) Consent(c echo.Context) error {
	oauthHeaders(c)
	result, err := h.Service.WithContext(c.Request().Context()).Consent(c.Param("request"), apimw.From(c).UserID)
	if err != nil {
		return oauthHumanError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *MCPOAuthHandler) Decide(c echo.Context) error {
	oauthHeaders(c)
	mediaType, _, err := mime.ParseMediaType(c.Request().Header.Get(echo.HeaderContentType))
	if err != nil || mediaType != echo.MIMEApplicationJSON {
		return httpx.WriteError(c, http.StatusUnsupportedMediaType, "mcp_oauth_request_invalid")
	}
	var input struct {
		Approve    *bool `json:"approve"`
		AllowWrite bool  `json:"allow_write"`
	}
	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.Approve == nil {
		return httpx.WriteError(c, http.StatusBadRequest, "mcp_oauth_request_invalid")
	}
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		return httpx.WriteError(c, http.StatusBadRequest, "mcp_oauth_request_invalid")
	}
	redirect, err := h.Service.WithContext(c.Request().Context()).Decide(c.Param("request"), apimw.From(c).UserID, *input.Approve, input.AllowWrite)
	if err != nil {
		return oauthHumanError(c, err)
	}
	return c.JSON(http.StatusOK, echo.Map{"redirect_uri": redirect})
}

func (h *MCPOAuthHandler) Grants(c echo.Context) error {
	oauthHeaders(c)
	result, err := h.Service.WithContext(c.Request().Context()).ListGrants(apimw.From(c).UserID)
	if err != nil {
		return oauthHumanError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *MCPOAuthHandler) RevokeGrant(c echo.Context) error {
	oauthHeaders(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || id == 0 {
		return httpx.WriteError(c, http.StatusBadRequest, "mcp_oauth_request_invalid")
	}
	if err := h.Service.WithContext(c.Request().Context()).RevokeGrant(apimw.From(c).UserID, uint(id)); err != nil {
		return oauthHumanError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}
