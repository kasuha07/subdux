package api

import (
	"encoding/json"
	"strings"

	"github.com/shiroha/subdux/internal/service"
)

type mcpToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
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

	definition, ok := mcpToolDefinitionByName(params.Name)
	if !ok {
		return nil, &mcpError{Code: -32602, Message: "unknown tool: " + params.Name}
	}
	return definition.Handler(h, principal.UserID, params.Arguments)
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
