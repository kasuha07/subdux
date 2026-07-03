package subscription

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

func newSubscriptionRolloverTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-subscription-rollover-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.NotificationPolicy{},
		&model.NotificationChannel{},
		&model.NotificationTemplate{},
		&model.NotificationOutbox{},
		&model.NotificationLog{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createSubscriptionRolloverTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "rollover-user",
		Email:    "rollover@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func setSubscriptionRolloverTestNow(t *testing.T) time.Time {
	t.Helper()
	now := mustDate(t, "2026-03-15")
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	return normalizeDateUTC(now.In(pkg.GetSystemTimezone()))
}

func findSubscriptionByID(t *testing.T, subs []model.Subscription, id uint) model.Subscription {
	t.Helper()
	for _, sub := range subs {
		if sub.ID == id {
			return sub
		}
	}
	t.Fatalf("subscription %d not found in list", id)
	return model.Subscription{}
}

func TestNextRecurringBillingDateOnOrAfter(t *testing.T) {
	referenceDate := mustDate(t, "2026-02-22")

	t.Run("interval recurrence advances from anchor", func(t *testing.T) {
		nextBillingDate := mustDate(t, "2026-01-01")
		intervalCount := 2
		sub := model.Subscription{
			BillingType:     billingTypeRecurring,
			RecurrenceType:  recurrenceTypeInterval,
			IntervalCount:   &intervalCount,
			IntervalUnit:    intervalUnitWeek,
			NextBillingDate: &nextBillingDate,
		}

		next, changed := nextRecurringBillingDateOnOrAfter(&sub, referenceDate)
		if !changed {
			t.Fatal("expected recurring interval subscription to be advanced")
		}
		if next == nil {
			t.Fatal("expected advanced date to be set")
		}
		if got, want := next.Format("2006-01-02"), "2026-02-26"; got != want {
			t.Fatalf("advanced date = %s, want %s", got, want)
		}
	})

	t.Run("monthly date recurrence clamps to month end", func(t *testing.T) {
		nextBillingDate := mustDate(t, "2026-01-31")
		monthlyDay := 31
		sub := model.Subscription{
			BillingType:     billingTypeRecurring,
			RecurrenceType:  recurrenceTypeMonthlyDate,
			MonthlyDay:      &monthlyDay,
			NextBillingDate: &nextBillingDate,
		}

		next, changed := nextRecurringBillingDateOnOrAfter(&sub, referenceDate)
		if !changed {
			t.Fatal("expected recurring monthly subscription to be advanced")
		}
		if next == nil {
			t.Fatal("expected advanced date to be set")
		}
		if got, want := next.Format("2006-01-02"), "2026-02-28"; got != want {
			t.Fatalf("advanced date = %s, want %s", got, want)
		}
	})

	t.Run("yearly date recurrence handles leap day", func(t *testing.T) {
		nextBillingDate := mustDate(t, "2024-02-29")
		yearlyMonth := 2
		yearlyDay := 29
		sub := model.Subscription{
			BillingType:     billingTypeRecurring,
			RecurrenceType:  recurrenceTypeYearlyDate,
			YearlyMonth:     &yearlyMonth,
			YearlyDay:       &yearlyDay,
			NextBillingDate: &nextBillingDate,
		}

		next, changed := nextRecurringBillingDateOnOrAfter(&sub, referenceDate)
		if !changed {
			t.Fatal("expected recurring yearly subscription to be advanced")
		}
		if next == nil {
			t.Fatal("expected advanced date to be set")
		}
		if got, want := next.Format("2006-01-02"), "2026-02-28"; got != want {
			t.Fatalf("advanced date = %s, want %s", got, want)
		}
	})

}

func TestListAutoAdvancesOverdueRecurringNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	today := setSubscriptionRolloverTestNow(t)
	overdueRecurring := today.AddDate(0, 0, -10)

	intervalCount := 1
	recurring, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Recurring overdue",
		Amount:          9.99,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitWeek,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create recurring subscription error = %v", err)
	}

	subs, err := service.List(user.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// The read presents the rolled-forward billing date in memory.
	expectedRecurring := nextIntervalOccurrence(overdueRecurring, today, intervalCount, intervalUnitWeek)
	presented := findSubscriptionByID(t, subs, recurring.ID)
	if presented.NextBillingDate == nil {
		t.Fatal("presented next billing date should not be nil")
	}
	if got, want := presented.NextBillingDate.Format("2006-01-02"), expectedRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("presented next billing date = %s, want %s", got, want)
	}

	// But the read must not write: the stored row stays at the overdue date
	// until the background sweep (or a write path) persists the transition.
	var stored model.Subscription
	if err := db.First(&stored, recurring.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if stored.NextBillingDate == nil {
		t.Fatal("stored next billing date should not be nil")
	}
	if got, want := stored.NextBillingDate.Format("2006-01-02"), overdueRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("stored next billing date = %s, want %s (read must not persist)", got, want)
	}
}

func TestAutoAdvanceRecurringNextBillingDatesForUserOnlyAdvancesAutoRenew(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	today := setSubscriptionRolloverTestNow(t)
	overdueRecurring := today.AddDate(0, 0, -10)
	intervalCount := 1

	autoRenew, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Auto renew overdue",
		Amount:          9.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitWeek,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create auto renew subscription error = %v", err)
	}

	manualRenew, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual renew overdue",
		Amount:          9.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitWeek,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create manual renew subscription error = %v", err)
	}

	canceling, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Canceling overdue",
		Amount:          9.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitWeek,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create canceling subscription error = %v", err)
	}

	if err := autoAdvanceRecurringNextBillingDatesForUser(db, user.ID, today); err != nil {
		t.Fatalf("autoAdvanceRecurringNextBillingDatesForUser() error = %v", err)
	}

	var refreshedAutoRenew model.Subscription
	if err := db.First(&refreshedAutoRenew, autoRenew.ID).Error; err != nil {
		t.Fatalf("load auto renew subscription error = %v", err)
	}
	expected := nextIntervalOccurrence(overdueRecurring, today, intervalCount, intervalUnitWeek)
	if got, want := refreshedAutoRenew.NextBillingDate.Format("2006-01-02"), expected.Format("2006-01-02"); got != want {
		t.Fatalf("auto renew next billing date = %s, want %s", got, want)
	}

	var refreshedManualRenew model.Subscription
	if err := db.First(&refreshedManualRenew, manualRenew.ID).Error; err != nil {
		t.Fatalf("load manual renew subscription error = %v", err)
	}
	if got, want := refreshedManualRenew.NextBillingDate.Format("2006-01-02"), overdueRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("manual renew next billing date = %s, want %s", got, want)
	}

	var refreshedCanceling model.Subscription
	if err := db.First(&refreshedCanceling, canceling.ID).Error; err != nil {
		t.Fatalf("load canceling subscription error = %v", err)
	}
	if got, want := refreshedCanceling.NextBillingDate.Format("2006-01-02"), overdueRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("canceling next billing date = %s, want %s", got, want)
	}
}

func TestCreateRejectsNonRecurringBillingType(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	_, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Usage billed service",
		Amount:          4.99,
		BillingType:     "usage",
		NextBillingDate: setSubscriptionRolloverTestNow(t).Format("2006-01-02"),
	})
	if err == nil {
		t.Fatal("Create() error = nil, want non-recurring billing type error")
	}
	if got, want := err.Error(), "billing_type must be recurring"; got != want {
		t.Fatalf("Create() error = %q, want %q", got, want)
	}
}

