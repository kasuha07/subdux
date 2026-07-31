package mcp

import (
	"context"
	"math"
	"strings"
	"testing"

	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"github.com/kasuha07/subdux/internal/service/money"
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
	handler := newMCPTestHandler(db)
	principal := &mcpPrincipal{
		UserID:  user.ID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}

	for _, value := range []interface{}{"NaN", "Infinity", math.NaN(), math.Inf(1)} {
		_, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
			"idempotency_key":   "create-nonfinite",
			"name":              "Broken Plan",
			"amount":            value,
			"next_billing_date": "2026-06-15",
		})
		if rpcErr == nil {
			t.Fatalf("callCreateSubscription(amount=%v) rpcErr = nil, want rejection", value)
		}
	}
}

func TestMCPUpdateSubscriptionRejectsNonFiniteAmount(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := &mcpPrincipal{
		UserID:  user.ID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}

	created, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key":   "create-for-update",
		"name":              "Claude Pro",
		"amount":            20,
		"next_billing_date": "2026-06-15",
	})
	if rpcErr != nil {
		t.Fatalf("callCreateSubscription() rpcErr = %v", rpcErr)
	}
	if created == nil || created.IsError {
		t.Fatalf("create result = %#v, want success", created)
	}

	for _, value := range []interface{}{"NaN", math.Inf(-1)} {
		_, rpcErr := handler.callUpdateSubscription(context.Background(), principal, map[string]interface{}{
			"idempotency_key": "update-nonfinite",
			"id":              1,
			"amount":          value,
		})
		if rpcErr == nil {
			t.Fatalf("callUpdateSubscription(amount=%v) rpcErr = nil, want rejection", value)
		}
	}
}

func TestMCPCreateSubscriptionReturnsTypedAmountErrors(t *testing.T) {
	testMCPAmountErrors(t, "create", func(t *testing.T, handler *MCPHandler, principal *mcpPrincipal, value float64) *mcpError {
		t.Helper()
		_, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
			"idempotency_key":   "create-typed-amount-error",
			"name":              "Broken Plan",
			"amount":            value,
			"next_billing_date": "2026-06-15",
		})
		return rpcErr
	})
}

func TestMCPUpdateSubscriptionReturnsTypedAmountErrors(t *testing.T) {
	testMCPAmountErrors(t, "update", func(t *testing.T, handler *MCPHandler, principal *mcpPrincipal, value float64) *mcpError {
		t.Helper()
		_, rpcErr := handler.callUpdateSubscription(context.Background(), principal, map[string]interface{}{
			"idempotency_key": "update-typed-amount-error",
			"id":              1,
			"amount":          value,
		})
		return rpcErr
	})
}

func testMCPAmountErrors(t *testing.T, operation string, call func(*testing.T, *MCPHandler, *mcpPrincipal, float64) *mcpError) {
	t.Helper()
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := mcpWriteTestPrincipal(user.ID)

	tests := []struct {
		name     string
		amount   float64
		wantCode string
	}{
		{name: "nan", amount: math.NaN(), wantCode: "amount_must_be_finite"},
		{name: "negative infinity", amount: math.Inf(-1), wantCode: "amount_must_be_finite"},
		{name: "negative", amount: -1, wantCode: "amount_must_not_be_negative"},
		{name: "above maximum", amount: money.MaxAmount + 0.01, wantCode: "amount_too_large"},
		{name: "positive infinity", amount: math.Inf(1), wantCode: "amount_too_large"},
	}

	for _, tt := range tests {
		t.Run(operation+"/"+tt.name, func(t *testing.T) {
			rpcErr := call(t, handler, principal, tt.amount)
			if rpcErr == nil {
				t.Fatal("rpcErr = nil, want invalid params")
			}
			if rpcErr.Code != -32602 {
				t.Fatalf("rpcErr.Code = %d, want -32602", rpcErr.Code)
			}
			data, ok := rpcErr.Data.(map[string]interface{})
			if !ok {
				t.Fatalf("rpcErr.Data type = %T, want map", rpcErr.Data)
			}
			if got := data["error_code"]; got != tt.wantCode {
				t.Fatalf("error_code = %#v, want %q", got, tt.wantCode)
			}
		})
	}
}

func TestMCPCreateSubscriptionRejectsAmountAboveMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := &mcpPrincipal{
		UserID:  user.ID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}

	for _, value := range []interface{}{money.MaxAmount + 0.01, 1.8e306} {
		_, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
			"idempotency_key":   "create-too-large",
			"name":              "Huge Plan",
			"amount":            value,
			"next_billing_date": "2026-06-15",
		})
		if rpcErr == nil {
			t.Fatalf("callCreateSubscription(amount=%v) rpcErr = nil, want rejection", value)
		}
		if !strings.Contains(rpcErr.Message, "amount is too large") {
			t.Fatalf("callCreateSubscription(amount=%v) message = %q, want the too-large reason", value, rpcErr.Message)
		}
	}
}

func TestMCPCreateSubscriptionAcceptsMaximumAmount(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := &mcpPrincipal{
		UserID:  user.ID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}

	result, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key":   "create-at-maximum",
		"name":              "Maximum Plan",
		"amount":            money.MaxAmount,
		"next_billing_date": "2026-06-15",
	})
	if rpcErr != nil {
		t.Fatalf("callCreateSubscription() rpcErr = %v, want the bound itself accepted", rpcErr)
	}
	if result == nil || result.IsError {
		t.Fatalf("create result = %#v, want success", result)
	}
}

func TestMCPUpdateSubscriptionRejectsAmountAboveMaximum(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := &mcpPrincipal{
		UserID:  user.ID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}

	created, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key":   "create-for-too-large-update",
		"name":              "Claude Pro",
		"amount":            20,
		"next_billing_date": "2026-06-15",
	})
	if rpcErr != nil {
		t.Fatalf("callCreateSubscription() rpcErr = %v", rpcErr)
	}
	if created == nil || created.IsError {
		t.Fatalf("create result = %#v, want success", created)
	}

	_, rpcErr = handler.callUpdateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key": "update-too-large",
		"id":              1,
		"amount":          money.MaxAmount + 0.01,
	})
	if rpcErr == nil {
		t.Fatal("callUpdateSubscription() rpcErr = nil, want rejection above the maximum amount")
	}
	if !strings.Contains(rpcErr.Message, "amount is too large") {
		t.Fatalf("callUpdateSubscription() message = %q, want the too-large reason", rpcErr.Message)
	}
}
