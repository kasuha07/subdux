package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestGetSubscriptionDetailAggregatesHistoryLogsAndUpcomingCharges(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.NotificationLog{}); err != nil {
		t.Fatalf("failed to migrate notification logs: %v", err)
	}
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Video Pro",
		Amount:          10,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	updatedAmount := 15.0
	if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{Amount: &updatedAmount}); err != nil {
		t.Fatalf("update subscription failed: %v", err)
	}

	if err := db.Create(&model.NotificationLog{
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelType:    "email",
		NotifyDate:     mustDate(t, "2026-03-12"),
		Status:         "sent",
		SentAt:         time.Date(2026, 3, 12, 8, 30, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("create notification log failed: %v", err)
	}

	detail, err := service.GetDetail(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}

	if got, want := detail.Subscription.Name, "Video Pro"; got != want {
		t.Fatalf("subscription name = %q, want %q", got, want)
	}
	if got, want := len(detail.Timeline), 2; got != want {
		t.Fatalf("timeline length = %d, want %d", got, want)
	}
	if got, want := detail.Timeline[0].Type, subscriptionEventUpdated; got != want {
		t.Fatalf("latest event type = %q, want %q", got, want)
	}
	if got, want := len(detail.PriceHistory), 2; got != want {
		t.Fatalf("price history length = %d, want %d", got, want)
	}
	assertFloatEqual(t, detail.PriceHistory[0].Amount, 10, "initial price amount")
	assertFloatEqual(t, detail.PriceHistory[1].Amount, 15, "updated price amount")

	if got, want := len(detail.NotificationLogs), 1; got != want {
		t.Fatalf("notification logs length = %d, want %d", got, want)
	}
	if got, want := detail.NotificationLogs[0].NotifyDate, "2026-03-12"; got != want {
		t.Fatalf("notify date = %q, want %q", got, want)
	}

	if got, want := len(detail.UpcomingCharges), 12; got != want {
		t.Fatalf("upcoming charges length = %d, want %d", got, want)
	}
	if got, want := detail.UpcomingCharges[0].Date, "2026-03-15"; got != want {
		t.Fatalf("first upcoming charge date = %q, want %q", got, want)
	}
	if got, want := detail.UpcomingCharges[11].Date, "2027-02-15"; got != want {
		t.Fatalf("last upcoming charge date = %q, want %q", got, want)
	}
	if !detail.Calendar.HasUpcomingEvent || detail.Calendar.NextEventDate == nil {
		t.Fatal("calendar should expose the next upcoming event")
	}
	if got, want := *detail.Calendar.NextEventDate, "2026-03-15"; got != want {
		t.Fatalf("calendar next event = %q, want %q", got, want)
	}
}

func TestGetSubscriptionDetailLimitsManualRenewToCurrentCharge(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.NotificationLog{}); err != nil {
		t.Fatalf("failed to migrate notification logs: %v", err)
	}
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	yearly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Annual Tool",
		Amount:          120,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeManualRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &yearly,
		IntervalUnit:    intervalUnitYear,
		NextBillingDate: "2026-04-01",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	detail, err := service.GetDetail(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}

	if got, want := len(detail.UpcomingCharges), 1; got != want {
		t.Fatalf("manual renewal upcoming charge length = %d, want %d", got, want)
	}
	if got, want := detail.UpcomingCharges[0].Date, "2026-04-01"; got != want {
		t.Fatalf("manual renewal charge date = %q, want %q", got, want)
	}
}

func TestGetSubscriptionDetailExcludesCancelAtPeriodEndUpcomingCharges(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.NotificationLog{}); err != nil {
		t.Fatalf("failed to migrate notification logs: %v", err)
	}
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Ending Tool",
		Amount:          20,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeCancelAtPeriodEnd,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	detail, err := service.GetDetail(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}

	if got, want := len(detail.UpcomingCharges), 0; got != want {
		t.Fatalf("upcoming charges length = %d, want %d", got, want)
	}
	if detail.Calendar.HasUpcomingEvent || detail.Calendar.NextEventDate != nil {
		t.Fatal("calendar summary should not expose an upcoming charge for cancel-at-period-end subscription")
	}
}

func TestGetSubscriptionDetailPriceHistoryIsNotLimitedByTimeline(t *testing.T) {
	restoreClock := pkg.SetNowForTest(mustDate(t, "2026-03-01"))
	t.Cleanup(restoreClock)

	db := newTestDB(t)
	if err := db.AutoMigrate(&model.NotificationLog{}); err != nil {
		t.Fatalf("failed to migrate notification logs: %v", err)
	}
	user := createTestUser(t, db)
	service := NewSubscriptionService(db)

	monthly := 1
	sub, err := service.Create(user.ID, CreateSubscriptionInput{
		Name:            "Archive Plan",
		Amount:          10,
		Currency:        "USD",
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &monthly,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: "2026-03-15",
	})
	if err != nil {
		t.Fatalf("create subscription failed: %v", err)
	}

	for i := 0; i < 55; i++ {
		paymentMethod := model.PaymentMethod{
			UserID: user.ID,
			Name:   fmt.Sprintf("Method %02d", i),
		}
		if err := db.Create(&paymentMethod).Error; err != nil {
			t.Fatalf("create payment method %d failed: %v", i, err)
		}
		if _, err := service.Update(user.ID, sub.ID, UpdateSubscriptionInput{PaymentMethodID: &paymentMethod.ID}); err != nil {
			t.Fatalf("update payment method %d failed: %v", i, err)
		}
	}

	detail, err := service.GetDetail(user.ID, sub.ID)
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}

	if got, want := len(detail.Timeline), 50; got != want {
		t.Fatalf("timeline length = %d, want %d", got, want)
	}
	if got, want := len(detail.PriceHistory), 1; got != want {
		t.Fatalf("price history length = %d, want %d", got, want)
	}
	if got, want := detail.PriceHistory[0].Type, subscriptionEventCreated; got != want {
		t.Fatalf("price history event type = %q, want %q", got, want)
	}
	assertFloatEqual(t, detail.PriceHistory[0].Amount, 10, "initial price amount")
}
