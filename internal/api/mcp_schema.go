package api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shiroha/subdux/internal/service"
	"github.com/shiroha/subdux/internal/version"
)

type mcpToolHandler func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError)

type mcpToolDefinition struct {
	Name        string
	Title       string
	Description string
	InputSchema func() map[string]interface{}
	Write       bool
	Handler     mcpToolHandler
}

func mcpToolDefinitions() []mcpToolDefinition {
	return []mcpToolDefinition{
		{
			Name:        "list_subscriptions",
			Title:       "List Subscriptions",
			Description: "List subscriptions owned by the authenticated user.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{}, nil)
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callListSubscriptions(principal.UserID)
			},
		},
		{
			Name:        "search_subscriptions",
			Title:       "Search Subscriptions",
			Description: "Search subscriptions by text and optional filters. Text matches name, category, currency, status, renewal mode, billing type, recurrence type, URL, and notes.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{
					"query":             stringSchema("Optional case-insensitive text query."),
					"status":            enumSchema("Optional subscription status.", []string{"active", "ended"}),
					"currency":          stringSchema("Optional currency code, such as USD or CNY."),
					"renewal_mode":      enumSchema("Optional renewal mode.", []string{"auto_renew", "manual_renew", "cancel_at_period_end"}),
					"billing_type":      enumSchema("Optional billing type.", []string{"recurring"}),
					"recurrence_type":   enumSchema("Optional recurrence type.", []string{"interval", "monthly_date", "yearly_date"}),
					"category":          stringSchema("Optional category name substring. Matches the category_id label and legacy category label."),
					"category_id":       nullableIntegerSchema("Optional category ID. Use null to find subscriptions without a category."),
					"payment_method_id": nullableIntegerSchema("Optional payment method ID. Use null to find subscriptions without a payment method."),
					"next_billing_from": stringSchema("Optional inclusive next billing start date in YYYY-MM-DD format."),
					"next_billing_to":   stringSchema("Optional inclusive next billing end date in YYYY-MM-DD format."),
					"limit":             integerRangeSchema("Maximum number of subscriptions to return. Defaults to 20.", 1, 100),
				}, nil)
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callSearchSubscriptions(principal.UserID, args)
			},
		},
		{
			Name:        "get_subscription",
			Title:       "Get Subscription",
			Description: "Get one subscription by ID.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{
					"id": idSchema("Subscription ID."),
				}, []string{"id"})
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callGetSubscription(principal.UserID, args)
			},
		},
		{
			Name:        "create_subscription",
			Title:       "Create Subscription",
			Description: "Create a recurring subscription. If recurrence fields are omitted, it defaults to every 1 month.",
			InputSchema: func() map[string]interface{} {
				return subscriptionWriteInputSchema([]string{"name", "amount", "next_billing_date"})
			},
			Write: true,
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callCreateSubscription(principal, args)
			},
		},
		{
			Name:        "update_subscription",
			Title:       "Update Subscription",
			Description: "Update a subscription by ID. Send only fields that should change.",
			InputSchema: func() map[string]interface{} {
				return subscriptionWriteInputSchema([]string{"id"})
			},
			Write: true,
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callUpdateSubscription(principal, args)
			},
		},
		{
			Name:        "delete_subscription",
			Title:       "Delete Subscription",
			Description: "Delete a subscription by ID.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{
					"id": idSchema("Subscription ID."),
				}, []string{"id"})
			},
			Write: true,
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callDeleteSubscription(principal, args)
			},
		},
		{
			Name:        "mark_subscription_renewed",
			Title:       "Mark Subscription Renewed",
			Description: "Advance a manual-renew subscription to its next billing date.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{
					"id": idSchema("Subscription ID."),
				}, []string{"id"})
			},
			Write: true,
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callMarkSubscriptionRenewed(principal, args)
			},
		},
		{
			Name:        "get_dashboard_summary",
			Title:       "Get Dashboard Summary",
			Description: "Return dashboard spending totals. Defaults to the user's preferred currency.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{
					"currency": stringSchema("Optional target currency code, such as USD or CNY."),
				}, nil)
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callDashboardSummary(principal.UserID, args)
			},
		},
		{
			Name:        "list_categories",
			Title:       "List Categories",
			Description: "List subscription categories owned by the authenticated user.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{}, nil)
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callListCategories(principal.UserID)
			},
		},
		{
			Name:        "list_payment_methods",
			Title:       "List Payment Methods",
			Description: "List payment methods owned by the authenticated user.",
			InputSchema: func() map[string]interface{} {
				return objectSchema(map[string]interface{}{}, nil)
			},
			Handler: func(h *MCPHandler, principal *mcpPrincipal, args map[string]interface{}) (*mcpToolResult, *mcpError) {
				return h.callListPaymentMethods(principal.UserID)
			},
		},
	}
}

