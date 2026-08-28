package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

const derivedAmountOverflowInput = 50_000_000_000

func TestMCPCreateSubscriptionReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-derived-too-large",
		"name":              "Large daily CLF plan",
		"amount":            derivedAmountOverflowInput,
		"currency":          "CLF",
		"interval_unit":     "day",
		"next_billing_date": "2026-08-15",
	})
	assertMCPToolErrorCode(t, rec, response, "amount_too_large")
	assertMCPSubscriptionCount(t, db, user.ID, 0)
	assertNoMCPIdempotencyRecord(t, db, user.ID, "create-derived-too-large")
}

func TestMCPUpdateSubscriptionReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-for-derived-update",
		"name":              "Large monthly CLF plan",
		"amount":            derivedAmountOverflowInput,
		"currency":          "CLF",
		"interval_unit":     "month",
		"next_billing_date": "2026-08-15",
	})
	assertMCPToolSuccess(t, rec, response)

	rec, response = performMCPToolCall(t, handler, apiKey, "update_subscription", map[string]interface{}{
		"idempotency_key": "update-derived-too-large",
		"id":              1,
		"interval_unit":   "day",
	})
	assertMCPToolErrorCode(t, rec, response, "amount_too_large")

	var stored model.Subscription
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("load subscription: %v", err)
	}
	if stored.IntervalUnit != "month" {
		t.Fatalf("stored interval_unit = %q, want month after rolled-back update", stored.IntervalUnit)
	}
	assertNoMCPIdempotencyRecord(t, db, user.ID, "update-derived-too-large")
}

func assertMCPToolErrorCode(t *testing.T, rec *httptest.ResponseRecorder, response map[string]interface{}, wantCode string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("MCP status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rpcError, ok := response["error"]; ok {
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
	if got := structured["error_code"]; got != wantCode {
		t.Fatalf("structured error_code = %#v, want %q", got, wantCode)
	}
}

func assertNoMCPIdempotencyRecord(t *testing.T, db *gorm.DB, userID uint, key string) {
	t.Helper()
	var count int64
	if err := db.Model(&model.MCPIdempotencyKey{}).
		Where("user_id = ? AND idempotency_key = ?", userID, key).
		Count(&count).Error; err != nil {
		t.Fatalf("count idempotency records: %v", err)
	}
	if count != 0 {
		t.Fatalf("idempotency record count = %d, want 0 after rolled-back write", count)
	}
}
