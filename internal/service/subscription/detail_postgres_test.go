package subscription

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
)

func TestPostgresPriceHistoryUsesJSONBFields(t *testing.T) {
	db := openSubscriptionPostgresTestDB(t)
	var indexCount int64
	if err := db.Raw(`SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = 'idx_subscription_events_changed_fields_jsonb'`).Scan(&indexCount).Error; err != nil {
		t.Fatalf("inspect JSONB index: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("JSONB expression index count = %d, want 1", indexCount)
	}

	user := model.User{Username: "jsonb-" + uuid.NewString(), Email: uuid.NewString() + "@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	sub := model.Subscription{UserID: user.ID, Name: "JSONB history", Amount: 10, Currency: "USD", Status: "active", RenewalMode: "auto_renew", BillingType: "recurring"}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	initial, changed, unrelated := 10.0, 12.0, 99.0
	events := []model.SubscriptionEvent{
		{UserID: user.ID, SubscriptionID: &sub.ID, SubscriptionName: sub.Name, Type: subscriptionEventCreated, ChangedFields: `["created"]`, NewAmount: &initial, NewCurrency: "USD", CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{UserID: user.ID, SubscriptionID: &sub.ID, SubscriptionName: sub.Name, Type: subscriptionEventUpdated, ChangedFields: `["notes"]`, NewAmount: &unrelated, NewCurrency: "USD", CreatedAt: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
		{UserID: user.ID, SubscriptionID: &sub.ID, SubscriptionName: sub.Name, Type: subscriptionEventUpdated, ChangedFields: `["amount","notes"]`, NewAmount: &changed, NewCurrency: "USD", CreatedAt: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatal(err)
	}
	history, err := NewService(db).subscriptionDetailPriceHistory(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("subscriptionDetailPriceHistory: %v", err)
	}
	if len(history) != 2 || history[0].Amount != initial || history[1].Amount != changed {
		t.Fatalf("price history = %+v, want created and amount-change events", history)
	}
}
