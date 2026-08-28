package mcp

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
	"gorm.io/gorm"
)

func TestMCPDashboardAggregateOverflowReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)
	createMCPUserCurrency(t, db, user.ID, "CLF")

	nextBillingDate := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	legacyAggregate := model.Subscription{
		UserID:          user.ID,
		Name:            "Legacy aggregate overflow",
		Amount:          money.MaxAmount,
		Currency:        "CLF",
		Enabled:         true,
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   intPtr(1),
		IntervalUnit:    "month",
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&legacyAggregate).Error; err != nil {
		t.Fatalf("seed legacy aggregate subscription: %v", err)
	}

	rec, response := performMCPToolCall(t, handler, apiKey, "get_dashboard_summary", map[string]interface{}{
		"currency": "CLF",
	})
	assertMCPToolErrorCode(t, rec, response, "amount_too_large")
}

func TestMCPDashboardLegacyOverflowReturnsTypedServiceErrorCode(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)
	createMCPUserCurrency(t, db, user.ID, "CLF")

	daily := 1
	nextBillingDate := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	legacyOverflow := model.Subscription{
		UserID:          user.ID,
		Name:            "Legacy daily overflow",
		Amount:          900_000_000_000,
		Currency:        "CLF",
		Enabled:         true,
		Status:          "active",
		RenewalMode:     "auto_renew",
		BillingType:     "recurring",
		RecurrenceType:  "interval",
		IntervalCount:   &daily,
		IntervalUnit:    "day",
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&legacyOverflow).Error; err != nil {
		t.Fatalf("seed legacy overflow subscription: %v", err)
	}

	rec, response := performMCPToolCall(t, handler, apiKey, "get_dashboard_summary", map[string]interface{}{
		"currency": "CLF",
	})
	assertMCPToolErrorCode(t, rec, response, "amount_too_large")
}

func createMCPUserCurrency(t *testing.T, db *gorm.DB, userID uint, code string) {
	t.Helper()
	if err := db.Create(&model.UserCurrency{UserID: userID, Code: code}).Error; err != nil {
		t.Fatalf("create user currency %s: %v", code, err)
	}
}
