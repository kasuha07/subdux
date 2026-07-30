package subscription

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/money"
	"gorm.io/gorm"
)

func TestLegacyAmountAboveCurrentMaximumRemainsUsable(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	monthly := 1
	nextBillingDate := mustDate(t, "2026-03-15")
	const legacyAmount = 700_000_000_000.0

	legacy := model.Subscription{
		UserID:          user.ID,
		Name:            "Legacy large plan",
		Amount:          legacyAmount,
		Currency:        "USD",
		Enabled:         true,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy subscription failed: %v", err)
	}

	subs, err := service.List(user.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(subs) != 1 || subs[0].Amount != legacyAmount {
		t.Fatalf("List() subscriptions = %+v, want unchanged legacy amount %v", subs, legacyAmount)
	}

	got, err := service.GetByID(user.ID, legacy.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Amount != legacyAmount {
		t.Fatalf("GetByID() amount = %v, want unchanged %v", got.Amount, legacyAmount)
	}

	summary, err := service.GetDashboardSummary(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}
	if summary.TotalMonthly != legacyAmount {
		t.Fatalf("dashboard total_monthly = %v, want %v", summary.TotalMonthly, legacyAmount)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}
	if report.KPIs.TotalMonthly != legacyAmount {
		t.Fatalf("report total_monthly = %v, want %v", report.KPIs.TotalMonthly, legacyAmount)
	}

	renamed := "Renamed legacy large plan"
	unchangedLegacyAmount := legacyAmount
	updated, err := service.Update(user.ID, legacy.ID, UpdateSubscriptionInput{
		Name:   &renamed,
		Amount: &unchangedLegacyAmount,
	})
	if err != nil {
		t.Fatalf("Update(name only) error = %v", err)
	}
	if updated.Amount != legacyAmount {
		t.Fatalf("Update(name only) amount = %v, want unchanged %v", updated.Amount, legacyAmount)
	}

	tooLarge := float64(money.MaxAmount + 1)
	if _, err := service.Update(user.ID, legacy.ID, UpdateSubscriptionInput{Amount: &tooLarge}); !errors.Is(err, ErrAmountTooLarge) {
		t.Fatalf("Update(actual oversized amount) error = %v, want %v", err, ErrAmountTooLarge)
	}

	if err := service.Delete(user.ID, legacy.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	var deleted model.Subscription
	if err := db.First(&deleted, legacy.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("load deleted subscription error = %v, want record not found", err)
	}
}

func TestDailyAmountMayDeriveAbovePersistedMaximumWhenMinorUnitsAreSafe(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	daily := 1
	const amount = 20_000_000_000.0
	const wantMonthly = 608_737_500_000.0
	const wantYearly = 7_304_850_000_000.0

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Large daily plan",
		Amount:          amount,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &daily,
		IntervalUnit:    intervalUnitDay,
		NextBillingDate: "2026-03-15",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sub.Amount != amount {
		t.Fatalf("Create() amount = %v, want unchanged %v", sub.Amount, amount)
	}

	summary, err := service.GetDashboardSummary(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}
	if summary.TotalMonthly != wantMonthly {
		t.Fatalf("dashboard total_monthly = %v, want %v", summary.TotalMonthly, wantMonthly)
	}
	if summary.TotalYearly != wantYearly {
		t.Fatalf("dashboard total_yearly = %v, want %v", summary.TotalYearly, wantYearly)
	}

	report, err := service.GetAnalyticsReport(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetAnalyticsReport() error = %v", err)
	}
	if report.KPIs.TotalMonthly != wantMonthly {
		t.Fatalf("report total_monthly = %v, want %v", report.KPIs.TotalMonthly, wantMonthly)
	}
	if report.KPIs.TotalYearly != wantYearly {
		t.Fatalf("report total_yearly = %v, want %v", report.KPIs.TotalYearly, wantYearly)
	}
}

func TestLegacyMinorUnitUnsafeDerivedAmountCanStillBeUpdatedAndDeleted(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	user := createTestUser(t, db)
	service := NewService(db)
	daily := 1
	nextBillingDate := mustDate(t, "2026-03-15")
	const legacyAmount = 900_000_000_000.0
	const rawMonthlyAmount = 27_393_187_500_000.0

	legacy := model.Subscription{
		UserID:          user.ID,
		Name:            "Legacy unsafe-derived plan",
		Amount:          legacyAmount,
		Currency:        "CLF",
		Enabled:         true,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &daily,
		IntervalUnit:    intervalUnitDay,
		NextBillingDate: &nextBillingDate,
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatalf("seed legacy subscription failed: %v", err)
	}

	renamed := "Renamed legacy unsafe-derived plan"
	updated, err := service.Update(user.ID, legacy.ID, UpdateSubscriptionInput{Name: &renamed})
	if err != nil {
		t.Fatalf("Update(name only) error = %v", err)
	}
	if updated.Amount != legacyAmount {
		t.Fatalf("Update(name only) amount = %v, want unchanged %v", updated.Amount, legacyAmount)
	}

	if err := service.Delete(user.ID, legacy.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	var deletedEvent model.SubscriptionEvent
	if err := db.Where("user_id = ? AND type = ?", user.ID, subscriptionEventDeleted).First(&deletedEvent).Error; err != nil {
		t.Fatalf("load deletion event failed: %v", err)
	}
	if deletedEvent.PreviousAmount == nil || *deletedEvent.PreviousAmount != legacyAmount {
		t.Fatalf("deletion event previous_amount = %v, want %v", deletedEvent.PreviousAmount, legacyAmount)
	}
	if deletedEvent.PreviousMonthlyAmount == nil || *deletedEvent.PreviousMonthlyAmount != rawMonthlyAmount {
		t.Fatalf("deletion event previous_monthly_amount = %v, want verbatim %v", deletedEvent.PreviousMonthlyAmount, rawMonthlyAmount)
	}
}
