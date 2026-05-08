package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
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
		&model.NotificationPolicy{},
		&model.NotificationChannel{},
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
	service := NewSubscriptionService(db)

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

	if _, err := service.List(user.ID); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	var refreshedRecurring model.Subscription
	if err := db.First(&refreshedRecurring, recurring.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if refreshedRecurring.NextBillingDate == nil {
		t.Fatal("recurring next billing date should not be nil")
	}
	expectedRecurring := nextIntervalOccurrence(overdueRecurring, today, intervalCount, intervalUnitWeek)
	if got, want := refreshedRecurring.NextBillingDate.Format("2006-01-02"), expectedRecurring.Format("2006-01-02"); got != want {
		t.Fatalf("recurring next billing date = %s, want %s", got, want)
	}
}

func TestCreateRejectsLegacyOneTimeSubscriptions(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

	_, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Legacy buyout",
		Amount:          4.99,
		BillingType:     billingTypeOneTime,
		NextBillingDate: setSubscriptionRolloverTestNow(t).Format("2006-01-02"),
	})
	if err == nil {
		t.Fatal("Create() error = nil, want unsupported one_time error")
	}
	if got, want := err.Error(), "one_time subscriptions are no longer supported"; got != want {
		t.Fatalf("Create() error = %q, want %q", got, want)
	}
}

func TestDashboardAutoAdvancesOverdueRecurringNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

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

	if _, err := service.GetDashboardSummary(user.ID, "USD", nil); err != nil {
		t.Fatalf("GetDashboardSummary() error = %v", err)
	}

	var refreshed model.Subscription
	if err := db.First(&refreshed, sub.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if refreshed.NextBillingDate == nil {
		t.Fatal("recurring next billing date should not be nil")
	}
	expected := nextIntervalOccurrence(overdueRecurring, today, intervalCount, intervalUnitMonth)
	if got, want := refreshed.NextBillingDate.Format("2006-01-02"), expected.Format("2006-01-02"); got != want {
		t.Fatalf("recurring next billing date = %s, want %s", got, want)
	}
}

func TestListEndsOverdueManualRenewSubscription(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

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

	if _, err := service.List(user.ID); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	var refreshed model.Subscription
	if err := db.First(&refreshed, sub.ID).Error; err != nil {
		t.Fatalf("load manual renew subscription error = %v", err)
	}
	if got, want := refreshed.Status, subscriptionStatusEnded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if refreshed.EndsAt == nil {
		t.Fatal("ends_at should be set for overdue manual renew subscription")
	}
	if got, want := refreshed.EndsAt.Format("2006-01-02"), overdue.Format("2006-01-02"); got != want {
		t.Fatalf("ends_at = %s, want %s", got, want)
	}
}

func TestListEndsCancelAtPeriodEndSubscription(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

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

	if _, err := service.List(user.ID); err != nil {
		t.Fatalf("List() error = %v", err)
	}

	var refreshed model.Subscription
	if err := db.First(&refreshed, sub.ID).Error; err != nil {
		t.Fatalf("load cancel_at_period_end subscription error = %v", err)
	}
	if got, want := refreshed.Status, subscriptionStatusEnded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if refreshed.EndsAt == nil {
		t.Fatal("ends_at should be set for cancel_at_period_end subscription")
	}
	if got, want := refreshed.EndsAt.Format("2006-01-02"), periodEnd.Format("2006-01-02"); got != want {
		t.Fatalf("ends_at = %s, want %s", got, want)
	}
}

func TestMarkManualRenewedAdvancesNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	service := NewSubscriptionService(db)

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

func TestProcessUserNotificationsAutoAdvancesOverdueRecurringNextBillingDate(t *testing.T) {
	db := newSubscriptionRolloverTestDB(t)
	user := createSubscriptionRolloverTestUser(t, db)
	subscriptionService := NewSubscriptionService(db)
	notificationService := NewNotificationService(db, nil, nil)

	today := setSubscriptionRolloverTestNow(t)
	overdueRecurring := today.AddDate(-1, 0, 0)
	intervalCount := 1

	sub, err := subscriptionService.Create(user.ID, CreateSubscriptionInput{
		Name:            "Notification overdue recurring",
		Amount:          5.99,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitYear,
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create recurring subscription error = %v", err)
	}

	if err := notificationService.processUserNotifications(user.ID); err != nil {
		t.Fatalf("processUserNotifications() error = %v", err)
	}

	var refreshed model.Subscription
	if err := db.First(&refreshed, sub.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if refreshed.NextBillingDate == nil {
		t.Fatal("recurring next billing date should not be nil")
	}
	expected := nextIntervalOccurrence(overdueRecurring, today, intervalCount, intervalUnitYear)
	if got, want := refreshed.NextBillingDate.Format("2006-01-02"), expected.Format("2006-01-02"); got != want {
		t.Fatalf("recurring next billing date = %s, want %s", got, want)
	}
}
