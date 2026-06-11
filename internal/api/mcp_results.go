package api

import (
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

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

func (r *mcpToolResult) sdkResult() *mcp.CallToolResult {
	if r == nil {
		result := &mcp.CallToolResult{}
		result.SetError(errMissingMCPResult)
		return result
	}

	content := make([]mcp.Content, 0, len(r.Content))
	for _, item := range r.Content {
		if item.Type == "text" {
			content = append(content, &mcp.TextContent{Text: item.Text})
		}
	}
	if len(content) == 0 {
		content = []mcp.Content{&mcp.TextContent{}}
	}

	return &mcp.CallToolResult{
		Content:           content,
		StructuredContent: r.StructuredContent,
		IsError:           r.IsError,
	}
}
