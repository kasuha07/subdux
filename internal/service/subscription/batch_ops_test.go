package subscription

import (
	"errors"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

func newBatchTestService(t *testing.T) (*Service, model.User, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	return NewService(db), createTestUser(t, db), db
}

func createBatchTestSubscription(t *testing.T, service *Service, userID uint, name string) model.Subscription {
	t.Helper()
	monthly := 1
	sub, err := service.Create(userID, CreateSubscriptionInput{
		Name:            name,
		Amount:          10,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-05-15",
	})
	if err != nil {
		t.Fatalf("create subscription %q failed: %v", name, err)
	}
	return *sub
}

func TestBatchDeleteRemovesSubscriptions(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	first := createBatchTestSubscription(t, service, user.ID, "First")
	second := createBatchTestSubscription(t, service, user.ID, "Second")
	third := createBatchTestSubscription(t, service, user.ID, "Third")

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionDelete,
		IDs:    []uint{first.ID, second.ID, third.ID},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Total != 3 || result.Succeeded != 3 || result.Failed != 0 {
		t.Fatalf("Batch() result = %+v, want 3 succeeded", result)
	}

	var count int64
	if err := service.DB.Model(&model.Subscription{}).Where("user_id = ?", user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("remaining subscriptions = %d, want 0", count)
	}
}

func TestBatchDeleteReportsNotFoundPerItem(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	existing := createBatchTestSubscription(t, service, user.ID, "Existing")

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionDelete,
		IDs:    []uint{existing.ID, 9999},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Total != 2 || result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("Batch() result = %+v, want 1 succeeded / 1 failed", result)
	}
	if len(result.Failures) != 1 || result.Failures[0].ID != 9999 || result.Failures[0].Code != "subscription_not_found" {
		t.Fatalf("Batch() failures = %+v, want id 9999 with subscription_not_found", result.Failures)
	}
}

func TestBatchDeleteScopedToUser(t *testing.T) {
	service, user, db := newBatchTestService(t)
	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}
	owned := createBatchTestSubscription(t, service, user.ID, "Mine")
	foreign := createBatchTestSubscription(t, service, other.ID, "Theirs")

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionDelete,
		IDs:    []uint{owned.ID, foreign.ID},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("Batch() result = %+v, want the foreign row rejected", result)
	}
	if result.Failures[0].ID != foreign.ID {
		t.Fatalf("Batch() failure = %+v, want the foreign subscription id", result.Failures[0])
	}

	var remaining model.Subscription
	if err := db.Where("id = ?", foreign.ID).First(&remaining).Error; err != nil {
		t.Fatalf("foreign subscription should be untouched: %v", err)
	}
}

func TestBatchUpdateStatusEndsSubscriptions(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	first := createBatchTestSubscription(t, service, user.ID, "First")
	second := createBatchTestSubscription(t, service, user.ID, "Second")

	status := subscriptionStatusEnded
	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionUpdate,
		IDs:    []uint{first.ID, second.ID},
		Status: &status,
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Succeeded != 2 || result.Failed != 0 {
		t.Fatalf("Batch() result = %+v, want 2 succeeded", result)
	}

	for _, id := range []uint{first.ID, second.ID} {
		sub, err := service.GetByID(user.ID, id)
		if err != nil {
			t.Fatalf("GetByID(%d) error = %v", id, err)
		}
		if normalizeStatus(sub.Status) != subscriptionStatusEnded {
			t.Fatalf("subscription %d status = %s, want ended", id, sub.Status)
		}
		if sub.EndsAt == nil {
			t.Fatalf("subscription %d should have ends_at set when ended", id)
		}
	}
}

func TestBatchUpdateStatusReactivatesSubscriptions(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	sub := createBatchTestSubscription(t, service, user.ID, "Sub")

	ended := subscriptionStatusEnded
	if _, err := service.Batch(user.ID, BatchSubscriptionInput{Action: BatchActionUpdate, IDs: []uint{sub.ID}, Status: &ended}); err != nil {
		t.Fatalf("end failed: %v", err)
	}

	active := subscriptionStatusActive
	autoRenew := renewalModeAutoRenew
	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action:      BatchActionUpdate,
		IDs:         []uint{sub.ID},
		Status:      &active,
		RenewalMode: &autoRenew,
	})
	if err != nil {
		t.Fatalf("Batch() reactivate error = %v", err)
	}
	if result.Succeeded != 1 {
		t.Fatalf("Batch() result = %+v, want 1 succeeded", result)
	}

	refreshed, err := service.GetByID(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("GetByID error = %v", err)
	}
	if normalizeStatus(refreshed.Status) != subscriptionStatusActive {
		t.Fatalf("status = %s, want active", refreshed.Status)
	}
	if normalizeRenewalMode(refreshed.RenewalMode) != renewalModeAutoRenew {
		t.Fatalf("renewal_mode = %s, want auto_renew", refreshed.RenewalMode)
	}
	if refreshed.EndsAt != nil {
		t.Fatalf("ends_at = %v, want nil after reactivation", refreshed.EndsAt)
	}
}