func (d mcpToolDefinition) sdkTool() *mcp.Tool {
	annotations := readOnlySDKToolAnnotation()
	if d.Write {
		annotations = writeSDKToolAnnotation(d.Name == "delete_subscription")
	}
	return &mcp.Tool{
		Name:        d.Name,
		Title:       d.Title,
		Description: d.Description,
		InputSchema: d.InputSchema(),
		Annotations: annotations,
	}
}

func mcpToolDefinitionByName(name string) (mcpToolDefinition, bool) {
	for _, definition := range mcpToolDefinitions() {
		if definition.Name == name {
			return definition, true
		}
	}
	return mcpToolDefinition{}, false
}

func (h *MCPHandler) buildServer() *mcp.Server {
	info := version.Get()
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "subdux",
			Title:   "Subdux",
			Version: info.Version,
		},
		&mcp.ServerOptions{
			Instructions: "Use X-API-Key authentication. Read tools require the read scope; write tools require the write scope.",
			Capabilities: &mcp.ServerCapabilities{
				Tools: &mcp.ToolCapabilities{},
			},
			GetSessionID: func() string { return "" },
		},
	)

	for _, definition := range mcpToolDefinitions() {
		definition := definition
		server.AddTool(definition.sdkTool(), func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			principal, ok := ctx.Value(mcpPrincipalContextKey{}).(*mcpPrincipal)
			if !ok || principal == nil {
				return nil, newMCPJSONRPCError(jsonrpc.CodeInvalidRequest, "missing mcp principal").sdkError()
			}

			args := map[string]interface{}{}
			if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
				if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
					return nil, newMCPJSONRPCError(jsonrpc.CodeInvalidParams, "invalid tool call params").sdkError()
				}
			}

			requiredScope := service.APIKeyScopeRead
			if definition.Write {
				requiredScope = service.APIKeyScopeWrite
			}
			if !mcpPrincipalHasScope(principal, requiredScope) {
				return mcpToolExecutionError("api key does not have required scope").sdkResult(), nil
			}

			result, rpcErr := definition.Handler(h, principal, args)
			if rpcErr != nil {
				return nil, rpcErr.sdkError()
			}
			return result.sdkResult(), nil
		})
	}

	return server
}

func newMCPJSONRPCError(code int64, message string) *mcpError {
	return &mcpError{Code: int(code), Message: message}
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
	definition, ok := mcpToolDefinitionByName(name)
	return ok && definition.Write
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
		"notify_days_before": nullableIntegerRangeSchema("Notification lead time, 0-10 days.", 0, 10),
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

func integerRangeSchema(description string, minimum, maximum int) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description, "minimum": minimum, "maximum": maximum}
}

func idSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": "integer", "description": description}
}

func nullableIntegerSchema(description string) map[string]interface{} {
	return map[string]interface{}{"type": []string{"integer", "null"}, "description": description, "minimum": 0}
}

func nullableIntegerRangeSchema(description string, minimum, maximum int) map[string]interface{} {
	return map[string]interface{}{
		"anyOf": []map[string]interface{}{
			{"type": "integer", "minimum": minimum, "maximum": maximum},
			{"type": "null"},
		},
		"description": description,
	}
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

func (e *mcpError) sdkError() error {
	if e == nil {
		return nil
	}

	wireErr := &jsonrpc.Error{
		Code:    int64(e.Code),
		Message: e.Message,
	}
	if e.Data != nil {
		if data, err := json.Marshal(e.Data); err == nil {
			wireErr.Data = data
		}
	}
	return wireErr
}

func sdkBoolPointer(value bool) *bool {
	return &value
}

func readOnlySDKToolAnnotation() *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    true,
		DestructiveHint: sdkBoolPointer(false),
		OpenWorldHint:   sdkBoolPointer(false),
	}
}

func writeSDKToolAnnotation(destructive bool) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: sdkBoolPointer(destructive),
		OpenWorldHint:   sdkBoolPointer(false),
	}
}

var errMissingMCPResult = errors.New("missing mcp tool result")
