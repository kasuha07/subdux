package api

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shiroha/subdux/internal/service"
)

const (
	mcpProtocolVersion = "2025-06-18"
)

type MCPHandler struct {
	apiKeys        *service.APIKeyService
	audit          *service.AuditService
	subscriptions  *service.SubscriptionService
	exchangeRates  *service.ExchangeRateService
	currencies     *service.CurrencyService
	categories     *service.CategoryService
	paymentMethods *service.PaymentMethodService
	server         *mcp.Server
	httpHandler    http.Handler
}

func NewMCPHandler(
	apiKeys *service.APIKeyService,
	audit *service.AuditService,
	subscriptions *service.SubscriptionService,
	exchangeRates *service.ExchangeRateService,
	currencies *service.CurrencyService,
	categories *service.CategoryService,
	paymentMethods *service.PaymentMethodService,
) *MCPHandler {
	handler := &MCPHandler{
		apiKeys:        apiKeys,
		audit:          audit,
		subscriptions:  subscriptions,
		exchangeRates:  exchangeRates,
		currencies:     currencies,
		categories:     categories,
		paymentMethods: paymentMethods,
	}
	handler.server = handler.buildServer()
	handler.httpHandler = mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return handler.server },
		&mcp.StreamableHTTPOptions{
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
	UserID  uint
	KeyID   uint
	KeyKind string
	Scopes  []string
	Request mcpRequestMetadata
}

type mcpPrincipalContextKey struct{}

type mcpRequestMetadata struct {
	ClientName    string
	ClientVersion string
	RequestID     string
}

func (h *MCPHandler) HandlePost(c echo.Context) error {
	c.Response().Header().Set("MCP-Protocol-Version", mcpProtocolVersion)

	principal, status, err := h.authenticate(c)
	if err != nil {
		return c.JSON(status, echo.Map{"error": err.Error()})
	}
	if err := validateMCPOrigin(c); err != nil {
		return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
	}
	if err := validateMCPProtocolHeader(c); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}
	if err := validateMCPContentTypeHeader(c); err != nil {
		return c.JSON(http.StatusUnsupportedMediaType, echo.Map{"error": err.Error()})
	}
	if err := validateMCPAcceptHeader(c); err != nil {
		return c.JSON(http.StatusNotAcceptable, echo.Map{"error": err.Error()})
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
	c.Response().Header().Set("MCP-Protocol-Version", mcpProtocolVersion)

	if _, status, err := h.authenticate(c); err != nil {
		return c.JSON(status, echo.Map{"error": err.Error()})
	}
	if err := validateMCPOrigin(c); err != nil {
		return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
	}
	if err := validateMCPAcceptHeader(c); err != nil {
		return c.JSON(http.StatusNotAcceptable, echo.Map{"error": err.Error()})
	}
	if err := validateMCPContentTypeHeader(c); err != nil {
		return c.JSON(http.StatusUnsupportedMediaType, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusMethodNotAllowed)
}

func (h *MCPHandler) authenticate(c echo.Context) (*mcpPrincipal, int, error) {
	key := strings.TrimSpace(c.Request().Header.Get("X-API-Key"))
	if key == "" {
		return nil, http.StatusUnauthorized, errors.New("api key is required")
	}

	principal, err := h.apiKeys.ValidateKey(key)
	if err != nil {
		return nil, http.StatusUnauthorized, err
	}
	if principal.KeyKind != service.APIKeyKindMCPClient {
		return nil, http.StatusForbidden, errors.New("api key kind cannot access mcp")
	}

	return &mcpPrincipal{
		UserID:  principal.UserID,
		KeyID:   principal.KeyID,
		KeyKind: principal.KeyKind,
		Scopes:  principal.Scopes,
	}, http.StatusOK, nil
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
	case "", "2024-11-05", "2025-03-26", mcpProtocolVersion, "2025-11-25":
		return nil
	default:
		return fmt.Errorf("unsupported MCP protocol version: %s", protocolVersion)
	}
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
