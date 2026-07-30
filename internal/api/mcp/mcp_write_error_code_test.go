package mcp

import (
	"context"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"gorm.io/gorm"
)

const derivedAmountOverflowInput = 50_000_000_000

func TestMCPCreateSubscriptionReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := mcpWriteTestPrincipal(user.ID)

	result, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key":   "create-derived-too-large",
		"name":              "Large daily CLF plan",
		"amount":            derivedAmountOverflowInput,
		"currency":          "CLF",
		"interval_unit":     "day",
		"next_billing_date": "2026-08-15",
	})
	assertMCPToolErrorCode(t, result, rpcErr, "amount_too_large")

	var subscriptionCount int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&subscriptionCount).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	if subscriptionCount != 0 {
		t.Fatalf("subscription count = %d, want 0 after rolled-back create", subscriptionCount)
	}
	assertNoMCPIdempotencyRecord(t, db, user.ID, "create-derived-too-large")
}

func TestMCPUpdateSubscriptionReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	handler := newMCPTestHandler(db)
	principal := mcpWriteTestPrincipal(user.ID)

	created, rpcErr := handler.callCreateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key":   "create-for-derived-update",
		"name":              "Large monthly CLF plan",
		"amount":            derivedAmountOverflowInput,
		"currency":          "CLF",
		"interval_unit":     "month",
		"next_billing_date": "2026-08-15",
	})
	if rpcErr != nil {
		t.Fatalf("create setup rpcErr = %v", rpcErr)
	}
	if created == nil || created.IsError {
		t.Fatalf("create setup result = %#v, want success", created)
	}

	var subscription model.Subscription
	if err := db.Where("user_id = ?", user.ID).First(&subscription).Error; err != nil {
		t.Fatalf("load created subscription: %v", err)
	}
	result, rpcErr := handler.callUpdateSubscription(context.Background(), principal, map[string]interface{}{
		"idempotency_key": "update-derived-too-large",
		"id":              float64(subscription.ID),
		"interval_unit":   "day",
	})
	assertMCPToolErrorCode(t, result, rpcErr, "amount_too_large")

	var stored model.Subscription
	if err := db.First(&stored, subscription.ID).Error; err != nil {
		t.Fatalf("load subscription: %v", err)
	}
	if stored.IntervalUnit != "month" {
		t.Fatalf("stored interval_unit = %q, want month after rolled-back update", stored.IntervalUnit)
	}
	assertNoMCPIdempotencyRecord(t, db, user.ID, "update-derived-too-large")
}

func mcpWriteTestPrincipal(userID uint) *mcpPrincipal {
	return &mcpPrincipal{
		UserID:  userID,
		KeyID:   7,
		KeyKind: apikeyservice.APIKeyKindMCPClient,
		Scopes:  []string{apikeyservice.APIKeyScopeRead, apikeyservice.APIKeyScopeWrite},
	}
}

func assertMCPToolErrorCode(t *testing.T, result *mcpToolResult, rpcErr *mcpError, wantCode string) {
	t.Helper()
	if rpcErr != nil {
		t.Fatalf("rpcErr = %v, want tool execution error", rpcErr)
	}
	if result == nil || !result.IsError {
		t.Fatalf("result = %#v, want isError tool result", result)
	}
	structured, ok := result.StructuredContent.(map[string]interface{})
	if !ok {
		t.Fatalf("structured content type = %T, want map", result.StructuredContent)
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
