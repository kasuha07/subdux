package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
	"github.com/shiroha/subdux/internal/version"
	"gorm.io/gorm"
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

type mcpTool struct {
	Name        string                 `json:"name"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

type mcpToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type mcpToolResult struct {
	Content           []mcpTextContent `json:"content"`
	StructuredContent interface{}      `json:"structuredContent,omitempty"`
	IsError           bool             `json:"isError,omitempty"`
}

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
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

func (h *MCPHandler) handleToolCall(principal *mcpPrincipal, rawParams json.RawMessage) (*mcpToolResult, *mcpError) {
	var params mcpToolCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, &mcpError{Code: -32602, Message: "invalid tool call params"}
	}
	params.Name = strings.TrimSpace(params.Name)
	if params.Name == "" {
		return nil, &mcpError{Code: -32602, Message: "tool name is required"}
	}
	if params.Arguments == nil {
		params.Arguments = map[string]interface{}{}
	}

	requiredScope := service.APIKeyScopeRead
	if isMCPWriteTool(params.Name) {
		requiredScope = service.APIKeyScopeWrite
	}
	if !mcpPrincipalHasScope(principal, requiredScope) {
		return mcpToolExecutionError("api key does not have required scope"), nil
	}

	switch params.Name {
	case "list_subscriptions":
		return h.callListSubscriptions(principal.UserID)
	case "get_subscription":
		return h.callGetSubscription(principal.UserID, params.Arguments)
	case "create_subscription":
		return h.callCreateSubscription(principal.UserID, params.Arguments)
	case "update_subscription":
		return h.callUpdateSubscription(principal.UserID, params.Arguments)
	case "delete_subscription":
		return h.callDeleteSubscription(principal.UserID, params.Arguments)
	case "mark_subscription_renewed":
		return h.callMarkSubscriptionRenewed(principal.UserID, params.Arguments)
	case "get_dashboard_summary":
		return h.callDashboardSummary(principal.UserID, params.Arguments)
	case "list_categories":
		return h.callListCategories(principal.UserID)
	case "list_payment_methods":
		return h.callListPaymentMethods(principal.UserID)
	default:
		return nil, &mcpError{Code: -32602, Message: "unknown tool: " + params.Name}
	}
}

func mcpPrincipalHasScope(principal *mcpPrincipal, scope string) bool {
	for _, candidate := range principal.Scopes {
		if candidate == scope {
			return true
		}
	}
	return false
}

func isMCPWriteTool(name string) bool {
	switch name {
	case "create_subscription", "update_subscription", "delete_subscription", "mark_subscription_renewed":
		return true
	default:
		return false
	}
}

func (h *MCPHandler) callListSubscriptions(userID uint) (*mcpToolResult, *mcpError) {
	subs, err := h.subscriptions.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"subscriptions": mapSubscriptionResponses(subs),
	}), nil
}

func (h *MCPHandler) callGetSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	sub, err := h.subscriptions.GetByID(userID, id)
	if err != nil {
		return mcpToolExecutionError("subscription not found"), nil
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callCreateSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	if err := validateSubscriptionWriteArgTypes(args); err != nil {
		return nil, invalidMCPParams(err)
	}
	input := createSubscriptionInputFromMCPArgs(args)
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, invalidMCPParams(errors.New("name is required"))
	}
	if input.Amount < 0 {
		return nil, invalidMCPParams(errors.New("amount must not be negative"))
	}
	if !validateSubscriptionIcon(input.Icon) {
		return nil, invalidMCPParams(errors.New("invalid icon value"))
	}

	sub, err := h.subscriptions.Create(userID, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callUpdateSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	input, err := updateSubscriptionInputFromMCPArgs(args)
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	if input.Amount != nil && *input.Amount < 0 {
		return nil, invalidMCPParams(errors.New("amount must not be negative"))
	}
	if input.Icon != nil && !validateSubscriptionIcon(*input.Icon) {
		return nil, invalidMCPParams(errors.New("invalid icon value"))
	}

	sub, err := h.subscriptions.Update(userID, id, input)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callDeleteSubscription(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	if err := h.subscriptions.Delete(userID, id); err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{"deleted": true, "id": id}), nil
}

func (h *MCPHandler) callMarkSubscriptionRenewed(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	id, err := readRequiredIDArg(args, "id")
	if err != nil {
		return nil, invalidMCPParams(err)
	}
	sub, err := h.subscriptions.MarkManualRenewed(userID, id)
	if err != nil {
		if isSubscriptionBadRequestError(err.Error()) || errors.Is(err, gorm.ErrRecordNotFound) {
			return mcpToolExecutionError(err.Error()), nil
		}
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(mapSubscriptionResponse(*sub)), nil
}

func (h *MCPHandler) callDashboardSummary(userID uint, args map[string]interface{}) (*mcpToolResult, *mcpError) {
	if err := validateMCPArgTypes(args, []mcpArgSpec{{Key: "currency", Type: "string"}}); err != nil {
		return nil, invalidMCPParams(err)
	}
	currency, _ := readStringArg(args, "currency")
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency != "" {
		if err := h.validateUserCurrency(userID, currency); err != nil {
			return nil, invalidMCPParams(err)
		}
	} else {
		pref, _ := h.exchangeRates.GetUserPreference(userID)
		currency = pref.PreferredCurrency
	}

	summary, err := h.subscriptions.GetDashboardSummary(userID, currency, h.exchangeRates)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(summary), nil
}

func (h *MCPHandler) validateUserCurrency(userID uint, code string) error {
	currencies, err := h.currencies.List(userID)
	if err != nil {
		return err
	}
	for _, currency := range currencies {
		if strings.EqualFold(currency.Code, code) {
			return nil
		}
	}
	return errors.New("currency not found")
}

func (h *MCPHandler) callListCategories(userID uint) (*mcpToolResult, *mcpError) {
	categories, err := h.categories.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"categories": mapCategoryResponses(categories),
	}), nil
}

func (h *MCPHandler) callListPaymentMethods(userID uint) (*mcpToolResult, *mcpError) {
	methods, err := h.paymentMethods.List(userID)
	if err != nil {
		return nil, internalMCPError(err)
	}
	return mcpStructuredResult(map[string]interface{}{
		"payment_methods": mapPaymentMethodResponses(methods),
	}), nil
}

func createSubscriptionInputFromMCPArgs(args map[string]interface{}) service.CreateSubscriptionInput {
	intervalCount := 1
	if value, ok := readIntArg(args, "interval_count"); ok {
		intervalCount = value
	}

	input := service.CreateSubscriptionInput{
		Name:            readStringArgOrDefault(args, "name", ""),
		Amount:          readFloatArgOrDefault(args, "amount", 0),
		Currency:        readStringArgOrDefault(args, "currency", "USD"),
		Status:          readStringArgOrDefault(args, "status", "active"),
		RenewalMode:     readStringArgOrDefault(args, "renewal_mode", "auto_renew"),
		EndsAt:          readStringArgOrDefault(args, "ends_at", ""),
		BillingType:     readStringArgOrDefault(args, "billing_type", "recurring"),
		RecurrenceType:  readStringArgOrDefault(args, "recurrence_type", "interval"),
		IntervalCount:   &intervalCount,
		IntervalUnit:    readStringArgOrDefault(args, "interval_unit", "month"),
		NextBillingDate: readStringArgOrDefault(args, "next_billing_date", ""),
		Category:        readStringArgOrDefault(args, "category", ""),
		Icon:            readStringArgOrDefault(args, "icon", ""),
		URL:             readStringArgOrDefault(args, "url", ""),
		Notes:           readStringArgOrDefault(args, "notes", ""),
	}

	if value, ok := readUintPointerArg(args, "category_id"); ok {
		input.CategoryID = value
	}
	if value, ok := readUintPointerArg(args, "payment_method_id"); ok {
		input.PaymentMethodID = value
	}
	if value, ok := readBoolPointerArg(args, "notify_enabled"); ok {
		input.NotifyEnabled = value
	}
	if value, ok := readIntPointerArg(args, "notify_days_before"); ok {
		input.NotifyDaysBefore = value
	}

	switch input.RecurrenceType {
	case "monthly_date":
		input.IntervalCount = nil
		input.IntervalUnit = ""
		if value, ok := readIntPointerArg(args, "monthly_day"); ok {
			input.MonthlyDay = value
		}
	case "yearly_date":
		input.IntervalCount = nil
		input.IntervalUnit = ""
		if value, ok := readIntPointerArg(args, "yearly_month"); ok {
			input.YearlyMonth = value
		}
		if value, ok := readIntPointerArg(args, "yearly_day"); ok {
			input.YearlyDay = value
		}
	default:
		input.MonthlyDay = nil
		input.YearlyMonth = nil
		input.YearlyDay = nil
	}

	return input
}

func updateSubscriptionInputFromMCPArgs(args map[string]interface{}) (service.UpdateSubscriptionInput, error) {
	var input service.UpdateSubscriptionInput
	if err := validateSubscriptionWriteArgTypes(args); err != nil {
		return input, err
	}

	if value, ok := readStringArg(args, "name"); ok {
		trimmed := strings.TrimSpace(value)
		input.Name = &trimmed
	}
	if value, ok := readFloatArg(args, "amount"); ok {
		input.Amount = &value
	}
	if value, ok := readStringArg(args, "currency"); ok {
		input.Currency = &value
	}
	if value, ok := readStringArg(args, "status"); ok {
		input.Status = &value
	}
	if value, ok := readStringArg(args, "renewal_mode"); ok {
		input.RenewalMode = &value
	}
	if value, ok := readNullableStringArg(args, "ends_at"); ok {
		input.EndsAt = &value
	}
	if value, ok := readStringArg(args, "billing_type"); ok {
		input.BillingType = &value
	}
	if value, ok := readStringArg(args, "recurrence_type"); ok {
		input.RecurrenceType = &value
	}
	if value, ok := readNullableIntArg(args, "interval_count"); ok {
		input.IntervalCount = value
	}
	if value, ok := readStringArg(args, "interval_unit"); ok {
		input.IntervalUnit = &value
	}
	if value, ok := readNullableStringArg(args, "next_billing_date"); ok {
		input.NextBillingDate = &value
	}
	if value, ok := readNullableIntArg(args, "monthly_day"); ok {
		input.MonthlyDay = value
	}
	if value, ok := readNullableIntArg(args, "yearly_month"); ok {
		input.YearlyMonth = value
	}
	if value, ok := readNullableIntArg(args, "yearly_day"); ok {
		input.YearlyDay = value
	}
	if value, ok := readStringArg(args, "category"); ok {
		input.Category = &value
	}
	if value, ok := readNullableUintArg(args, "category_id"); ok {
		input.CategoryIDSet = true
		input.CategoryID = value
	}
	if value, ok := readNullableUintArg(args, "payment_method_id"); ok {
		input.PaymentMethodIDSet = true
		input.PaymentMethodID = value
	}
	if value, ok := readNullableBoolArg(args, "notify_enabled"); ok {
		input.NotifyEnabledSet = true
		input.NotifyEnabled = value
	}
	if value, ok := readNullableIntArg(args, "notify_days_before"); ok {
		input.NotifyDaysBeforeSet = true
		input.NotifyDaysBefore = value
	}
	if value, ok := readStringArg(args, "icon"); ok {
		input.Icon = &value
	}
	if value, ok := readStringArg(args, "url"); ok {
		input.URL = &value
	}
	if value, ok := readStringArg(args, "notes"); ok {
		input.Notes = &value
	}
	return input, nil
}

func validateSubscriptionWriteArgTypes(args map[string]interface{}) error {
	return validateMCPArgTypes(args, []mcpArgSpec{
		{Key: "id", Type: "integer"},
		{Key: "name", Type: "string"},
		{Key: "amount", Type: "number"},
		{Key: "currency", Type: "string"},
		{Key: "status", Type: "string"},
		{Key: "renewal_mode", Type: "string"},
		{Key: "ends_at", Type: "string", Nullable: true},
		{Key: "billing_type", Type: "string"},
		{Key: "recurrence_type", Type: "string"},
		{Key: "interval_count", Type: "integer", Nullable: true},
		{Key: "interval_unit", Type: "string"},
		{Key: "next_billing_date", Type: "string", Nullable: true},
		{Key: "monthly_day", Type: "integer", Nullable: true},
		{Key: "yearly_month", Type: "integer", Nullable: true},
		{Key: "yearly_day", Type: "integer", Nullable: true},
		{Key: "category", Type: "string"},
		{Key: "category_id", Type: "integer", Nullable: true},
		{Key: "payment_method_id", Type: "integer", Nullable: true},
		{Key: "notify_enabled", Type: "boolean", Nullable: true},
		{Key: "notify_days_before", Type: "integer", Nullable: true},
		{Key: "icon", Type: "string"},
		{Key: "url", Type: "string"},
		{Key: "notes", Type: "string"},
	})
}

type mcpArgSpec struct {
	Key      string
	Type     string
	Nullable bool
}

func validateMCPArgTypes(args map[string]interface{}, specs []mcpArgSpec) error {
	for _, spec := range specs {
		value, exists := args[spec.Key]
		if !exists {
			continue
		}
		if value == nil {
			if spec.Nullable {
				continue
			}
			return fmt.Errorf("%s must be %s", spec.Key, spec.Type)
		}

		var ok bool
		switch spec.Type {
		case "string":
			_, ok = value.(string)
		case "number":
			_, ok = readFloatArg(args, spec.Key)
		case "integer":
			_, ok = readIntArg(args, spec.Key)
		case "boolean":
			_, ok = readBoolArg(args, spec.Key)
		default:
			ok = true
		}
		if !ok {
			return fmt.Errorf("%s must be %s", spec.Key, spec.Type)
		}
	}
	return nil
}

func readRequiredIDArg(args map[string]interface{}, key string) (uint, error) {
	value, ok := readIntArg(args, key)
	if !ok {
		return 0, fmt.Errorf("%s is required", key)
	}
	if value < 1 {
		return 0, nil
	}
	return uint(value), nil
}

func readStringArgOrDefault(args map[string]interface{}, key, fallback string) string {
	if value, ok := readStringArg(args, key); ok {
		return value
	}
	return fallback
}

func readFloatArgOrDefault(args map[string]interface{}, key string, fallback float64) float64 {
	if value, ok := readFloatArg(args, key); ok {
		return value
	}
	return fallback
}

func readStringArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return "", false
	}
	switch typed := value.(type) {
	case string:
		return typed, true
	default:
		return fmt.Sprint(typed), true
	}
}

func readNullableStringArg(args map[string]interface{}, key string) (string, bool) {
	value, ok := args[key]
	if !ok {
		return "", false
	}
	if value == nil {
		return "", true
	}
	return readStringArg(args, key)
}

func readFloatArg(args map[string]interface{}, key string) (float64, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func readIntArg(args map[string]interface{}, key string) (int, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		asInt := int(typed)
		return asInt, typed == float64(asInt)
	case int:
		return typed, true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return 0, false
	}
}

func readNullableIntArg(args map[string]interface{}, key string) (*int, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readIntArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readIntPointerArg(args map[string]interface{}, key string) (*int, bool) {
	parsed, ok := readIntArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readUintArg(args map[string]interface{}, key string) (uint, bool) {
	parsed, ok := readIntArg(args, key)
	if !ok || parsed < 0 {
		return 0, false
	}
	return uint(parsed), true
}

func readNullableUintArg(args map[string]interface{}, key string) (*uint, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readUintArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readUintPointerArg(args map[string]interface{}, key string) (*uint, bool) {
	parsed, ok := readUintArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readBoolPointerArg(args map[string]interface{}, key string) (*bool, bool) {
	parsed, ok := readBoolArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readNullableBoolArg(args map[string]interface{}, key string) (*bool, bool) {
	value, ok := args[key]
	if !ok {
		return nil, false
	}
	if value == nil {
		return nil, true
	}
	parsed, ok := readBoolArg(args, key)
	if !ok {
		return nil, false
	}
	return &parsed, true
}

func readBoolArg(args map[string]interface{}, key string) (bool, bool) {
	value, ok := args[key]
	if !ok || value == nil {
		return false, false
	}
	switch typed := value.(type) {
	case bool:
		return typed, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		return parsed, err == nil
	default:
		return false, false
	}
}

func invalidMCPParams(err error) *mcpError {
	return &mcpError{Code: -32602, Message: err.Error()}
}

func internalMCPError(err error) *mcpError {
	return &mcpError{Code: -32603, Message: "internal server error", Data: err.Error()}
}

func mcpSuccessResponse(id json.RawMessage, result interface{}) mcpResponse {
	return mcpResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func mcpErrorResponse(id json.RawMessage, code int, message string, data interface{}) mcpResponse {
	return mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func mcpStructuredResult(data interface{}) *mcpToolResult {
	text, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		text = []byte("{}")
	}
	return &mcpToolResult{
		Content: []mcpTextContent{{
			Type: "text",
			Text: string(text),
		}},
		StructuredContent: data,
	}
}

func mcpToolExecutionError(message string) *mcpToolResult {
	return &mcpToolResult{
		Content: []mcpTextContent{{
			Type: "text",
			Text: message,
		}},
		StructuredContent: map[string]interface{}{"error": message},
		IsError:           true,
	}
}

func (h *MCPHandler) buildTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "list_subscriptions",
			Title:       "List Subscriptions",
			Description: "List subscriptions owned by the authenticated user.",
			InputSchema: objectSchema(map[string]interface{}{}, nil),
			Annotations: readOnlyToolAnnotation(),
		},
		{
			Name:        "get_subscription",
			Title:       "Get Subscription",
			Description: "Get one subscription by ID.",
			InputSchema: objectSchema(map[string]interface{}{
				"id": idSchema("Subscription ID."),
			}, []string{"id"}),
			Annotations: readOnlyToolAnnotation(),
		},
		{
			Name:        "create_subscription",
			Title:       "Create Subscription",
			Description: "Create a recurring subscription. If recurrence fields are omitted, it defaults to every 1 month.",
			InputSchema: subscriptionWriteInputSchema([]string{"name", "amount", "next_billing_date"}),
			Annotations: destructiveToolAnnotation(),
		},
		{
			Name:        "update_subscription",
			Title:       "Update Subscription",
			Description: "Update a subscription by ID. Send only fields that should change.",
			InputSchema: subscriptionWriteInputSchema([]string{"id"}),
			Annotations: destructiveToolAnnotation(),
		},
		{
			Name:        "delete_subscription",
			Title:       "Delete Subscription",
			Description: "Delete a subscription by ID.",
			InputSchema: objectSchema(map[string]interface{}{
				"id": idSchema("Subscription ID."),
			}, []string{"id"}),
			Annotations: destructiveToolAnnotation(),
		},
		{
			Name:        "mark_subscription_renewed",
			Title:       "Mark Subscription Renewed",
			Description: "Advance a manual-renew subscription to its next billing date.",
			InputSchema: objectSchema(map[string]interface{}{
				"id": idSchema("Subscription ID."),
			}, []string{"id"}),
			Annotations: destructiveToolAnnotation(),
		},
		{
			Name:        "get_dashboard_summary",
			Title:       "Get Dashboard Summary",
			Description: "Return dashboard spending totals. Defaults to the user's preferred currency.",
			InputSchema: objectSchema(map[string]interface{}{
				"currency": stringSchema("Optional target currency code, such as USD or CNY."),
			}, nil),
			Annotations: readOnlyToolAnnotation(),
		},
		{
			Name:        "list_categories",
			Title:       "List Categories",
			Description: "List subscription categories owned by the authenticated user.",
			InputSchema: objectSchema(map[string]interface{}{}, nil),
			Annotations: readOnlyToolAnnotation(),
		},
		{
			Name:        "list_payment_methods",
			Title:       "List Payment Methods",
			Description: "List payment methods owned by the authenticated user.",
			InputSchema: objectSchema(map[string]interface{}{}, nil),
			Annotations: readOnlyToolAnnotation(),
		},
	}
}

func subscriptionWriteInputSchema(required []string) map[string]interface{} {
	properties := map[string]interface{}{
		"id":                 idSchema("Subscription ID. Required for updates."),
		"name":               stringSchema("Subscription name."),
		"amount":             numberSchema("Subscription amount."),
		"currency":           stringSchema("Currency code, such as USD or CNY."),
		"status":             enumSchema("Subscription status.", []string{"active", "ended"}),
		"renewal_mode":       enumSchema("Renewal mode.", []string{"auto_renew", "manual_renew", "cancel_at_period_end"}),
		"ends_at":            nullableStringSchema("End date in YYYY-MM-DD format. Use null to clear."),
		"billing_type":       enumSchema("Billing type. Only recurring is supported.", []string{"recurring"}),
		"recurrence_type":    enumSchema("Recurrence type.", []string{"interval", "monthly_date", "yearly_date"}),
		"interval_count":     nullableIntegerSchema("Interval count for interval recurrence."),
		"interval_unit":      enumSchema("Interval unit.", []string{"day", "week", "month", "year"}),
		"next_billing_date":  stringSchema("Next billing date in YYYY-MM-DD format."),
		"monthly_day":        nullableIntegerSchema("Day of month for monthly_date recurrence."),
		"yearly_month":       nullableIntegerSchema("Month number for yearly_date recurrence."),
		"yearly_day":         nullableIntegerSchema("Day of month for yearly_date recurrence."),
		"category":           stringSchema("Legacy category label. Prefer category_id when available."),
		"category_id":        nullableIntegerSchema("Category ID. Use null to clear on update."),
		"payment_method_id":  nullableIntegerSchema("Payment method ID. Use null to clear on update."),
		"notify_enabled":     nullableBoolSchema("Notification override. Use null for default policy."),
		"notify_days_before": nullableIntegerSchema("Notification lead time, 0-10 days."),
		"icon":               stringSchema("Emoji, icon identifier, managed file, or image URL."),
		"url":                stringSchema("Related website URL."),
		"notes":              stringSchema("Free-form notes."),
	}
	return objectSchema(properties, required)
}

func objectSchema(properties map[string]interface{}, required []string) map[string]interface{} {
	schema := map[string]interface{}{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": description}
}

func nullableStringSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": []string{"string", "null"}, "description": description}
}

func integerSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description, "minimum": 0}
}

func idSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description}
}

func nullableIntegerSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": []string{"integer", "null"}, "description": description, "minimum": 0}
}

func numberSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "number", "description": description, "minimum": 0}
}

func nullableBoolSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": []string{"boolean", "null"}, "description": description}
}

func enumSchema(description string, values []string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": description, "enum": values}
}

func readOnlyToolAnnotation() map[string]interface{} {
	return map[string]interface{}{
		"readOnlyHint":    true,
		"destructiveHint": false,
	}
}

func destructiveToolAnnotation() map[string]interface{} {
	return map[string]interface{}{
		"readOnlyHint":    false,
		"destructiveHint": true,
	}
}
