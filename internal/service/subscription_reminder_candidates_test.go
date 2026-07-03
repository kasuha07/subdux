package service

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

func newSubscriptionReminderCandidateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-subscription-reminder-candidates-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}, &model.SubscriptionEvent{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func createSubscriptionReminderCandidateTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "reminder-candidate-user",
		Email:    "reminder-candidate@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user
}

func createSubscriptionReminderCandidateTestSubscription(
	t *testing.T,
	db *gorm.DB,
	sub model.Subscription,
) model.Subscription {
	t.Helper()

	if sub.Currency == "" {
		sub.Currency = "USD"
	}
	if sub.Status == "" {
		sub.Status = subscriptionStatusActive
	}
	if sub.RenewalMode == "" {
		sub.RenewalMode = renewalModeAutoRenew
	}
	if sub.BillingType == "" {
		sub.BillingType = billingTypeRecurring
	}
	if sub.RecurrenceType == "" {
		sub.RecurrenceType = recurrenceTypeInterval
	}
	if sub.IntervalCount == nil {
		count := 1
		sub.IntervalCount = &count
	}
	if sub.IntervalUnit == "" {
		sub.IntervalUnit = intervalUnitMonth
	}
	sub.Enabled = true

	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("failed to create subscription %q: %v", sub.Name, err)
	}
	return sub
}

func TestListSubscriptionReminderCandidatesCreatesExpectedTriggerCandidates(t *testing.T) {
	db := newSubscriptionReminderCandidateTestDB(t)
	user := createSubscriptionReminderCandidateTestUser(t, db)
	now := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	today := normalizeDateUTC(now)
	inTwoDays := today.AddDate(0, 0, 2)
	inThreeDays := today.AddDate(0, 0, 3)
	notifyDisabled := false

	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "Due today",
		Amount:          10,
		NextBillingDate: &today,
	})
	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "Three days out",
		Amount:          20,
		NextBillingDate: &inThreeDays,
	})
	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "Manual daily",
		Amount:          30,
		RenewalMode:     renewalModeManualRenew,
		NextBillingDate: &inTwoDays,
	})
	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "Ending soon",
		Amount:          40,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		EndsAt:          &inThreeDays,
		NextBillingDate: &inThreeDays,
	})
	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "Opted out",
		Amount:          50,
		NotifyEnabled:   &notifyDisabled,
		NextBillingDate: &today,
	})
	createSubscriptionReminderCandidateTestSubscription(t, db, model.Subscription{
		UserID:          user.ID,
		Name:            "One time",
		Amount:          60,
		BillingType:     "one_time",
		NextBillingDate: &today,
	})

	candidates, err := listSubscriptionReminderCandidates(context.Background(), db, user.ID, now, subscriptionReminderPolicy{
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: true,
	})
	if err != nil {
		t.Fatalf("listSubscriptionReminderCandidates() error = %v", err)
	}
	if len(candidates) != 4 {
		t.Fatalf("candidate count = %d, want 4: %#v", len(candidates), candidates)
	}

	byName := make(map[string]subscriptionReminderCandidate, len(candidates))
	for _, candidate := range candidates {
		byName[candidate.Template.Name] = candidate
	}

	assertCandidate := func(name, triggerType, eventType, notifyDate string, daysUntil int) {
		t.Helper()
		candidate, ok := byName[name]
		if !ok {
			t.Fatalf("missing candidate %q", name)
		}
		if candidate.TriggerType != triggerType {
			t.Fatalf("%s trigger = %q, want %q", name, candidate.TriggerType, triggerType)
		}
		if candidate.EventType != eventType {
			t.Fatalf("%s event type = %q, want %q", name, candidate.EventType, eventType)
		}
		if got := candidate.NotifyDate.Format("2006-01-02"); got != notifyDate {
			t.Fatalf("%s notify date = %s, want %s", name, got, notifyDate)
		}
		if candidate.DaysUntil != daysUntil {
			t.Fatalf("%s days until = %d, want %d", name, candidate.DaysUntil, daysUntil)
		}
	}

	assertCandidate("Due today", notificationTriggerDueDay, "auto_renew_reminder", "2026-03-10", 0)
	assertCandidate("Three days out", notificationTriggerDaysBefore, "auto_renew_reminder", "2026-03-13", 3)
	assertCandidate("Manual daily", notificationTriggerManualDaily, "manual_renew_reminder", "2026-03-12", 2)
	assertCandidate("Ending soon", notificationTriggerEndingSoon, "ending_soon", "2026-03-13", 3)

	if got := byName["Manual daily"].DedupeDate.Format("2006-01-02"); got != "2026-03-10" {
		t.Fatalf("manual daily dedupe date = %s, want scan date 2026-03-10", got)
	}
	if _, ok := byName["Opted out"]; ok {
		t.Fatal("disabled subscription produced a reminder candidate")
	}
	if _, ok := byName["One time"]; ok {
		t.Fatal("one-time subscription produced a reminder candidate")
	}
}

