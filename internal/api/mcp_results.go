package api

import "encoding/json"

type mcpToolResult struct {
	Content           []mcpTextContent `json:"content"`
	StructuredContent interface{}      `json:"structuredContent,omitempty"`
	IsError           bool             `json:"isError,omitempty"`
}

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
