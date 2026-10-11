package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
)

func TestUpdatePendingPriceSetKeepAndClear(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	reload := func() model.Subscription {
		t.Helper()
		var reloaded model.Subscription
		if err := handler.Service.DB.First(&reloaded, sub.ID).Error; err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		return reloaded
	}

	rec := postUpdateJSON(t, handler, userID, sub.ID, `{"pending_amount":30,"pending_from":"2099-01-15"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("set status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"pending_amount":30`) || !strings.Contains(rec.Body.String(), `"pending_from":"2099-01-15"`) {
		t.Fatalf("response does not expose the scheduled price: %s", rec.Body.String())
	}

	// Omitting both keys leaves the schedule untouched.
	if rec := postUpdateJSON(t, handler, userID, sub.ID, `{"name":"Renamed"}`); rec.Code != http.StatusOK {
		t.Fatalf("rename status = %d (body %s)", rec.Code, rec.Body.String())
	}
	if reloaded := reload(); reloaded.PendingAmount == nil || reloaded.PendingFrom == nil {
		t.Fatal("unrelated update cleared the scheduled price")
	}

	// Explicit nulls clear the pair.
	if rec := postUpdateJSON(t, handler, userID, sub.ID, `{"pending_amount":null,"pending_from":null}`); rec.Code != http.StatusOK {
		t.Fatalf("clear status = %d (body %s)", rec.Code, rec.Body.String())
	}
	if reloaded := reload(); reloaded.PendingAmount != nil || reloaded.PendingFrom != nil {
		t.Fatalf("scheduled price = %v/%v, want cleared", reloaded.PendingAmount, reloaded.PendingFrom)
	}
}

func TestUpdatePendingPriceRejectsInvalidInput(t *testing.T) {
	handler, userID := newBatchHandler(t)
	sub := seedBatchHandlerSubscription(t, handler, userID, "Sub")

	for name, tc := range map[string]struct {
		body string
		code string
	}{
		"amount only":     {`{"pending_amount":30}`, "pending_amount_and_pending_from_must_be_set_together"},
		"negative amount": {`{"pending_amount":-1,"pending_from":"2099-01-15"}`, "amount_must_not_be_negative"},
		"past date":       {`{"pending_amount":30,"pending_from":"2000-01-01"}`, "pending_from_must_be_after_today"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := postUpdateJSON(t, handler, userID, sub.ID, tc.body)
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), tc.code) {
				t.Fatalf("status = %d body = %s, want 400 %s", rec.Code, rec.Body.String(), tc.code)
			}
		})
	}
}
