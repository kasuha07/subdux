package mcp

import (
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
	"gorm.io/gorm"
)

func TestReadFloatArgRejectsNonFiniteValues(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  float64
		wantO bool
	}{
		{name: "plain number", value: 12.5, want: 12.5, wantO: true},
		{name: "integer", value: 3, want: 3, wantO: true},
		{name: "numeric string", value: "12.5", want: 12.5, wantO: true},
		{name: "nan string", value: "NaN"},
		{name: "lowercase nan string", value: "nan"},
		{name: "infinity string", value: "Infinity"},
		{name: "inf string", value: "Inf"},
		{name: "negative infinity string", value: "-Inf"},
		{name: "nan float", value: math.NaN()},
		{name: "positive infinity float", value: math.Inf(1)},
		{name: "negative infinity float", value: math.Inf(-1)},
		{name: "garbage string", value: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := readFloatArg(map[string]interface{}{"amount": tt.value}, "amount")
			if ok != tt.wantO {
				t.Fatalf("readFloatArg() ok = %v, want %v", ok, tt.wantO)
			}
			if got != tt.want {
				t.Fatalf("readFloatArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSubscriptionWriteArgTypesRejectsNonFiniteAmount(t *testing.T) {
	for _, value := range []string{"NaN", "Infinity", "-Infinity"} {
		t.Run(value, func(t *testing.T) {
			err := validateSubscriptionWriteArgTypes(map[string]interface{}{"amount": value})
			if err == nil {
				t.Fatalf("validateSubscriptionWriteArgTypes(amount=%q) error = nil, want rejection", value)
			}
			if !strings.Contains(err.Error(), "amount must be number") {
				t.Fatalf("validateSubscriptionWriteArgTypes(amount=%q) error = %q, want amount type error", value, err.Error())
			}
		})
	}
}

func TestMCPCreateSubscriptionRejectsNonFiniteAmount(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	for _, tt := range []struct {
		value    string
		wantCode string
	}{
		{value: "NaN", wantCode: "amount_must_be_finite"},
		{value: "Infinity", wantCode: "amount_too_large"},
		{value: "-Infinity", wantCode: "amount_must_be_finite"},
	} {
		t.Run(tt.value, func(t *testing.T) {
			rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
				"idempotency_key":   "create-nonfinite-" + tt.value,
				"name":              "Broken Plan",
				"amount":            tt.value,
				"next_billing_date": "2026-06-15",
			})
			assertMCPInvalidAmountRPCError(t, rec, response, tt.wantCode)
		})
	}

	assertMCPSubscriptionCount(t, db, user.ID, 0)
}

func TestMCPUpdateSubscriptionRejectsNonFiniteAmount(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-for-update-nonfinite",
		"name":              "Claude Pro",
		"amount":            20,
		"next_billing_date": "2026-06-15",
	})
	assertMCPToolSuccess(t, rec, response)

	for _, tt := range []struct {
		value    string
		wantCode string
	}{
		{value: "NaN", wantCode: "amount_must_be_finite"},
		{value: "Infinity", wantCode: "amount_too_large"},
		{value: "-Infinity", wantCode: "amount_must_be_finite"},
	} {
		t.Run(tt.value, func(t *testing.T) {
			rec, response := performMCPToolCall(t, handler, apiKey, "update_subscription", map[string]interface{}{
				"idempotency_key": "update-nonfinite-" + tt.value,
				"id":              1,
				"amount":          tt.value,
			})
			assertMCPInvalidAmountRPCError(t, rec, response, tt.wantCode)
		})
	}

	var stored model.Subscription
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("load subscription: %v", err)
	}
	if stored.Amount != 20 {
		t.Fatalf("stored amount = %v, want 20 after rejected updates", stored.Amount)
	}
}

func TestMCPCreateSubscriptionReturnsTypedAmountErrors(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	tests := []struct {
		name     string
		amount   float64
		wantCode string
	}{
		{name: "negative", amount: -1, wantCode: "amount_must_not_be_negative"},
		{name: "above maximum", amount: money.MaxAmount + 0.01, wantCode: "amount_too_large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
				"idempotency_key":   "create-typed-amount-error-" + tt.name,
				"name":              "Broken Plan",
				"amount":            tt.amount,
				"next_billing_date": "2026-06-15",
			})
			assertMCPInvalidAmountRPCError(t, rec, response, tt.wantCode)
		})
	}

	assertMCPSubscriptionCount(t, db, user.ID, 0)
	assertMCPIdempotencyCount(t, db, user.ID, 0)
}

