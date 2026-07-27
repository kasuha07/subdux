package importer

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/servicetest"
)

// subduxImportDataWithSubscriptions builds the minimal valid Subdux payload:
// validateSubduxImportData rejects nil collections.
func subduxImportDataWithSubscriptions(subs ...model.Subscription) SubduxImportData {
	return SubduxImportData{
		Currencies:     []model.UserCurrency{},
		Categories:     []model.Category{},
		PaymentMethods: []model.PaymentMethod{},
		Subscriptions:  subs,
	}
}

func TestExtractCurrencyAndAmountRejectsUnusableAmounts(t *testing.T) {
	tests := []struct {
		name         string
		price        string
		wantAmount   float64
		wantCurrency string
	}{
		{name: "normal price", price: "$15.99", wantAmount: 15.99, wantCurrency: "USD"},
		{name: "negative price collapses to zero", price: "$-15.99", wantAmount: 0, wantCurrency: "USD"},
		{name: "unparseable price collapses to zero", price: "$abc", wantAmount: 0, wantCurrency: "USD"},
		{name: "empty price collapses to zero", price: "", wantAmount: 0, wantCurrency: "USD"},
		{name: "above the maximum price collapses to zero", price: "$99999999999999", wantAmount: 0, wantCurrency: "USD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency := extractCurrencyAndAmount(tt.price, "USD")
			if amount != tt.wantAmount {
				t.Fatalf("extractCurrencyAndAmount(%q) amount = %v, want %v", tt.price, amount, tt.wantAmount)
			}
			if currency != tt.wantCurrency {
				t.Fatalf("extractCurrencyAndAmount(%q) currency = %q, want %q", tt.price, currency, tt.wantCurrency)
			}
		})
	}
}

func TestImportFromWallosNeverWritesNegativeAmount(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)

	if _, err := svc.ImportFromWallos(user.ID, []WallosSubscription{{
		Name:         "Refund Plan",
		Price:        "$-12.50",
		PaymentCycle: "Monthly",
		NextPayment:  "2026-03-01",
		Active:       "1",
	}}, true); err != nil {
		t.Fatalf("ImportFromWallos() error = %v", err)
	}

	var subs []model.Subscription
	if err := db.Where("user_id = ?", user.ID).Find(&subs).Error; err != nil {
		t.Fatalf("load subscriptions failed: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("subscription count = %d, want 1", len(subs))
	}
	if subs[0].Amount != 0 {
		t.Fatalf("amount = %v, want 0 (negative price must not reach the DB)", subs[0].Amount)
	}
}

func TestNormalizeImportedAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount float64
		want   float64
	}{
		{name: "positive is kept", amount: 15.99, want: 15.99},
		{name: "zero is kept", amount: 0, want: 0},
		{name: "negative collapses to zero", amount: -1, want: 0},
		{name: "nan collapses to zero", amount: math.NaN(), want: 0},
		{name: "positive infinity collapses to zero", amount: math.Inf(1), want: 0},
		{name: "negative infinity collapses to zero", amount: math.Inf(-1), want: 0},
		{name: "exactly the maximum is kept", amount: 1000000000000, want: 1000000000000},
		{name: "just above the maximum collapses to zero", amount: 1000000000000.01, want: 0},
		{name: "far above the maximum collapses to zero", amount: 1e13, want: 0},
		{name: "wildly above the maximum collapses to zero", amount: 5e305, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeImportedAmount(tt.amount); got != tt.want {
				t.Fatalf("normalizeImportedAmount(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestImportFromSubduxNeverWritesUnusableAmount(t *testing.T) {
	nextBilling := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		amount float64
	}{
		{name: "negative", amount: -20},
		{name: "nan", amount: math.NaN()},
		{name: "infinity", amount: math.Inf(1)},
		{name: "above the maximum", amount: 1e13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			svc := NewService(db)

			data := subduxImportDataWithSubscriptions(model.Subscription{
				Name:            "Broken Plan",
				Amount:          tt.amount,
				Currency:        "USD",
				Enabled:         true,
				BillingType:     "recurring",
				RecurrenceType:  "interval",
				IntervalCount:   ptrInt(1),
				IntervalUnit:    "month",
				NextBillingDate: &nextBilling,
			})
			if _, err := svc.ImportFromSubdux(user.ID, data, true); err != nil {
				t.Fatalf("ImportFromSubdux() error = %v", err)
			}

			var subs []model.Subscription
			if err := db.Where("user_id = ?", user.ID).Find(&subs).Error; err != nil {
				t.Fatalf("load subscriptions failed: %v", err)
			}
			if len(subs) != 1 {
				t.Fatalf("subscription count = %d, want 1", len(subs))
			}
			if subs[0].Amount != 0 {
				t.Fatalf("amount = %v, want 0 (unusable amount must not reach the DB)", subs[0].Amount)
			}
		})
	}
}

// TestImportAmountKeyIsExactAndShared pins the shared dedup rendering both
// importers now use. The Subdux importer previously used %f, which truncated at
// six decimals and collapsed distinct amounts; the Wallos importer used %v,
// which switched to exponent notation, so the two disagreed on the same value.
func TestImportAmountKeyIsExactAndShared(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{amount: 15.99, want: "15.99"},
		{amount: 0.0000001, want: "0.0000001"},
		{amount: 0.0000002, want: "0.0000002"},
		{amount: 1234567.891234567, want: "1234567.891234567"},
		{amount: 0, want: "0"},
	}

	for _, tt := range tests {
		if got := importAmountKey(tt.amount); got != tt.want {
			t.Fatalf("importAmountKey(%v) = %q, want %q", tt.amount, got, tt.want)
		}
	}

	if importAmountKey(0.0000001) == importAmountKey(0.0000002) {
		t.Fatal("importAmountKey collapsed two distinct amounts into one key")
	}
	if got, want := fmt.Sprintf("%v", 1e21), importAmountKey(1e21); got == want {
		t.Fatalf("expected %%v (%q) to differ from the shared dedup key (%q)", got, want)
	}
}

// TestImportFromSubduxKeepsDistinctSubMicroAmounts proves the dedup key no
// longer truncates: two subscriptions differing below six decimals are distinct
// records, not a duplicate pair.
func TestImportFromSubduxKeepsDistinctSubMicroAmounts(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)

	nextBilling := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	newSub := func(amount float64) model.Subscription {
		return model.Subscription{
			Name:            "Micro Plan",
			Amount:          amount,
			Currency:        "USD",
			Enabled:         true,
			BillingType:     "recurring",
			RecurrenceType:  "interval",
			IntervalCount:   ptrInt(1),
			IntervalUnit:    "month",
			NextBillingDate: &nextBilling,
		}
	}

	data := subduxImportDataWithSubscriptions(newSub(0.0000001), newSub(0.0000002))
	if _, err := svc.ImportFromSubdux(user.ID, data, true); err != nil {
		t.Fatalf("ImportFromSubdux() error = %v", err)
	}

	var subs []model.Subscription
	if err := db.Where("user_id = ?", user.ID).Find(&subs).Error; err != nil {
		t.Fatalf("load subscriptions failed: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("subscription count = %d, want 2 (distinct amounts are not duplicates)", len(subs))
	}
}
