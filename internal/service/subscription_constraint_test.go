package service

import (
	"strings"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

func TestCreateSubscriptionRejectsCategoryOwnedByAnotherUser(t *testing.T) {
	db := newTestDB(t)
	owner := createTestUser(t, db)
	other := model.User{Username: "other", Email: "other@example.com", Password: "hashed-password", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}

	category := model.Category{UserID: other.ID, Name: "Other Category"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("failed to create cross-user category: %v", err)
	}

	nextBillingDate := time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02")
	svc := NewSubscriptionService(db)
	_, err := svc.Create(owner.ID, CreateSubscriptionInput{
		Name:            "Netflix",
		Amount:          12.99,
		Currency:        "USD",
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   func() *int { v := 1; return &v }(),
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: nextBillingDate,
		CategoryID:      &category.ID,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want category validation error")
	}
	if !strings.Contains(err.Error(), "category not found") {
		t.Fatalf("Create() error = %v, want category not found", err)
	}
}
