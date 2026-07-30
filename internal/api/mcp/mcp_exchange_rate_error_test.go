package mcp

import (
	"net/http"
	"testing"

	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestMCPDashboardExchangeRateUnavailableReturnsToolExecutionError(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	monthly := 1
	if _, err := subscriptionservice.NewService(db).Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:            "Euro Subscription",
		Amount:          12,
		Currency:        "EUR",
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &monthly,
		IntervalUnit:    "month",
		NextBillingDate: "2026-08-15",
	}); err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	rec, response := performMCPRequest(t, handler, apiKey, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "get_dashboard_summary",
			"arguments": map[string]interface{}{},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rpcError, exists := response["error"]; exists {
		t.Fatalf("JSON-RPC error = %#v, want tool execution error", rpcError)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result = %#v, want object", response["result"])
	}
	if result["isError"] != true {
		t.Fatalf("isError = %v, want true; result = %#v", result["isError"], result)
	}
	structured, ok := result["structuredContent"].(map[string]interface{})
	if !ok {
		t.Fatalf("structuredContent = %#v, want object", result["structuredContent"])
	}
	if got := structured["error_code"]; got != subscriptionservice.ErrExchangeRateUnavailable.Code {
		t.Fatalf("error_code = %v, want %q", got, subscriptionservice.ErrExchangeRateUnavailable.Code)
	}
	if got := structured["error"]; got != subscriptionservice.ErrExchangeRateUnavailable.Error() {
		t.Fatalf("error = %v, want %q", got, subscriptionservice.ErrExchangeRateUnavailable.Error())
	}
}
