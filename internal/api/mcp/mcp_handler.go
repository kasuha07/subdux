package mcp

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	auditservice "github.com/kasuha07/subdux/internal/service/audit"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	idempotencyservice "github.com/kasuha07/subdux/internal/service/idempotency"
	"github.com/kasuha07/subdux/internal/service/mcpoauth"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"github.com/labstack/echo/v4"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpProtocolVersion = "2025-06-18"
)

type MCPHandler struct {
	apiKeys        *apikeyservice.Service
	oauth          *mcpoauth.Service
	audit          *auditservice.Service
	idempotency    *idempotencyservice.Service
	subscriptions  *subscriptionservice.Service
	exchangeRates  *exchangerate.Service
	currencies     *catalogservice.CurrencyService
	categories     *catalogservice.CategoryService
	paymentMethods *catalogservice.PaymentMethodService
	server         *mcpsdk.Server
	httpHandler    http.Handler
}

func NewMCPHandler(
	apiKeys *apikeyservice.Service,
	audit *auditservice.Service,
	subscriptions *subscriptionservice.Service,
	exchangeRates *exchangerate.Service,
	currencies *catalogservice.CurrencyService,
	categories *catalogservice.CategoryService,
	paymentMethods *catalogservice.PaymentMethodService,
) *MCPHandler {
	handler := &MCPHandler{
		apiKeys:        apiKeys,
		oauth:          mcpoauth.NewService(subscriptions.DB),
		audit:          audit,
		idempotency:    idempotencyservice.NewService(subscriptions.DB),
		subscriptions:  subscriptions,
		exchangeRates:  exchangeRates,
		currencies:     currencies,
		categories:     categories,
		paymentMethods: paymentMethods,
	}
	handler.server = handler.buildServer()
	handler.httpHandler = mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return handler.server },
		&mcpsdk.StreamableHTTPOptions{
			Stateless:                  true,
			JSONResponse:               true,
			DisableLocalhostProtection: true,
		},
	)
	return handler
}

type mcpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type mcpPrincipal struct {
	UserID          uint
	KeyID           uint
	KeyKind         string
	Scopes          []string
	Request         mcpRequestMetadata
	OAuthGrantID    *uint
	OAuthClientID   string
	OAuthClientName string
}

type mcpPrincipalContextKey struct{}

type mcpRequestMetadata struct {
	ClientName    string
	ClientVersion string
	RequestID     string
}

func writeError(c echo.Context, status int, message string) error {
	return c.JSON(status, echo.Map{"error": message})
}

func (h *MCPHandler) HandlePost(c echo.Context) error {
	c.Response().Header().Set("MCP-Protocol-Version", mcpResponseProtocolVersion(c))

	principal, status, err := h.authenticate(c)
	if err != nil {
		return writeError(c, status, err.Error())
	}
	if err := validateMCPOrigin(c); err != nil {
		return writeError(c, http.StatusForbidden, err.Error())
	}
	if err := validateMCPProtocolHeader(c); err != nil {
		return writeError(c, http.StatusBadRequest, err.Error())
	}
	if err := validateMCPContentTypeHeader(c); err != nil {
		return writeError(c, http.StatusUnsupportedMediaType, err.Error())
	}
	if err := validateMCPAcceptHeader(c); err != nil {
		return writeError(c, http.StatusNotAcceptable, err.Error())
	}

	principal.Request = readMCPRequestMetadata(c)
	req := c.Request().Clone(context.WithValue(c.Request().Context(), mcpPrincipalContextKey{}, principal))
	req.Header.Set(echo.HeaderAccept, echo.MIMEApplicationJSON+", text/event-stream")
	h.httpHandler.ServeHTTP(c.Response(), req)
	return nil
}

func readMCPRequestMetadata(c echo.Context) mcpRequestMetadata {
	return mcpRequestMetadata{
		ClientName:    strings.TrimSpace(c.Request().Header.Get("MCP-Client-Name")),
		ClientVersion: strings.TrimSpace(c.Request().Header.Get("MCP-Client-Version")),
		RequestID:     strings.TrimSpace(c.Request().Header.Get("X-Request-ID")),
	}
}

func (h *MCPHandler) MethodNotAllowed(c echo.Context) error {
	c.Response().Header().Set("MCP-Protocol-Version", mcpResponseProtocolVersion(c))

	if _, status, err := h.authenticate(c); err != nil {
		return writeError(c, status, err.Error())
	}
	if err := validateMCPOrigin(c); err != nil {
		return writeError(c, http.StatusForbidden, err.Error())
	}
	if err := validateMCPAcceptHeader(c); err != nil {
		return writeError(c, http.StatusNotAcceptable, err.Error())
	}
	if err := validateMCPContentTypeHeader(c); err != nil {
		return writeError(c, http.StatusUnsupportedMediaType, err.Error())
	}
	return c.NoContent(http.StatusMethodNotAllowed)
}