func TestDashboardAutoAdvancesOverdueRecurringNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	today := setSubscriptionRolloverTestNow(t)
	overdueRecurring := today.AddDate(0, -2, 0)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Dashboard overdue recurring",
		Amount:          19.99,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create recurring subscription error = %v", err)
	}

	summary, err := service.GetDashboardSummary(user.ID, "USD", nil)
	if err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}

	// The auto-renew subscription is presented as active (rolled forward), so it
	// still contributes to the dashboard.
	if summary.ActiveCount != 1 {
		t.Fatalf("active_count = %d, want 1 (overdue auto-renew presented as active)", summary.ActiveCount)
	}

	// The read must not persist the rollover.
	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if stored.NextBillingDate == nil {
		t.Fatal("recurring next billing date should not be nil")
	}
	if got, want := stored.NextBillingDate.Format("2006-01-02"), overdueRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("stored next billing date = %s, want %s (read must not persist)", got, want)
	}
}

func TestListEndsOverdueManualRenewSubscription(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	overdue := setSubscriptionRolloverTestNow(t).AddDate(0, 0, -2)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual renew overdue",
		Amount:          12.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: overdue.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create manual renew subscription error = %v", err)
	}

	subs, err := service.List(user.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// The read presents the subscription as ended at its overdue billing date.
	presented := findSubscriptionByID(t, subs, sub.ID)
	if got, want := presented.Status, subscriptionStatusEnded; got != want {
		t.Fatalf("presented status = %q, want %q", got, want)
	}
	if presented.EndsAt == nil {
		t.Fatal("presented ends_at should be set for overdue manual renew subscription")
	}
	if got, want := presented.EndsAt.Format("2006-01-02"), overdue.Format("2006-01-02"); got != want {
		t.Fatalf("presented ends_at = %s, want %s", got, want)
	}

	// The read must not persist the transition.
	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load manual renew subscription error = %v", err)
	}
	if got, want := stored.Status, subscriptionStatusActive; got != want {
		t.Fatalf("stored status = %q, want %q (read must not persist)", got, want)
	}
}

func TestListEndsCancelAtPeriodEndSubscription(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	periodEnd := setSubscriptionRolloverTestNow(t).AddDate(0, 0, -1)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Cancel at period end overdue",
		Amount:          8.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: periodEnd.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create cancel_at_period_end subscription error = %v", err)
	}

	subs, err := service.List(user.ID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	// The read presents the subscription as ended at its period boundary.
	presented := findSubscriptionByID(t, subs, sub.ID)
	if got, want := presented.Status, subscriptionStatusEnded; got != want {
		t.Fatalf("presented status = %q, want %q", got, want)
	}
	if presented.EndsAt == nil {
		t.Fatal("presented ends_at should be set for cancel_at_period_end subscription")
	}
	if got, want := presented.EndsAt.Format("2006-01-02"), periodEnd.Format("2006-01-02"); got != want {
		t.Fatalf("presented ends_at = %s, want %s", got, want)
	}

	// The read must not persist the transition.
	var stored model.Subscription
	if err := db.First(&stored, sub.ID).Error; err != nil {
		t.Fatalf("load cancel_at_period_end subscription error = %v", err)
	}
	if got, want := stored.Status, subscriptionStatusActive; got != want {
		t.Fatalf("stored status = %q, want %q (read must not persist)", got, want)
	}
}

func TestMarkManualRenewedAdvancesNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewService(db)

	nextBillingDate := setSubscriptionRolloverTestNow(t).AddDate(0, 0, 3)
	intervalCount := 1

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Manual renew active",
		Amount:          6.99,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: nextBillingDate.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create manual renew subscription error = %v", err)
	}

	updated, err := service.MarkManualRenewed(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("MarkManualRenewed() error = %v", err)
	}

	if updated.NextBillingDate == nil {
		t.Fatal("next_billing_date should not be nil after mark renewed")
	}
	expected := nextIntervalOccurrence(nextBillingDate, nextBillingDate.AddDate(0, 0, 1), intervalCount, intervalUnitMonth)
	if got, want := updated.NextBillingDate.Format("2006-01-02"), expected.Format("2006-01-02"); got != want {
		t.Fatalf("next_billing_date = %s, want %s", got, want)
	}
	if got, want := updated.Status, subscriptionStatusActive; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
}
