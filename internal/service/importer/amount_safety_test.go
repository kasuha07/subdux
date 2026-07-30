package importer

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/money"
	"github.com/kasuha07/subdux/internal/service/servicetest"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
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
		wantErr      error
	}{
		{name: "normal price", price: "$15.99", wantAmount: 15.99, wantCurrency: "USD"},
		{name: "explicit UYI code", price: "UYI 1234", wantAmount: 1234, wantCurrency: "UYI"},
		{name: "negative price is rejected", price: "$-15.99", wantCurrency: "USD", wantErr: errImportedAmountNegative},
		{name: "unparseable price is rejected", price: "$abc", wantCurrency: "USD", wantErr: errImportedAmountInvalid},
		{name: "empty price is rejected", price: "", wantCurrency: "USD", wantErr: errImportedAmountInvalid},
		{name: "above the maximum price is rejected", price: "$99999999999999", wantCurrency: "USD", wantErr: errImportedAmountTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, currency, err := extractCurrencyAndAmount(tt.price, "USD")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("extractCurrencyAndAmount(%q) error = %v, want %v", tt.price, err, tt.wantErr)
			}
			if err == nil && amount != tt.wantAmount {
				t.Fatalf("extractCurrencyAndAmount(%q) amount = %v, want %v", tt.price, amount, tt.wantAmount)
			}
			if currency != tt.wantCurrency {
				t.Fatalf("extractCurrencyAndAmount(%q) currency = %q, want %q", tt.price, currency, tt.wantCurrency)
			}
		})
	}
}

func TestImportFromWallosReportsAndSkipsNegativeAmount(t *testing.T) {
	db := newImportTestDB(t)
	user := servicetest.CreateUser(t, db)
	svc := NewService(db)

	response, err := svc.ImportFromWallos(user.ID, []WallosSubscription{{
		Name:         "Refund Plan",
		Price:        "$-12.50",
		PaymentCycle: "Monthly",
		NextPayment:  "2026-03-01",
		Active:       "1",
	}}, true)
	if err != nil {
		t.Fatalf("ImportFromWallos() error = %v", err)
	}
	if response == nil || response.Result == nil {
		t.Fatal("ImportFromWallos() result = nil")
	}
	if response.Result.Skipped != 1 || len(response.Result.Errors) != 1 {
		t.Fatalf("ImportFromWallos() result = %+v, want one skipped item and one error", response.Result)
	}

	var subs []model.Subscription
	if err := db.Where("user_id = ?", user.ID).Find(&subs).Error; err != nil {
		t.Fatalf("load subscriptions failed: %v", err)
	}
	if len(subs) != 0 {
		t.Fatalf("subscription count = %d, want 0 (negative price must be skipped)", len(subs))
	}
}

func TestImportersPreviewInvalidAmountsConsistently(t *testing.T) {
	type previewResult struct {
		subscriptions []PreviewSubscriptionChange
	}

	importers := []struct {
		name string
		run  func(t *testing.T, svc *Service, userID uint) previewResult
	}{
		{
			name: "Subdux",
			run: func(t *testing.T, svc *Service, userID uint) previewResult {
				t.Helper()
				response, err := svc.ImportFromSubdux(userID, subduxImportDataWithSubscriptions(model.Subscription{
					Name:        "Broken Plan",
					Amount:      -12.5,
					Currency:    "USD",
					BillingType: "recurring",
				}), false)
				if err != nil {
					t.Fatalf("ImportFromSubdux() error = %v", err)
				}
				return previewResult{subscriptions: response.Preview.Subscriptions}
			},
		},
		{
			name: "Wallos",
			run: func(t *testing.T, svc *Service, userID uint) previewResult {
				t.Helper()
				response, err := svc.ImportFromWallos(userID, []WallosSubscription{{
					Name:         "Broken Plan",
					Price:        "$-12.50",
					PaymentCycle: "Monthly",
				}}, false)
				if err != nil {
					t.Fatalf("ImportFromWallos() error = %v", err)
				}
				return previewResult{subscriptions: response.Preview.Subscriptions}
			},
		},
	}

	for _, importer := range importers {
		t.Run(importer.name, func(t *testing.T) {
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			result := importer.run(t, NewService(db), user.ID)

			if len(result.subscriptions) != 1 {
				t.Fatalf("preview subscriptions = %#v, want one invalid subscription", result.subscriptions)
			}
			change := result.subscriptions[0]
			if !change.Skipped || change.SkipReason != "invalid_amount" {
				t.Fatalf("preview change = %+v, want skipped invalid_amount", change)
			}
			if change.Name != "Broken Plan" || change.Currency != "USD" || change.BillingType != "recurring" {
				t.Fatalf("preview change = %+v, want normalized name/currency/billing type", change)
			}
		})
	}
}

