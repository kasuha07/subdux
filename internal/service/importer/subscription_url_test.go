package importer

import (
	"strings"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
)

func TestImportFromSubduxSkipsUnsafeSubscriptionURL(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)
	data := sampleSubduxImportData()
	data.Subscriptions[0].URL = "javascript:alert(1)"

	resp, err := svc.ImportFromSubdux(user.ID, data, true)
	if err != nil {
		t.Fatalf("ImportFromSubdux() error = %v", err)
	}
	if resp.Result == nil {
		t.Fatal("ImportFromSubdux() result = nil")
	}
	if resp.Result.Skipped != 1 {
		t.Fatalf("ImportFromSubdux() skipped = %d, want 1", resp.Result.Skipped)
	}
	if len(resp.Result.Errors) != 1 || !strings.Contains(resp.Result.Errors[0], "invalid url") {
		t.Fatalf("ImportFromSubdux() errors = %#v, want invalid url error", resp.Result.Errors)
	}
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count subscriptions: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0", count)
	}
}

func TestImportFromWallosSkipsUnsafeSubscriptionURL(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)

	resp, err := svc.ImportFromWallos(user.ID, []WallosSubscription{{
		Name:         "Netflix",
		Price:        "12.99 USD",
		PaymentCycle: "Monthly",
		NextPayment:  time.Now().UTC().Add(24 * time.Hour).Format("2006-01-02"),
		Active:       "1",
		URL:          "data:text/html,<script>alert(1)</script>",
	}}, true)
	if err != nil {
		t.Fatalf("ImportFromWallos() error = %v", err)
	}
	if resp.Result == nil {
		t.Fatal("ImportFromWallos() result = nil")
	}
	if resp.Result.Imported != 0 || resp.Result.Skipped != 1 {
		t.Fatalf("ImportFromWallos() imported/skipped = %d/%d, want 0/1", resp.Result.Imported, resp.Result.Skipped)
	}
	if len(resp.Result.Errors) != 1 || !strings.Contains(resp.Result.Errors[0], "invalid url") {
		t.Fatalf("ImportFromWallos() errors = %#v, want invalid url error", resp.Result.Errors)
	}
}
