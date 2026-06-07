package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
	"github.com/shiroha/subdux/internal/version"
)

const (
	mcpProtocolVersion = "2025-06-18"
)

type MCPHandler struct {
	apiKeys        *service.APIKeyService
	subscriptions  *service.SubscriptionService
	exchangeRates  *service.ExchangeRateService
	currencies     *service.CurrencyService
	categories     *service.CategoryService
	paymentMethods *service.PaymentMethodService
	tools          []mcpTool
}

func NewMCPHandler(
	apiKeys *service.APIKeyService,
	subscriptions *service.SubscriptionService,
	exchangeRates *service.ExchangeRateService,
	currencies *service.CurrencyService,
	categories *service.CategoryService,
	paymentMethods *service.PaymentMethodService,
) *MCPHandler {
	handler := &MCPHandler{
		apiKeys:        apiKeys,
		subscriptions:  subscriptions,
		exchangeRates:  exchangeRates,
		currencies:     currencies,
		categories:     categories,
		paymentMethods: paymentMethods,
	}
	handler.tools = handler.buildTools()
	return handler
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type mcpPrincipal struct {
	UserID uint
	Scopes []string
}

func (h *MCPHandler) HandlePost(c echo.Context) error {
	c.Response().Header().Set("MCP-Protocol-Version", mcpProtocolVersion)

	principal, err := h.authenticate(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}
	if err := validateMCPOrigin(c); err != nil {
		return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
	}
	if err := validateMCPProtocolHeader(c); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	decoder := json.NewDecoder(c.Request().Body)
	decoder.DisallowUnknownFields()

	var request mcpRequest
	if err := decoder.Decode(&request); err != nil {
		return c.JSON(http.StatusBadRequest, mcpErrorResponse(nil, -32700, "parse error", nil))
	}
	if len(request.ID) == 0 {
		return c.NoContent(http.StatusAccepted)
	}
	if request.JSONRPC != "2.0" || request.Method == "" {
		return c.JSON(http.StatusOK, mcpErrorResponse(request.ID, -32600, "invalid request", nil))
	}

	switch request.Method {
	case "initialize":
		return c.JSON(http.StatusOK, mcpSuccessResponse(request.ID, h.initializeResult()))
	case "ping":
		return c.JSON(http.StatusOK, mcpSuccessResponse(request.ID, map[string]interface{}{}))
	case "tools/list":
		return c.JSON(http.StatusOK, mcpSuccessResponse(request.ID, map[string]interface{}{"tools": h.tools}))
	case "tools/call":
		result, rpcErr := h.handleToolCall(principal, request.Params)
		if rpcErr != nil {
			return c.JSON(http.StatusOK, mcpErrorResponse(request.ID, rpcErr.Code, rpcErr.Message, rpcErr.Data))
		}
		return c.JSON(http.StatusOK, mcpSuccessResponse(request.ID, result))
	default:
		return c.JSON(http.StatusOK, mcpErrorResponse(request.ID, -32601, "method not found", nil))
	}
}

func (h *MCPHandler) MethodNotAllowed(c echo.Context) error {
	c.Response().Header().Set("MCP-Protocol-Version", mcpProtocolVersion)

	if _, err := h.authenticate(c); err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}
	if err := validateMCPOrigin(c); err != nil {
		return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
	}
	return c.NoContent(http.StatusMethodNotAllowed)
}

func (h *MCPHandler) authenticate(c echo.Context) (*mcpPrincipal, error) {
	key := strings.TrimSpace(c.Request().Header.Get("X-API-Key"))
	if key == "" {
		return nil, errors.New("api key is required")
	}

	principal, err := h.apiKeys.ValidateKey(key)
	if err != nil {
		return nil, err
	}

	return &mcpPrincipal{
		UserID: principal.UserID,
		Scopes: principal.Scopes,
	}, nil
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
	case "", "2025-03-26", mcpProtocolVersion:
		return nil
	default:
		return fmt.Errorf("unsupported MCP protocol version: %s", protocolVersion)
	}
}

func (h *MCPHandler) initializeResult() map[string]interface{} {
	info := version.Get()
	return map[string]interface{}{
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "subdux",
			"title":   "Subdux",
			"version": info.Version,
		},
		"instructions": "Use X-API-Key authentication. Read tools require the read scope; write tools require the write scope.",
	}
}