func TestImportersReportAndSkipInvalidAmountsConsistently(t *testing.T) {
	const wantError = `skipped subscription "Broken Plan" with invalid amount: amount must not be negative`

	importers := []struct {
		name string
		run  func(t *testing.T, svc *Service, userID uint) *ImportResult
	}{
		{
			name: "Subdux",
			run: func(t *testing.T, svc *Service, userID uint) *ImportResult {
				t.Helper()
				response, err := svc.ImportFromSubdux(userID, subduxImportDataWithSubscriptions(model.Subscription{
					Name:        "Broken Plan",
					Amount:      -12.5,
					Currency:    "USD",
					BillingType: "recurring",
				}), true)
				if err != nil {
					t.Fatalf("ImportFromSubdux() error = %v", err)
				}
				return response.Result
			},
		},
		{
			name: "Wallos",
			run: func(t *testing.T, svc *Service, userID uint) *ImportResult {
				t.Helper()
				response, err := svc.ImportFromWallos(userID, []WallosSubscription{{
					Name:         "Broken Plan",
					Price:        "$-12.50",
					PaymentCycle: "Monthly",
				}}, true)
				if err != nil {
					t.Fatalf("ImportFromWallos() error = %v", err)
				}
				return response.Result
			},
		},
	}

	for _, importer := range importers {
		t.Run(importer.name, func(t *testing.T) {
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			result := importer.run(t, NewService(db), user.ID)

			if result.Imported != 0 || result.Skipped != 1 {
				t.Fatalf("result imported/skipped = %d/%d, want 0/1", result.Imported, result.Skipped)
			}
			if len(result.Errors) != 1 || result.Errors[0] != wantError {
				t.Fatalf("result errors = %#v, want [%q]", result.Errors, wantError)
			}

			var count int64
			if err := db.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
				t.Fatalf("count subscriptions: %v", err)
			}
			if count != 0 {
				t.Fatalf("subscription count = %d, want 0", count)
			}
		})
	}
}

func TestImportersRejectInvalidDerivedMonthlyAmounts(t *testing.T) {
	nextBilling := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	t.Run("Subdux", func(t *testing.T) {
		run := func(t *testing.T, confirm bool) (*SubduxImportResponse, *gorm.DB, uint) {
			t.Helper()
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			response, err := NewService(db).ImportFromSubdux(user.ID, subduxImportDataWithSubscriptions(model.Subscription{
				Name:            "Daily CLF maximum plan",
				Amount:          money.MaxAmount,
				Currency:        "CLF",
				Enabled:         true,
				BillingType:     subscriptionservice.BillingTypeRecurring,
				RecurrenceType:  subscriptionservice.RecurrenceTypeInterval,
				IntervalCount:   ptrInt(1),
				IntervalUnit:    subscriptionservice.IntervalUnitDay,
				NextBillingDate: &nextBilling,
			}), confirm)
			if err != nil {
				t.Fatalf("ImportFromSubdux(confirm=%v) error = %v", confirm, err)
			}
			return response, db, user.ID
		}

		preview, _, _ := run(t, false)
		assertInvalidDerivedAmountPreview(t, preview.Preview.Subscriptions)

		confirmed, db, userID := run(t, true)
		assertInvalidDerivedAmountResult(t, confirmed.Result)
		assertNoImportedSubscriptions(t, db, userID)
	})

	t.Run("Wallos", func(t *testing.T) {
		run := func(t *testing.T, confirm bool) (*WallosImportResponse, *gorm.DB, uint) {
			t.Helper()
			db := newImportTestDB(t)
			user := servicetest.CreateUser(t, db)
			response, err := NewService(db).ImportFromWallos(user.ID, []WallosSubscription{{
				Name:         "Daily CLF maximum plan",
				Price:        "CLF 500000000000",
				PaymentCycle: "Daily",
				NextPayment:  "2026-03-01",
				Active:       "1",
			}}, confirm)
			if err != nil {
				t.Fatalf("ImportFromWallos(confirm=%v) error = %v", confirm, err)
			}
			return response, db, user.ID
		}

		preview, _, _ := run(t, false)
		assertInvalidDerivedAmountPreview(t, preview.Preview.Subscriptions)

		confirmed, db, userID := run(t, true)
		assertInvalidDerivedAmountResult(t, confirmed.Result)
		assertNoImportedSubscriptions(t, db, userID)
	})
}

