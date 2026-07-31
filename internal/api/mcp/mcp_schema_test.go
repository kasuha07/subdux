package mcp

import (
	"net/http"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
)

func TestMCPToolsListDeclaresSubscriptionAmountMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("tools/list result = %#v, want object", response["result"])
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("tools/list tools = %#v, want array", result["tools"])
	}

	wanted := map[string]bool{
		"create_subscription": false,
		"update_subscription": false,
	}
	for _, item := range tools {
		tool, ok := item.(map[string]interface{})
		if !ok {
			t.Fatalf("tool = %#v, want object", item)
		}
		name, _ := tool["name"].(string)
		if _, tracked := wanted[name]; !tracked {
			continue
		}

		inputSchema, ok := tool["inputSchema"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s inputSchema = %#v, want object", name, tool["inputSchema"])
		}
		properties, ok := inputSchema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s properties = %#v, want object", name, inputSchema["properties"])
		}
		amount, ok := properties["amount"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s amount schema = %#v, want object", name, properties["amount"])
		}
		if got, ok := amount["maximum"].(float64); !ok || got != float64(money.MaxAmount) {
			t.Fatalf("%s amount maximum = %#v, want %v", name, amount["maximum"], money.MaxAmount)
		}
		wanted[name] = true
	}

	for name, found := range wanted {
		if !found {
			t.Fatalf("tools/list missing %s amount.maximum; tools = %#v", name, tools)
		}
	}
}

func TestMCPToolsCallRejectsSubscriptionAmountAboveMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "create_subscription",
			"arguments": map[string]interface{}{
				"idempotency_key":   "amount-above-maximum",
				"name":              "Too Large",
				"amount":            money.MaxAmount + 1,
				"next_billing_date": "2026-08-15",
			},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("tools/call status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	rpcError, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("tools/call response = %#v, want SDK argument-validation error", response)
	}
	if got := int(rpcError["code"].(float64)); got != -32602 {
		t.Fatalf("tools/call error code = %d, want -32602; error = %#v", got, rpcError)
	}
	if got := rpcError["message"]; got != "amount is too large" {
		t.Fatalf("tools/call error message = %v, want amount is too large", got)
	}
	data, ok := rpcError["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("tools/call error data = %#v, want object", rpcError["data"])
	}
	if got := data["error_code"]; got != "amount_too_large" {
		t.Fatalf("tools/call error_code = %#v, want amount_too_large", got)
	}

	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0 after rejected tools/call", count)
	}
}
