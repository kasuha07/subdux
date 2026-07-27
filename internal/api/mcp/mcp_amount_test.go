package mcp

import (
	"context"
	"math"
	"strings"
	"testing"

	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
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

func TestSubscriptionAmountErrorNamesTheActualReason(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   string
	}{
		{name: "negative", amount: -1, want: "amount must not be negative"},
		{name: "above the maximum", amount: 1_000_000_000_000.01, want: "amount is too large"},
		{name: "overflowing rounding", amount: 1.8e306, want: "amount is too large"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := subscriptionAmountError(tt.amount).Error(); got != tt.want {
				t.Fatalf("subscriptionAmountError(%v) = %q, want %q", tt.amount, got, tt.want)
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

	for _, value := range []interface{}{1_000_000_000_000.01, 1.8e306} {
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
		"amount":            1_000_000_000_000.0,
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
		"amount":          1_000_000_000_000.01,
	})
	if rpcErr == nil {
		t.Fatal("callUpdateSubscription() rpcErr = nil, want rejection above the maximum amount")
	}
	if !strings.Contains(rpcErr.Message, "amount is too large") {
		t.Fatalf("callUpdateSubscription() message = %q, want the too-large reason", rpcErr.Message)
	}
}