func assertInvalidDerivedAmountPreview(t *testing.T, changes []PreviewSubscriptionChange) {
	t.Helper()
	if len(changes) != 1 {
		t.Fatalf("preview subscriptions = %#v, want one invalid subscription", changes)
	}
	if !changes[0].Skipped || changes[0].SkipReason != "invalid_amount" {
		t.Fatalf("preview change = %+v, want skipped invalid_amount", changes[0])
	}
}

func assertInvalidDerivedAmountResult(t *testing.T, result *ImportResult) {
	t.Helper()
	if result == nil || result.Imported != 0 || result.Skipped != 1 || len(result.Errors) != 1 {
		t.Fatalf("import result = %+v, want zero imported and one invalid_amount skip", result)
	}
}

func assertNoImportedSubscriptions(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var count int64
	if err := db.Model(&model.Subscription{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count subscriptions: %v", err)
	}
	if count != 0 {
		t.Fatalf("subscription count = %d, want 0", count)
	}
}

func TestNormalizeImportedAmount(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		want    float64
		wantErr error
	}{
		{name: "positive is kept", amount: 15.99, want: 15.99},
		{name: "zero is kept", amount: 0, want: 0},
		{name: "negative is rejected", amount: -1, wantErr: errImportedAmountNegative},
		{name: "nan is rejected", amount: math.NaN(), wantErr: errImportedAmountNonFinite},
		{name: "positive infinity is rejected as too large", amount: math.Inf(1), wantErr: errImportedAmountTooLarge},
		{name: "negative infinity is rejected as nonfinite", amount: math.Inf(-1), wantErr: errImportedAmountNonFinite},
		{name: "exactly the maximum is kept", amount: money.MaxAmount, want: money.MaxAmount},
		{name: "just above the maximum is rejected", amount: money.MaxAmount + 0.01, wantErr: errImportedAmountTooLarge},
		{name: "far above the maximum is rejected", amount: 1e13, wantErr: errImportedAmountTooLarge},
		{name: "wildly above the maximum is rejected", amount: 5e305, wantErr: errImportedAmountTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedAmount(tt.amount)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("normalizeImportedAmount(%v) error = %v, want %v", tt.amount, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
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
			response, err := svc.ImportFromSubdux(user.ID, data, true)
			if err != nil {
				t.Fatalf("ImportFromSubdux() error = %v", err)
			}
			if response == nil || response.Result == nil {
				t.Fatal("ImportFromSubdux() result = nil")
			}
			if response.Result.Skipped != 1 || len(response.Result.Errors) != 1 {
				t.Fatalf("ImportFromSubdux() result = %+v, want one skipped item and one error", response.Result)
			}

			var subs []model.Subscription
			if err := db.Where("user_id = ?", user.ID).Find(&subs).Error; err != nil {
				t.Fatalf("load subscriptions failed: %v", err)
			}
			if len(subs) != 0 {
				t.Fatalf("subscription count = %d, want 0 (unusable amount must be skipped)", len(subs))
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
