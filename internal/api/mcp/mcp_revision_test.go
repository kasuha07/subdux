package mcp

import (
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestMCPRevisionRequiredAndConflictRollsBackIdempotency(t *testing.T) {
	db := newMCPTestDB(t)
	user := createMCPTestUser(t, db)
	apiKey := createMCPAPIKey(t, db, user, nil)
	handler := newMCPTestHandler(db)
	for _, tool := range []string{"update_subscription", "delete_subscription", "mark_subscription_renewed"} {
		_, response := performMCPToolCall(t, handler, apiKey, tool, map[string]interface{}{"id": 1, "idempotency_key": "missing-" + tool})
		if response["error"] == nil {
			t.Fatalf("%s accepted missing revision: %#v", tool, response)
		}
	}
	sub, err := subscriptionservice.NewService(db).Create(user.ID, subscriptionservice.CreateSubscriptionInput{Name: "original", Amount: 10, Currency: "USD", RecurrenceType: "interval", IntervalCount: intPtr(1), IntervalUnit: "month", NextBillingDate: "2099-01-01"})
	if err != nil {
		t.Fatal(err)
	}
	args := map[string]interface{}{"id": sub.ID, "revision": sub.Revision, "idempotency_key": "revision-winner", "name": "winner"}
	rec, response := performMCPToolCall(t, handler, apiKey, "update_subscription", args)
	assertMCPToolSuccess(t, rec, response)
	// Replaying a successful write uses the stored result despite its old revision.
	rec, response = performMCPToolCall(t, handler, apiKey, "update_subscription", args)
	assertMCPToolSuccess(t, rec, response)
	rec, response = performMCPToolCall(t, handler, apiKey, "update_subscription", map[string]interface{}{"id": sub.ID, "revision": sub.Revision, "idempotency_key": "revision-stale", "name": "stale"})
	assertMCPToolErrorCode(t, rec, response, "revision_conflict")
	assertNoMCPIdempotencyRecord(t, db, user.ID, "revision-stale")
	var saved model.Subscription
	if err := db.First(&saved, sub.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Name != "winner" || saved.Revision != sub.Revision+1 {
		t.Fatalf("saved=%+v", saved)
	}
}