func TestMCPUpdateSubscriptionReturnsTypedAmountErrors(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-for-typed-update",
		"name":              "Claude Pro",
		"amount":            20,
		"next_billing_date": "2026-06-15",
	})
	assertMCPToolSuccess(t, rec, response)

	tests := []struct {
		name     string
		amount   float64
		wantCode string
	}{
		{name: "negative", amount: -1, wantCode: "amount_must_not_be_negative"},
		{name: "above maximum", amount: money.MaxAmount + 0.01, wantCode: "amount_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, response := performMCPToolCall(t, handler, apiKey, "update_subscription", map[string]interface{}{
				"idempotency_key": "update-typed-amount-error-" + tt.name,
				"id":              1,
				"amount":          tt.amount,
			})
			assertMCPInvalidAmountRPCError(t, rec, response, tt.wantCode)
		})
	}

	var stored model.Subscription
	if err := db.First(&stored, 1).Error; err != nil {
		t.Fatalf("load subscription: %v", err)
	}
	if stored.Amount != 20 {
		t.Fatalf("stored amount = %v, want 20 after rejected updates", stored.Amount)
	}
	assertMCPIdempotencyCount(t, db, user.ID, 1)
}

func TestMCPCreateSubscriptionRejectsAmountAboveMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	for _, value := range []float64{money.MaxAmount + 0.01, 1.8e306} {
		rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
			"idempotency_key":   "create-too-large",
			"name":              "Huge Plan",
			"amount":            value,
			"next_billing_date": "2026-06-15",
		})
		assertMCPInvalidAmountRPCError(t, rec, response, "amount_too_large")
	}

	assertMCPSubscriptionCount(t, db, user.ID, 0)
}

func TestMCPCreateSubscriptionAcceptsMaximumAmount(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-at-maximum",
		"name":              "Maximum Plan",
		"amount":            money.MaxAmount,
		"next_billing_date": "2026-06-15",
	})
	assertMCPToolSuccess(t, rec, response)
	assertMCPSubscriptionCount(t, db, user.ID, 1)
}

func TestMCPUpdateSubscriptionRejectsAmountAboveMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)

	rec, response := performMCPToolCall(t, handler, apiKey, "create_subscription", map[string]interface{}{
		"idempotency_key":   "create-for-too-large-update",
		"name":              "Claude Pro",
		"amount":            20,
		"next_billing_date": "2026-06-15",
	})
	assertMCPToolSuccess(t, rec, response)

	rec, response = performMCPToolCall(t, handler, apiKey, "update_subscription", map[string]interface{}{
		"idempotency_key": "update-too-large",
		"id":              1,
		"amount":          money.MaxAmount + 0.01,
	})
	assertMCPInvalidAmountRPCError(t, rec, response, "amount_too_large")
	assertMCPIdempotencyCount(t, db, user.ID, 1)
}

func assertMCPRPCError(t *testing.T, rec *httptest.ResponseRecorder, response map[string]interface{}, wantMessage string) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("MCP status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	rpcError, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("response = %#v, want JSON-RPC error", response)
	}
	if got := int(rpcError["code"].(float64)); got != -32602 {
		t.Fatalf("JSON-RPC error code = %d, want -32602", got)
	}
	if got := rpcError["message"]; got != wantMessage {
		t.Fatalf("JSON-RPC error message = %v, want %q", got, wantMessage)
	}
}

func assertMCPInvalidAmountRPCError(t *testing.T, rec *httptest.ResponseRecorder, response map[string]interface{}, wantCode string) {
	t.Helper()
	assertMCPRPCError(t, rec, response, amountErrorMessage(wantCode))
	rpcError := response["error"].(map[string]interface{})
	data, ok := rpcError["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("JSON-RPC error data = %#v, want object", rpcError["data"])
	}
	if got := data["error_code"]; got != wantCode {
		t.Fatalf("JSON-RPC error_code = %#v, want %q", got, wantCode)
	}
}

func amountErrorMessage(code string) string {
	switch code {
	case "amount_must_be_finite":
		return "amount must be finite"
	case "amount_must_not_be_negative":
		return "amount must not be negative"
	case "amount_too_large":
		return "amount is too large"
	default:
		return code
	}
}

func assertMCPToolSuccess(t *testing.T, rec *httptest.ResponseRecorder, response map[string]interface{}) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("MCP status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rpcError, ok := response["error"]; ok {
		t.Fatalf("JSON-RPC error = %#v, want successful tool result", rpcError)
	}
	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("result = %#v, want object", response["result"])
	}
	if result["isError"] == true {
		t.Fatalf("result = %#v, want successful tool result", result)
	}
}

func assertMCPSubscriptionCount(t *testing.T, db *gorm.DB, userID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	if count != want {
		t.Fatalf("subscription count = %d, want %d", count, want)
	}
}

func assertMCPIdempotencyCount(t *testing.T, db *gorm.DB, userID uint, want int64) {
	t.Helper()
	var count int64
	if err := db.Model(&model.MCPIdempotencyKey{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count idempotency records: %v", err)
	}
	if count != want {
		t.Fatalf("idempotency record count = %d, want %d", count, want)
	}
}