func TestBatchUpdateCategorySetsAndClears(t *testing.T) {
	service, user, db := newBatchTestService(t)
	first := createBatchTestSubscription(t, service, user.ID, "First")
	second := createBatchTestSubscription(t, service, user.ID, "Second")

	category := model.Category{UserID: user.ID, Name: "Video"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action:     BatchActionUpdate,
		IDs:        []uint{first.ID, second.ID},
		CategoryID: &category.ID,
	})
	if err != nil {
		t.Fatalf("Batch() set category error = %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("Batch() result = %+v, want 2 succeeded", result)
	}
	for _, id := range []uint{first.ID, second.ID} {
		sub, err := service.GetByID(user.ID, id)
		if err != nil {
			t.Fatalf("GetByID(%d) error = %v", id, err)
		}
		if sub.CategoryID == nil || *sub.CategoryID != category.ID {
			t.Fatalf("subscription %d category_id = %v, want %d", id, sub.CategoryID, category.ID)
		}
	}

	// Clearing category: send category_id null.
	clearResult, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action:        BatchActionUpdate,
		IDs:           []uint{first.ID},
		CategoryID:    nil,
		CategoryIDSet: true,
	})
	if err != nil {
		t.Fatalf("Batch() clear category error = %v", err)
	}
	if clearResult.Succeeded != 1 {
		t.Fatalf("clear result = %+v, want 1 succeeded", clearResult)
	}
	cleared, err := service.GetByID(user.ID, first.ID)
	if err != nil {
		t.Fatalf("GetByID error = %v", err)
	}
	if cleared.CategoryID != nil {
		t.Fatalf("category_id = %v, want nil after clearing", cleared.CategoryID)
	}
}

func TestBatchUpdatePaymentMethod(t *testing.T) {
	service, user, db := newBatchTestService(t)
	first := createBatchTestSubscription(t, service, user.ID, "First")
	second := createBatchTestSubscription(t, service, user.ID, "Second")

	method := model.PaymentMethod{UserID: user.ID, Name: "Card"}
	if err := db.Create(&method).Error; err != nil {
		t.Fatalf("create payment method failed: %v", err)
	}

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action:          BatchActionUpdate,
		IDs:             []uint{first.ID, second.ID},
		PaymentMethodID: &method.ID,
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Succeeded != 2 {
		t.Fatalf("Batch() result = %+v, want 2 succeeded", result)
	}
	for _, id := range []uint{first.ID, second.ID} {
		sub, err := service.GetByID(user.ID, id)
		if err != nil {
			t.Fatalf("GetByID(%d) error = %v", id, err)
		}
		if sub.PaymentMethodID == nil || *sub.PaymentMethodID != method.ID {
			t.Fatalf("subscription %d payment_method_id = %v, want %d", id, sub.PaymentMethodID, method.ID)
		}
	}
}

func TestBatchMarkRenewedAdvancesManualRenewSubscriptions(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-04-01"))
	t.Cleanup(restoreClock)

	service, user, _ := newBatchTestService(t)
	monthly := 1
	first, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual",
		Amount:          10,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-04-15",
	})
	if err != nil {
		t.Fatalf("create manual subscription failed: %v", err)
	}

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionMarkRenewed,
		IDs:    []uint{first.ID},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Succeeded != 1 || result.Failed != 0 {
		t.Fatalf("Batch() result = %+v, want 1 succeeded", result)
	}

	sub, err := service.GetByID(user.ID, first.ID)
	if err != nil {
		t.Fatalf("GetByID error = %v", err)
	}
	if sub.NextBillingDate == nil || sub.NextBillingDate.Format("2006-01-02") != "2026-05-15" {
		t.Fatalf("next_billing_date = %v, want 2026-05-15", sub.NextBillingDate)
	}
}

func TestBatchMarkRenewedReportsPerItemFailures(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-04-01"))
	t.Cleanup(restoreClock)

	service, user, _ := newBatchTestService(t)
	autoRenew := createBatchTestSubscription(t, service, user.ID, "Auto")
	manualRenewSub := createBatchTestSubscription(t, service, user.ID, "Manual")
	manual := renewalModeManualRenew
	if _, err := service.Update(user.ID, manualRenewSub.ID, UpdateSubscriptionInput{RenewalMode: &manual}); err != nil {
		t.Fatalf("switch to manual renew failed: %v", err)
	}

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionMarkRenewed,
		IDs:    []uint{autoRenew.ID, manualRenewSub.ID},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Succeeded != 1 || result.Failed != 1 {
		t.Fatalf("Batch() result = %+v, want 1 succeeded / 1 failed", result)
	}
	if result.Failures[0].ID != autoRenew.ID || result.Failures[0].Code != "only_manual_renew_subscriptions_can_be_marked_as_renewed" {
		t.Fatalf("Batch() failure = %+v, want auto-renew id rejected", result.Failures[0])
	}
}

