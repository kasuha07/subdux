package migrations

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func TestMigrateHistoryRetentionKeepsRecentRowsPerUser(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "history-retention-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database error = %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.Subscription{}, &model.NotificationLog{}, &model.AuditEvent{}); err != nil {
		t.Fatalf("auto-migrate models error = %v", err)
	}

	now := time.Date(2026, 8, 28, 0, 0, 0, 0, time.UTC)
	users := []model.User{
		{Username: "history-primary", Email: "history-primary@example.com", Password: "hash"},
		{Username: "history-other", Email: "history-other@example.com", Password: "hash"},
	}
	for i := range users {
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatalf("create user %d error = %v", i, err)
		}
	}

	intervalCount := 1
	subscription := model.Subscription{
		UserID:         users[0].ID,
		Name:           "History subscription",
		Amount:         10,
		Currency:       "USD",
		Status:         "active",
		RenewalMode:    "auto_renew",
		BillingType:    "recurring",
		RecurrenceType: "interval",
		IntervalCount:  &intervalCount,
		IntervalUnit:   "month",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create subscription error = %v", err)
	}

	for i := 0; i < recentHistoryRetentionLimit+2; i++ {
		occurredAt := now.Add(time.Duration(i) * time.Minute)
		if err := db.Create(&model.NotificationLog{
			UserID:         users[0].ID,
			SubscriptionID: subscription.ID,
			ChannelType:    "email",
			NotifyDate:     occurredAt,
			Status:         "sent",
			SentAt:         occurredAt,
		}).Error; err != nil {
			t.Fatalf("create notification log %d error = %v", i, err)
		}
		if err := db.Create(&model.AuditEvent{
			EventID:      fmt.Sprintf("primary-%03d", i),
			OccurredAt:   occurredAt,
			UserID:       users[0].ID,
			ToolName:     "create_subscription",
			ResourceType: "subscription",
			Action:       "create",
			Status:       "success",
		}).Error; err != nil {
			t.Fatalf("create audit event %d error = %v", i, err)
		}
	}

	otherEvent := model.AuditEvent{
		EventID:      "other-000",
		OccurredAt:   now,
		UserID:       users[1].ID,
		ToolName:     "create_subscription",
		ResourceType: "subscription",
		Action:       "create",
		Status:       "success",
	}
	if err := db.Create(&otherEvent).Error; err != nil {
		t.Fatalf("create other audit event error = %v", err)
	}

	if err := migrateHistoryRetention(db); err != nil {
		t.Fatalf("migrateHistoryRetention() error = %v", err)
	}

	var notificationLogs []model.NotificationLog
	if err := db.Where("user_id = ?", users[0].ID).
		Order("sent_at DESC, id DESC").Find(&notificationLogs).Error; err != nil {
		t.Fatalf("load notification logs error = %v", err)
	}
	if len(notificationLogs) != recentHistoryRetentionLimit {
		t.Fatalf("notification log count = %d, want %d", len(notificationLogs), recentHistoryRetentionLimit)
	}
	if got, want := notificationLogs[0].NotifyDate, now.Add(time.Duration(recentHistoryRetentionLimit+1)*time.Minute); !got.Equal(want) {
		t.Fatalf("newest notification date = %v, want %v", got, want)
	}
	if got, want := notificationLogs[len(notificationLogs)-1].NotifyDate, now.Add(2*time.Minute); !got.Equal(want) {
		t.Fatalf("oldest retained notification date = %v, want %v", got, want)
	}

	var auditEvents []model.AuditEvent
	if err := db.Where("user_id = ?", users[0].ID).
		Order("occurred_at DESC, event_id DESC").Find(&auditEvents).Error; err != nil {
		t.Fatalf("load audit events error = %v", err)
	}
	if len(auditEvents) != recentHistoryRetentionLimit {
		t.Fatalf("audit event count = %d, want %d", len(auditEvents), recentHistoryRetentionLimit)
	}
	wantNewestAuditID := fmt.Sprintf("primary-%03d", recentHistoryRetentionLimit+1)
	if auditEvents[0].EventID != wantNewestAuditID || auditEvents[len(auditEvents)-1].EventID != "primary-002" {
		t.Fatalf("retained audit event range = (%s, %s), want (%s, primary-002)", auditEvents[0].EventID, auditEvents[len(auditEvents)-1].EventID, wantNewestAuditID)
	}

	var otherAuditCount int64
	if err := db.Model(&model.AuditEvent{}).Where("user_id = ?", users[1].ID).Count(&otherAuditCount).Error; err != nil {
		t.Fatalf("count other audit events error = %v", err)
	}
	if otherAuditCount != 1 {
		t.Fatalf("other user audit event count = %d, want 1", otherAuditCount)
	}
}
