package service

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newNotificationDaysBeforeTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-notification-days-before-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}, &model.NotificationPolicy{}); err != nil {
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

func TestUpdatePolicyRejectsDaysBeforeAboveMax(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewNotificationService(db, nil, nil)
	invalid := maxNotificationDaysBefore + 1

	_, err := service.UpdatePolicy(user.ID, UpdatePolicyInput{DaysBefore: &invalid})
	if err == nil {
		t.Fatal("UpdatePolicy() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "days_before must be between 0 and 10") {
		t.Fatalf("UpdatePolicy() error = %q, want days_before validation", err.Error())
	}
}

func TestCreateSubscriptionRejectsNotifyDaysBeforeAboveMax(t *testing.T) {
	db := newNotificationDaysBeforeTestDB(t)
	user := createNotificationDaysBeforeTestUser(t, db)
	service := NewSubscriptionService(db)
	invalid := maxNotificationDaysBefore + 1

	_, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:             "Example subscription",
		Amount:           9.99,
		BillingType:      billingTypeOneTime,
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
	service := NewSubscriptionService(db)
	initialNotifyDaysBefore := 3

	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:             "Example subscription",
		Amount:           9.99,
		BillingType:      billingTypeOneTime,
		NextBillingDate:  "2025-01-01",
		NotifyDaysBefore: &initialNotifyDaysBefore,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	invalid := maxNotificationDaysBefore + 1
	_, err = service.Update(user.ID, sub.ID, UpdateSubscriptionInput{
		NotifyDaysBefore: &invalid,
	})
	if err == nil {
		t.Fatal("Update() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "notify_days_before must be between 0 and 10") {
		t.Fatalf("Update() error = %q, want notify_days_before validation", err.Error())
	}
}
