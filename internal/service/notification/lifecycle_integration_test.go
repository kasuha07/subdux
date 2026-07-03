package notification

import (
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

func TestProcessUserNotificationsAutoAdvancesOverdueRecurringNextBillingDate(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	subscriptions := subscriptionservice.NewService(db)
	notifications := NewService(db, nil, nil)

	today := setNotificationLifecycleTestNow(t)
	overdueRecurring := today.AddDate(-1, 0, 0)
	intervalCount := 1

	sub, err := subscriptions.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:            "Notification overdue recurring",
		Amount:          5.99,
		BillingType:     subscriptionservice.BillingTypeRecurring,
		RecurrenceType:  "interval",
		IntervalCount:   &intervalCount,
		IntervalUnit:    "year",
		NextBillingDate: overdueRecurring.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create recurring subscription error = %v", err)
	}

	if err := notifications.processUserNotifications(user.ID); err != nil {
		t.Fatalf("processUserNotifications() error = %v", err)
	}

	var refreshed model.Subscription
	if err := db.First(&refreshed, sub.ID).Error; err != nil {
		t.Fatalf("load recurring subscription error = %v", err)
	}
	if refreshed.NextBillingDate == nil {
		t.Fatal("recurring next billing date should not be nil")
	}
	if got, want := refreshed.NextBillingDate.Format("2006-01-02"), "2026-03-15"; got != want {
		t.Fatalf("recurring next billing date = %s, want %s", got, want)
	}
}

func TestProcessUserNotificationsCreatesEndingSoonOutboxForCancelAtPeriodEndSubscription(t *testing.T) {
	db := newTestDB(t)
	user := createTestUser(t, db)
	subscriptions := subscriptionservice.NewService(db)
	if err := db.Create(&model.NotificationTemplate{
		UserID:   user.ID,
		Format:   "plaintext",
		Template: "{{.SubscriptionName}}|{{.BillingDate}}|{{.EventType}}",
	}).Error; err != nil {
		t.Fatalf("create notification template failed: %v", err)
	}
	notifications := NewService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))

	today := setNotificationLifecycleTestNow(t)
	intervalCount := 1
	sub, err := subscriptions.Create(user.ID, subscriptionservice.CreateSubscriptionInput{
		Name:            "Ending notification",
		Amount:          5.99,
		Status:          subscriptionservice.StatusActive,
		RenewalMode:     subscriptionservice.RenewalModeCancelAtPeriodEnd,
		BillingType:     subscriptionservice.BillingTypeRecurring,
		RecurrenceType:  "interval",
		IntervalCount:   &intervalCount,
		IntervalUnit:    "month",
		NextBillingDate: today.Format("2006-01-02"),
	})
	if err != nil {
		t.Fatalf("Create cancel-at-period-end subscription error = %v", err)
	}

	if err := db.Create(&model.NotificationPolicy{
		UserID:         user.ID,
		DaysBefore:     3,
		NotifyOnDueDay: true,
	}).Error; err != nil {
		t.Fatalf("create notification policy failed: %v", err)
	}
	if err := db.Create(&model.NotificationChannel{
		UserID:  user.ID,
		Type:    "unsupported-test",
		Enabled: true,
		Config:  "{}",
	}).Error; err != nil {
		t.Fatalf("create notification channel failed: %v", err)
	}

	if err := notifications.processUserNotifications(user.ID); err != nil {
		t.Fatalf("processUserNotifications() error = %v", err)
	}

	var jobs []model.NotificationOutbox
	if err := db.Where("subscription_id = ?", sub.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("load outbox jobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("outbox job count = %d, want 1", len(jobs))
	}
	if jobs[0].TriggerType != notificationTriggerEndingSoon {
		t.Fatalf("trigger_type = %q, want %q", jobs[0].TriggerType, notificationTriggerEndingSoon)
	}
	if jobs[0].Message != "Ending notification|2026-03-15|ending_soon" {
		t.Fatalf("message = %q, want ending_soon event", jobs[0].Message)
	}
}