func (h *MCPHandler) authenticate(c echo.Context) (*mcpPrincipal, int, error) {
	if authorization := c.Request().Header.Get(echo.HeaderAuthorization); authorization != "" {
		parts := strings.Fields(authorization)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			h.oauthChallenge(c, "invalid_token")
			return nil, http.StatusUnauthorized, errors.New("invalid MCP bearer authorization")
		}
		p, err := h.oauth.WithContext(c.Request().Context()).ValidateToken(parts[1])
		if err != nil {
			h.oauthChallenge(c, "invalid_token")
			return nil, http.StatusUnauthorized, errors.New("MCP access token expired or invalid")
		}
		return &mcpPrincipal{UserID: p.UserID, KeyKind: "mcp_oauth", Scopes: p.Scopes,
			OAuthGrantID: &p.GrantID, OAuthClientID: p.ClientID, OAuthClientName: p.ClientName}, http.StatusOK, nil
	}
	key := strings.TrimSpace(c.Request().Header.Get("X-API-Key"))
	if key == "" {
		h.oauthChallenge(c, "")
		return nil, http.StatusUnauthorized, errors.New("api key is required")
	}

	principal, err := h.apiKeys.WithContext(c.Request().Context()).ValidateKey(key)
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}
	if principal.KeyKind != apikeyservice.APIKeyKindMCPClient {
		return nil, http.StatusForbidden, errors.New("api key kind cannot access mcp")
	}

	return &mcpPrincipal{
		UserID:  principal.UserID,
		KeyID:   principal.KeyID,
		KeyKind: principal.KeyKind,
		Scopes:  principal.Scopes,
	}, http.StatusOK, nil
}

func (h *MCPHandler) oauthChallenge(c echo.Context, code string) {
	value := `Bearer scope="read"`
	if issuer, err := h.oauth.WithContext(c.Request().Context()).Issuer(); err == nil {
		value += fmt.Sprintf(", resource_metadata=%q", issuer+"/.well-known/oauth-protected-resource/mcp")
	}
	if code != "" {
		value += fmt.Sprintf(", error=%q, error_description=%q", code, "MCP authorization is required")
	}
	c.Response().Header().Set("WWW-Authenticate", value)
}

func validateMCPOrigin(c echo.Context) error {
	origin := strings.TrimSpace(c.Request().Header.Get("Origin"))
	if origin == "" {
		return nil
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("invalid origin")
	}
	if strings.EqualFold(parsed.Host, c.Request().Host) {
		return nil
	}

	return errors.New("origin is not allowed")
}

func validateMCPProtocolHeader(c echo.Context) error {
	protocolVersion := strings.TrimSpace(c.Request().Header.Get("MCP-Protocol-Version"))
	switch protocolVersion {
	case "", "2024-11-05", "2025-03-26", mcpProtocolVersion, "2025-11-25", "2026-07-28":
		return nil
	default:
		return fmt.Errorf("unsupported MCP protocol version: %s", protocolVersion)
	}
}

func mcpResponseProtocolVersion(c echo.Context) string {
	if requested := strings.TrimSpace(c.Request().Header.Get("MCP-Protocol-Version")); requested == "2026-07-28" || requested == "2025-11-25" {
		return requested
	}
	return mcpProtocolVersion
}

func validateMCPContentTypeHeader(c echo.Context) error {
	contentType := strings.TrimSpace(c.Request().Header.Get(echo.HeaderContentType))
	if contentType == "" {
		return errors.New("content-type application/json is required")
	}
	return validateMCPContentTypeValue(contentType)
}

func validateMCPContentTypeValue(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, echo.MIMEApplicationJSON) {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
	return nil
}

func validateMCPAcceptHeader(c echo.Context) error {
	accept := strings.TrimSpace(c.Request().Header.Get(echo.HeaderAccept))
	if accept == "" {
		return errors.New("accept application/json is required")
	}
	if !mcpAcceptsJSON(accept) {
		return fmt.Errorf("unsupported accept header: %s", accept)
	}
	return nil
}

func mcpAcceptsJSON(accept string) bool {
	for _, part := range strings.Split(accept, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		mediaType, params, err := mime.ParseMediaType(part)
		if err != nil {
			continue
		}
		if q, ok := params["q"]; ok {
			value, err := strconv.ParseFloat(q, 64)
			if err == nil && value <= 0 {
				continue
			}
		}

		mediaType = strings.ToLower(strings.TrimSpace(mediaType))
		switch {
		case mediaType == echo.MIMEApplicationJSON || mediaType == "*/*":
			return true
		case strings.HasSuffix(mediaType, "/*"):
			prefix := strings.TrimSuffix(mediaType, "/*")
			if prefix == "application" {
				return true
			}
		}
	}
	return false
}