func TestSubscriptionReminderCandidateKeepsTemplateSnapshotBounded(t *testing.T) {
	assertFieldSet(t, reflect.TypeOf(subscriptionReminderCandidate{}), []string{
		"SubscriptionID",
		"UserID",
		"NotifyDate",
		"DedupeDate",
		"DaysUntil",
		"TriggerType",
		"EventType",
		"Template",
	})
	assertFieldSet(t, reflect.TypeOf(subscriptionReminderTemplateSnapshot{}), []string{
		"Name",
		"Amount",
		"Currency",
		"Status",
		"RenewalMode",
		"Category",
		"PaymentMethodID",
		"URL",
		"Notes",
	})
}

func TestSubscriptionReminderCandidateFromSubscriptionCopiesTemplateSnapshot(t *testing.T) {
	paymentMethodID := uint(42)
	notifyDate := mustDate(t, "2026-03-13")
	sub := model.Subscription{
		ID:              7,
		UserID:          11,
		Name:            "Snapshot Plan",
		Amount:          19.99,
		Currency:        "EUR",
		Status:          " Active ",
		RenewalMode:     " MANUAL_RENEW ",
		Category:        "Tools",
		PaymentMethodID: &paymentMethodID,
		URL:             "https://example.com/plan",
		Notes:           "keep this note",
	}

	candidate := subscriptionReminderCandidateFromSubscription(
		sub,
		notifyDate,
		notifyDate,
		3,
		notificationTriggerDaysBefore,
		"manual_renew_reminder",
	)
	paymentMethodID = 99

	if candidate.SubscriptionID != sub.ID || candidate.UserID != sub.UserID {
		t.Fatalf("candidate identity = (%d, %d), want (%d, %d)", candidate.SubscriptionID, candidate.UserID, sub.ID, sub.UserID)
	}
	if candidate.Template.Name != sub.Name || candidate.Template.Amount != sub.Amount || candidate.Template.Currency != sub.Currency {
		t.Fatalf("template snapshot basic fields = %#v, want values from subscription", candidate.Template)
	}
	if candidate.Template.Status != subscriptionStatusActive {
		t.Fatalf("template status = %q, want %q", candidate.Template.Status, subscriptionStatusActive)
	}
	if candidate.Template.RenewalMode != renewalModeManualRenew {
		t.Fatalf("template renewal mode = %q, want %q", candidate.Template.RenewalMode, renewalModeManualRenew)
	}
	if candidate.Template.PaymentMethodID == nil || *candidate.Template.PaymentMethodID != 42 {
		t.Fatalf("template payment method id = %v, want copied value 42", candidate.Template.PaymentMethodID)
	}
	if candidate.Template.URL != sub.URL || candidate.Template.Notes != sub.Notes || candidate.Template.Category != sub.Category {
		t.Fatalf("template snapshot display fields = %#v, want values from subscription", candidate.Template)
	}
}

func assertFieldSet(t *testing.T, typ reflect.Type, want []string) {
	t.Helper()

	got := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, typ.Field(i).Name)
	}
	sort.Strings(got)
	sort.Strings(want)

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, want %v", typ.Name(), got, want)
	}
}
