package notification

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
)

func newNotificationDaysBeforeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-notification-days-before-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}, &model.SubscriptionEvent{}, &model.NotificationPolicy{}, &model.NotificationOutbox{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createNotificationDaysBeforeTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "notify-days-before-user",
		Email:    "notify-days-before@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}

	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func intPtr(value int) *int {
	return &value
}

func TestUpdatePolicyRejectsDaysBeforeAboveMax(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewService(db, nil, nil)
	invalid := maxNotificationDaysBefore + 1

	_, err := service.UpdatePolicy(user.ID, UpdatePolicyInput{DaysBefore: &invalid})
	if err == nil {
		t.Fatal("UpdatePolicy() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "days_before must be between 0 and 10") {
		t.Fatalf("UpdatePolicy() error = %q, want days_before validation", err.Error())
	}
}

func TestUpdatePolicySavesManualRenewDailyPreference(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewService(db, nil, nil)
	enabled := true

	policy, err := service.UpdatePolicy(user.ID, UpdatePolicyInput{NotifyManualRenewDaily: &enabled})
	if err != nil {
		t.Fatalf("UpdatePolicy() error = %v", err)
	}
	if !policy.NotifyManualRenewDaily {
		t.Fatal("NotifyManualRenewDaily = false, want true")
	}
	if policy.DaysBefore != 3 || !policy.NotifyOnDueDay {
		t.Fatalf("policy defaults = (%d, %v), want (3, true)", policy.DaysBefore, policy.NotifyOnDueDay)
	}
}

func TestDefaultPolicyDisablesManualRenewDailyPreference(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewService(db, nil, nil)

	policy, err := service.GetPolicy(user.ID)
	if err != nil {
		t.Fatalf("GetPolicy() error = %v", err)
	}
	if policy.NotifyManualRenewDaily {
		t.Fatal("NotifyManualRenewDaily = true, want false default")
	}
}

func TestCreateSubscriptionRejectsNotifyDaysBeforeAboveMax(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := subscriptionservice.NewService(db)
	invalid := maxNotificationDaysBefore + 1

	_, err := service.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:             "Example subscription",
		Amount:           9.99,
		BillingType:      billingTypeRecurring,
		RecurrenceType:   recurrenceTypeInterval,
		IntervalCount:    intPtr(1),
		IntervalUnit:     intervalUnitMonth,
		NextBillingDate:  "2025-01-01",
		NotifyDaysBefore: &invalid,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "notify_days_before must be between 0 and 10") {
		t.Fatalf("Create() error = %q, want notify_days_before validation", err.Error())
	}
}

func TestUpdateSubscriptionRejectsNotifyDaysBeforeAboveMax(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := subscriptionservice.NewService(db)
	initialNotifyDaysBefore := 3

	sub, err := service.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:             "Example subscription",
		Amount:           9.99,
		BillingType:      billingTypeRecurring,
		RecurrenceType:   recurrenceTypeInterval,
		IntervalCount:    intPtr(1),
		IntervalUnit:     intervalUnitMonth,
		NextBillingDate:  "2025-01-01",
		NotifyDaysBefore: &initialNotifyDaysBefore,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	invalid := maxNotificationDaysBefore + 1
	_, err = service.Update(user.ID, sub.ID, subscriptionservice.UpdateSubscriptionInput{
		NotifyDaysBefore: &invalid,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "notify_days_before must be between 0 and 10") {
		t.Fatalf("Update() error = %q, want notify_days_before validation", err.Error())
	}
}

func TestUpdateSubscriptionAllowsClearingNotifyOverridesWithNull(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := subscriptionservice.NewService(db)

	initialEnabled := false
	initialDays := 3
	sub, err := service.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:             "Example subscription",
		Amount:           9.99,
		BillingType:      billingTypeRecurring,
		RecurrenceType:   recurrenceTypeInterval,
		IntervalCount:    intPtr(1),
		IntervalUnit:     intervalUnitMonth,
		NextBillingDate:  "2025-01-01",
		NotifyEnabled:    &initialEnabled,
		NotifyDaysBefore: &initialDays,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updateInput := subscriptionservice.UpdateSubscriptionInput{
		NotifyEnabledSet:    true,
		NotifyDaysBeforeSet: true,
	}

	updated, err := service.Update(user.ID, sub.ID, updateInput)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.NotifyEnabled != nil {
		t.Fatalf("updated.NotifyEnabled = %v, want nil", *updated.NotifyEnabled)
	}
	if updated.NotifyDaysBefore != nil {
		t.Fatalf("updated.NotifyDaysBefore = %v, want nil", *updated.NotifyDaysBefore)
	}
}