func TestBatchDeduplicatesIDs(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	sub := createBatchTestSubscription(t, service, user.ID, "Sub")

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionDelete,
		IDs:    []uint{sub.ID, sub.ID, sub.ID},
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Total != 1 || result.Succeeded != 1 {
		t.Fatalf("Batch() result = %+v, want deduplicated to 1", result)
	}
}

func TestBatchAllowsDuplicateIDsBeyondUniqueLimit(t *testing.T) {
	service, user, _ := newBatchTestService(t)
	sub := createBatchTestSubscription(t, service, user.ID, "Sub")
	ids := make([]uint, MaxBatchSubscriptionIDs+1)
	for i := range ids {
		ids[i] = sub.ID
	}

	result, err := service.Batch(user.ID, BatchSubscriptionInput{
		Action: BatchActionDelete,
		IDs:    ids,
	})
	if err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if result.Total != 1 || result.Succeeded != 1 || result.Failed != 0 {
		t.Fatalf("Batch() result = %+v, want one deduplicated success", result)
	}
}

func TestBatchValidationErrors(t *testing.T) {
	service, user, db := newBatchTestService(t)
	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("create other user failed: %v", err)
	}
	foreignCategory := model.Category{UserID: other.ID, Name: "Foreign"}
	if err := db.Create(&foreignCategory).Error; err != nil {
		t.Fatalf("create foreign category failed: %v", err)
	}
	tooManyIDs := make([]uint, MaxBatchSubscriptionIDs+1)
	for i := range tooManyIDs {
		tooManyIDs[i] = uint(i + 1)
	}

	tests := []struct {
		name  string
		input BatchSubscriptionInput
		want  string
	}{
		{
			name:  "empty ids",
			input: BatchSubscriptionInput{Action: BatchActionDelete, IDs: nil},
			want:  "batch_ids_required",
		},
		{
			name:  "too many ids",
			input: BatchSubscriptionInput{Action: BatchActionDelete, IDs: tooManyIDs},
			want:  "batch_too_many_ids",
		},
		{
			name:  "invalid action",
			input: BatchSubscriptionInput{Action: "explode", IDs: []uint{1}},
			want:  "invalid_batch_action",
		},
		{
			name:  "update without fields",
			input: BatchSubscriptionInput{Action: BatchActionUpdate, IDs: []uint{1}},
			want:  "batch_update_requires_at_least_one_field",
		},
		{
			name:  "invalid status",
			input: BatchSubscriptionInput{Action: BatchActionUpdate, IDs: []uint{1}, Status: stringPtr("archived")},
			want:  "status_must_be_one_of_active_ended",
		},
		{
			name:  "invalid renewal mode",
			input: BatchSubscriptionInput{Action: BatchActionUpdate, IDs: []uint{1}, RenewalMode: stringPtr("monthly")},
			want:  "renewal_mode_must_be_one_of_auto_renew_manual_renew_cancel_at_period_end",
		},
		{
			name:  "category owned by another user",
			input: BatchSubscriptionInput{Action: BatchActionUpdate, IDs: []uint{1}, CategoryID: &foreignCategory.ID},
			want:  "category_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Batch(user.ID, tt.input)
			if err == nil {
				t.Fatalf("Batch() error = nil, want code %q", tt.want)
			}
			var typed *serviceerr.Error
			if !errors.As(err, &typed) || typed == nil {
				t.Fatalf("Batch() error = %v, want a typed service error with code %q", err, tt.want)
			}
			if typed.Code != tt.want {
				t.Fatalf("Batch() error code = %q, want %q", typed.Code, tt.want)
			}
		})
	}
}

func stringPtr(value string) *string {
	return &value
}

func TestBatchFailureForUntypedErrorIsSanitized(t *testing.T) {
	f := batchFailureFor(42, errors.New("sqlite: disk I/O error (code 778): database disk image is malformed"))
	if f.Code != ErrBatchInternal.Code {
		t.Fatalf("failure code = %q, want %q", f.Code, ErrBatchInternal.Code)
	}
	if f.Message == "" {
		t.Fatal("failure message must not be empty")
	}
	if strings.Contains(f.Message, "sqlite") || strings.Contains(f.Message, "disk I/O") {
		t.Fatalf("failure message leaks internal error detail: %q", f.Message)
	}
}

func TestBatchFailureForKeepsTypedErrorCode(t *testing.T) {
	typed := serviceerr.New(serviceerr.KindInvalid, "category_not_found", "category not found")
	f := batchFailureFor(7, typed)
	if f.Code != "category_not_found" || f.Message != "category not found" {
		t.Fatalf("failure = %+v, want category_not_found surfaced verbatim", f)
	}
}
