package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

func newNotificationOutboxTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-notification-outbox-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.Subscription{},
		&model.SubscriptionEvent{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationTemplate{},
		&model.NotificationOutbox{},
		&model.NotificationLog{},
		&model.BackgroundTaskLease{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func createNotificationOutboxUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "outbox-user",
		Email:    "outbox@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func createNotificationOutboxSubscription(t *testing.T, db *gorm.DB, userID uint, nextBilling time.Time) model.Subscription {
	t.Helper()

	intervalCount := 1
	sub := model.Subscription{
		UserID:          userID,
		Name:            "Outbox Plan",
		Amount:          12.5,
		Currency:        "USD",
		Enabled:         true,
		Status:          subscriptionStatusActive,
		RenewalMode:     renewalModeAutoRenew,
		BillingType:     billingTypeRecurring,
		RecurrenceType:  recurrenceTypeInterval,
		IntervalCount:   &intervalCount,
		IntervalUnit:    intervalUnitMonth,
		NextBillingDate: &nextBilling,
		URL:             "https://example.com/subscription",
	}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatalf("failed to create subscription: %v", err)
	}
	return sub
}

func createNotificationOutboxChannel(t *testing.T, db *gorm.DB, userID uint, channelType, config string) model.NotificationChannel {
	t.Helper()

	channel := model.NotificationChannel{
		UserID:  userID,
		Type:    channelType,
		Enabled: true,
		Config:  config,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
	return channel
}

func createNotificationOutboxTemplate(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()

	template := model.NotificationTemplate{
		UserID:   userID,
		Format:   "plaintext",
		Template: "{{.SubscriptionName}}|{{.BillingDate}}|{{.EventType}}",
	}
	if err := db.Create(&template).Error; err != nil {
		t.Fatalf("failed to create notification template: %v", err)
	}
}

func TestEnqueuePendingNotificationsCreatesTriggerSpecificOutboxJobs(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now)
	createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	if err := db.Create(&model.NotificationPolicy{UserID: user.ID, DaysBefore: 0, NotifyOnDueDay: true}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("second EnqueuePendingNotifications() error = %v", err)
	}

	var jobs []model.NotificationOutbox
	if err := db.Find(&jobs).Error; err != nil {
		t.Fatalf("load outbox jobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("outbox job count = %d, want 1 after duplicate scans", len(jobs))
	}
	job := jobs[0]
	if job.TriggerType != notificationTriggerDueDay {
		t.Fatalf("trigger_type = %q, want %q", job.TriggerType, notificationTriggerDueDay)
	}
	if job.Status != notificationOutboxStatusPending {
		t.Fatalf("status = %q, want %q", job.Status, notificationOutboxStatusPending)
	}
	if job.DedupeKey != notificationOutboxDedupeKey(user.ID, job.SubscriptionID, "webhook", notificationTriggerDueDay, notifyDate) {
		t.Fatalf("unexpected dedupe key %q", job.DedupeKey)
	}
}

func TestNotificationOutboxDedupeSeparatesDaysBeforeAndDueDay(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)
	svc := NewNotificationService(db, nil, nil)

	for _, triggerType := range []string{notificationTriggerDaysBefore, notificationTriggerDueDay} {
		if err := svc.enqueueNotificationOutbox(notificationOutboxJob{
			userID:          user.ID,
			subscriptionID:  sub.ID,
			channel:         channel,
			triggerType:     triggerType,
			notifyDate:      notifyDate,
			message:         "hello",
			targetEmail:     user.Email,
			subscriptionURL: sub.URL,
		}); err != nil {
			t.Fatalf("enqueue trigger %s error = %v", triggerType, err)
		}
	}

	var count int64
	if err := db.Model(&model.NotificationOutbox{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("outbox count = %d, want 2 trigger-specific jobs", count)
	}
}

func TestEnqueuePendingNotificationsCreatesDailyManualRenewOutboxJobs(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now.AddDate(0, 0, 2))
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	if err := db.Create(&model.NotificationPolicy{
		UserID:                 user.ID,
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: true,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("second EnqueuePendingNotifications() error = %v", err)
	}

	var jobs []model.NotificationOutbox
	if err := db.Order("id ASC").Find(&jobs).Error; err != nil {
		t.Fatalf("load outbox jobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("outbox job count = %d, want 1 after duplicate same-day scans", len(jobs))
	}
	if jobs[0].TriggerType != notificationTriggerManualDaily {
		t.Fatalf("trigger_type = %q, want %q", jobs[0].TriggerType, notificationTriggerManualDaily)
	}
	if got, want := jobs[0].NotifyDate.Format("2006-01-02"), notifyDate.Format("2006-01-02"); got != want {
		t.Fatalf("notify_date = %s, want billing date %s", got, want)
	}

	restoreClock()
	restoreClock = pkg.SetNowForTest(now.AddDate(0, 0, 1))
	t.Cleanup(restoreClock)
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("third EnqueuePendingNotifications() error = %v", err)
	}

	var dailyCount int64
	if err := db.Model(&model.NotificationOutbox{}).
		Where("subscription_id = ? AND trigger_type = ?", sub.ID, notificationTriggerManualDaily).
		Count(&dailyCount).Error; err != nil {
		t.Fatalf("count daily manual-renew jobs failed: %v", err)
	}
	if dailyCount != 2 {
		t.Fatalf("daily manual-renew outbox count = %d, want 2 across two scan dates", dailyCount)
	}
}

func TestEnqueuePendingNotificationsSkipsManualRenewDailyWhenPreferenceDisabled(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now.AddDate(0, 0, 2))
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	if err := db.Create(&model.NotificationPolicy{
		UserID:                 user.ID,
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: false,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}

	var count int64
	if err := db.Model(&model.NotificationOutbox{}).Where("subscription_id = ?", sub.ID).Count(&count).Error; err != nil {
		t.Fatalf("count outbox jobs failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox job count = %d, want 0 when daily manual-renew preference is disabled", count)
	}
}

func TestEnqueuePendingNotificationsManualRenewDailyCoversDueDayWhenDueDayDisabled(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	if err := db.Model(&model.NotificationPolicy{}).Create(map[string]interface{}{
		"user_id":                   user.ID,
		"days_before":               3,
		"notify_on_due_day":         false,
		"notify_manual_renew_daily": true,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}

	var job model.NotificationOutbox
	if err := db.Where("subscription_id = ?", sub.ID).First(&job).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if job.TriggerType != notificationTriggerManualDaily {
		t.Fatalf("trigger_type = %q, want %q", job.TriggerType, notificationTriggerManualDaily)
	}
}

func TestEnqueuePendingNotificationsDoesNotDuplicateManualRenewDueDay(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)

	if err := db.Create(&model.NotificationPolicy{
		UserID:                 user.ID,
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: true,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.EnqueuePendingNotifications(); err != nil {
		t.Fatalf("EnqueuePendingNotifications() error = %v", err)
	}

	var jobs []model.NotificationOutbox
	if err := db.Where("subscription_id = ?", sub.ID).Find(&jobs).Error; err != nil {
		t.Fatalf("load outbox jobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("outbox job count = %d, want 1 due-day job", len(jobs))
	}
	if jobs[0].TriggerType != notificationTriggerDueDay {
		t.Fatalf("trigger_type = %q, want %q", jobs[0].TriggerType, notificationTriggerDueDay)
	}
}

func TestNotificationOutboxSkipsLegacySentLogsWithoutTriggerType(t *testing.T) {
	tests := []struct {
		name         string
		triggerValue interface{}
	}{
		{name: "empty trigger", triggerValue: ""},
		{name: "null trigger", triggerValue: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newNotificationOutboxTestDB(t)
			user := createNotificationOutboxUser(t, db)
			notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
			sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
			channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)
			svc := NewNotificationService(db, nil, nil)

			logEntry := map[string]interface{}{
				"user_id":         user.ID,
				"subscription_id": sub.ID,
				"channel_type":    channel.Type,
				"trigger_type":    tt.triggerValue,
				"notify_date":     notifyDate,
				"status":          notificationLogStatusSent,
				"sent_at":         notifyDate,
			}
			if err := db.Table("notification_logs").Create(logEntry).Error; err != nil {
				t.Fatalf("create legacy notification log failed: %v", err)
			}

			err := svc.enqueueNotificationOutbox(notificationOutboxJob{
				userID:          user.ID,
				subscriptionID:  sub.ID,
				channel:         channel,
				triggerType:     notificationTriggerDueDay,
				notifyDate:      notifyDate,
				message:         "hello",
				targetEmail:     user.Email,
				subscriptionURL: sub.URL,
			})
			if err != nil {
				t.Fatalf("enqueueNotificationOutbox() error = %v", err)
			}

			var count int64
			if err := db.Model(&model.NotificationOutbox{}).Count(&count).Error; err != nil {
				t.Fatalf("count outbox failed: %v", err)
			}
			if count != 0 {
				t.Fatalf("outbox count = %d, want 0 for legacy sent log", count)
			}
		})
	}
}

func TestNotificationOutboxSkipsLegacySentLogInOriginalTimezone(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load timezone failed: %v", err)
	}
	legacyNotifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, shanghai)
	sub := createNotificationOutboxSubscription(t, db, user.ID, normalizeDateUTC(legacyNotifyDate))
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"https://example.com/hook"}`)
	svc := NewNotificationService(db, nil, nil)

	logEntry := map[string]interface{}{
		"user_id":         user.ID,
		"subscription_id": sub.ID,
		"channel_type":    channel.Type,
		"notify_date":     legacyNotifyDate,
		"status":          notificationLogStatusSent,
		"sent_at":         legacyNotifyDate,
	}
	if err := db.Table("notification_logs").Create(logEntry).Error; err != nil {
		t.Fatalf("create legacy notification log failed: %v", err)
	}

	err = svc.enqueueNotificationOutbox(notificationOutboxJob{
		userID:          user.ID,
		subscriptionID:  sub.ID,
		channel:         channel,
		triggerType:     notificationTriggerDueDay,
		notifyDate:      legacyNotifyDate,
		message:         "hello",
		targetEmail:     user.Email,
		subscriptionURL: sub.URL,
	})
	if err != nil {
		t.Fatalf("enqueueNotificationOutbox() error = %v", err)
	}

	var count int64
	if err := db.Model(&model.NotificationOutbox{}).Count(&count).Error; err != nil {
		t.Fatalf("count outbox failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox count = %d, want 0 for legacy timezone sent log", count)
	}
}

func TestBackgroundTaskLeaseAllowsSingleOwnerUntilExpired(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	first := NewNotificationService(db, nil, nil)
	first.ownerID = "owner-a"
	second := NewNotificationService(db, nil, nil)
	second.ownerID = "owner-b"

	acquired, err := first.acquireBackgroundTaskLease("notification_scan", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("first acquire = %v, %v; want acquired", acquired, err)
	}

	acquired, err = second.acquireBackgroundTaskLease("notification_scan", time.Minute)
	if err != nil {
		t.Fatalf("second acquire error = %v", err)
	}
	if acquired {
		t.Fatal("second owner acquired unexpired lease")
	}

	restoreClock()
	restoreClock = pkg.SetNowForTest(now.Add(2 * time.Minute))
	t.Cleanup(restoreClock)
	acquired, err = second.acquireBackgroundTaskLease("notification_scan", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("second acquire after expiry = %v, %v; want acquired", acquired, err)
	}
}

func TestClaimDueNotificationOutboxIsOwnerExclusive(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      "claim-test",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelType:    "webhook",
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "hello",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	first := NewNotificationService(db, nil, nil)
	first.ownerID = "owner-a"
	second := NewNotificationService(db, nil, nil)
	second.ownerID = "owner-b"

	firstJobs, err := first.claimDueNotificationOutbox(context.Background(), 10, time.Minute)
	if err != nil {
		t.Fatalf("first claim error = %v", err)
	}
	if len(firstJobs) != 1 {
		t.Fatalf("first claim count = %d, want 1", len(firstJobs))
	}

	secondJobs, err := second.claimDueNotificationOutbox(context.Background(), 10, time.Minute)
	if err != nil {
		t.Fatalf("second claim error = %v", err)
	}
	if len(secondJobs) != 0 {
		t.Fatalf("second claim count = %d, want 0 while lease active", len(secondJobs))
	}
}

func TestDispatchNotificationOutboxSuccessWritesLogAndMarksSent(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      "dispatch-success",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "hello",
		TargetEmail:    user.Email,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Sent != 1 {
		t.Fatalf("summary = %#v, want claimed=1 sent=1", summary)
	}
	select {
	case <-requestCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxied webhook request")
	}

	var saved model.NotificationOutbox
	if err := db.First(&saved, job.ID).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if saved.Status != notificationOutboxStatusSent || saved.SentAt == nil {
		t.Fatalf("outbox status/sent_at = %q/%v, want sent timestamp", saved.Status, saved.SentAt)
	}

	var logEntry model.NotificationLog
	if err := db.Where("outbox_id = ?", job.ID).First(&logEntry).Error; err != nil {
		t.Fatalf("load notification log failed: %v", err)
	}
	if logEntry.Status != notificationLogStatusSent || logEntry.TriggerType != notificationTriggerDueDay {
		t.Fatalf("log status/trigger = %q/%q, want sent/%s", logEntry.Status, logEntry.TriggerType, notificationTriggerDueDay)
	}
}

func TestProcessPendingNotificationsEnqueuesAndDispatchesWebhook(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	createNotificationOutboxTemplate(t, db, user.ID)
	now := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	notifyDate := normalizeDateUTC(now)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)

	if err := db.Create(&model.NotificationPolicy{UserID: user.ID, DaysBefore: 0, NotifyOnDueDay: true}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}

	svc := NewNotificationService(db, NewNotificationTemplateService(db, NewTemplateValidator()), NewTemplateRenderer(NewTemplateValidator()))
	if err := svc.ProcessPendingNotifications(); err != nil {
		t.Fatalf("ProcessPendingNotifications() error = %v", err)
	}
	select {
	case <-requestCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for proxied webhook request")
	}

	var outbox model.NotificationOutbox
	if err := db.Where("subscription_id = ?", sub.ID).First(&outbox).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if outbox.Status != notificationOutboxStatusSent || outbox.SentAt == nil || outbox.TriggerType != notificationTriggerDueDay {
		t.Fatalf("outbox status/sent_at/trigger = %q/%v/%q, want sent timestamp due_day", outbox.Status, outbox.SentAt, outbox.TriggerType)
	}

	var logEntry model.NotificationLog
	if err := db.Where("outbox_id = ?", outbox.ID).First(&logEntry).Error; err != nil {
		t.Fatalf("load notification log failed: %v", err)
	}
	if logEntry.Status != notificationLogStatusSent {
		t.Fatalf("log status = %q, want sent", logEntry.Status)
	}
}

func TestDispatchNotificationOutboxCancelsDeletedChannelWithoutFallback(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	deletedChannel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://deleted.example.com/hook","method":"POST"}`)
	createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://fallback.example.com/hook","method":"POST"}`)
	if err := db.Delete(&model.NotificationChannel{}, deletedChannel.ID).Error; err != nil {
		t.Fatalf("delete notification channel failed: %v", err)
	}

	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	job := model.NotificationOutbox{
		DedupeKey:      "deleted-channel-no-fallback",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &deletedChannel.ID,
		ChannelType:    deletedChannel.Type,
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "hello",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Cancelled != 1 || summary.Sent != 0 {
		t.Fatalf("summary = %#v, want claimed=1 cancelled=1 sent=0", summary)
	}
	select {
	case <-requestCh:
		t.Fatal("dispatcher sent through a fallback channel after original channel was deleted")
	default:
	}

	var saved model.NotificationOutbox
	if err := db.First(&saved, job.ID).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if saved.Status != notificationOutboxStatusCancelled {
		t.Fatalf("outbox status = %q, want cancelled", saved.Status)
	}
}

func TestDispatchNotificationOutboxCancelsWhenQueuedBillingDateIsStale(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      "stale-billing-date",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "old billing date",
		TargetEmail:    user.Email,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	updatedBillingDate := notifyDate.AddDate(0, 0, 10)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("next_billing_date", updatedBillingDate).Error; err != nil {
		t.Fatalf("update subscription billing date failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Cancelled != 1 || summary.Sent != 0 {
		t.Fatalf("summary = %#v, want claimed=1 cancelled=1 sent=0", summary)
	}
	select {
	case <-requestCh:
		t.Fatal("dispatcher sent a stale queued reminder after billing date changed")
	default:
	}

	var saved model.NotificationOutbox
	if err := db.First(&saved, job.ID).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if saved.Status != notificationOutboxStatusCancelled {
		t.Fatalf("outbox status = %q, want cancelled", saved.Status)
	}
}

func TestDispatchNotificationOutboxCancelsWhenQueuedTriggerNoLongerMatches(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	notifyDate := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      "stale-trigger",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerDaysBefore,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "old reminder timing",
		TargetEmail:    user.Email,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	updatedDaysBefore := 5
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("notify_days_before", updatedDaysBefore).Error; err != nil {
		t.Fatalf("update subscription notify days before failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Cancelled != 1 || summary.Sent != 0 {
		t.Fatalf("summary = %#v, want claimed=1 cancelled=1 sent=0", summary)
	}
	select {
	case <-requestCh:
		t.Fatal("dispatcher sent a queued reminder after reminder timing changed")
	default:
	}
}

func TestDispatchNotificationOutboxCancelsManualRenewDailyWhenPreferenceDisabled(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	notifyDate := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)
	if err := db.Create(&model.NotificationPolicy{
		UserID:                 user.ID,
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: false,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      notificationOutboxDedupeKeyForTrigger(user.ID, sub.ID, channel.Type, notificationTriggerManualDaily, notifyDate, now),
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerManualDaily,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "manual renewal reminder",
		TargetEmail:    user.Email,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Cancelled != 1 || summary.Sent != 0 {
		t.Fatalf("summary = %#v, want claimed=1 cancelled=1 sent=0", summary)
	}
	select {
	case <-requestCh:
		t.Fatal("dispatcher sent manual-renew daily reminder after preference was disabled")
	default:
	}
}

func TestDispatchNotificationOutboxCancelsStaleManualRenewDailyJob(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "test-settings-key")

	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	scheduledAt := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	dispatchAt := scheduledAt.AddDate(0, 0, 1)
	notifyDate := time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	if err := db.Model(&model.Subscription{}).
		Where("id = ? AND user_id = ?", sub.ID, user.ID).
		Update("renewal_mode", renewalModeManualRenew).Error; err != nil {
		t.Fatalf("update renewal mode failed: %v", err)
	}
	requestCh := make(chan struct{}, 1)
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requestCh <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer proxyServer.Close()
	seedProxySettings(t, db, "true", "http", proxyServer.URL)
	channel := createNotificationOutboxChannel(t, db, user.ID, "webhook", `{"url":"http://notify.example.com/hook","method":"POST"}`)
	if err := db.Create(&model.NotificationPolicy{
		UserID:                 user.ID,
		DaysBefore:             3,
		NotifyOnDueDay:         true,
		NotifyManualRenewDaily: true,
	}).Error; err != nil {
		t.Fatalf("failed to create notification policy: %v", err)
	}
	restoreClock := pkg.SetNowForTest(dispatchAt)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      notificationOutboxDedupeKeyForTrigger(user.ID, sub.ID, channel.Type, notificationTriggerManualDaily, notifyDate, scheduledAt),
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerManualDaily,
		NotifyDate:     notifyDate,
		ScheduledFor:   scheduledAt,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    5,
		NextAttemptAt:  dispatchAt,
		Message:        "stale manual renewal reminder",
		TargetEmail:    user.Email,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Cancelled != 1 || summary.Sent != 0 {
		t.Fatalf("summary = %#v, want claimed=1 cancelled=1 sent=0", summary)
	}
	select {
	case <-requestCh:
		t.Fatal("dispatcher sent a stale manual-renew daily reminder on a later day")
	default:
	}
}

func TestDispatchNotificationOutboxRetriesThenFails(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	channel := createNotificationOutboxChannel(t, db, user.ID, "unsupported-test", `{}`)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)

	job := model.NotificationOutbox{
		DedupeKey:      "dispatch-failure",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelID:      &channel.ID,
		ChannelType:    channel.Type,
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusPending,
		MaxAttempts:    1,
		NextAttemptAt:  now,
		Message:        "hello",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	summary, err := svc.DispatchDueNotificationOutbox(context.Background())
	if err != nil {
		t.Fatalf("DispatchDueNotificationOutbox() error = %v", err)
	}
	if summary.Claimed != 1 || summary.Failed != 1 {
		t.Fatalf("summary = %#v, want claimed=1 failed=1", summary)
	}

	var saved model.NotificationOutbox
	if err := db.First(&saved, job.ID).Error; err != nil {
		t.Fatalf("load outbox job failed: %v", err)
	}
	if saved.Status != notificationOutboxStatusFailed {
		t.Fatalf("outbox status = %q, want failed", saved.Status)
	}
	if saved.LastError == "" {
		t.Fatal("last_error should be populated")
	}

	var logEntry model.NotificationLog
	if err := db.Where("outbox_id = ?", job.ID).First(&logEntry).Error; err != nil {
		t.Fatalf("load failure log failed: %v", err)
	}
	if logEntry.Status != notificationLogStatusFailed {
		t.Fatalf("log status = %q, want failed", logEntry.Status)
	}
}

func TestDispatchNotificationOutboxReclaimsExpiredProcessingLease(t *testing.T) {
	db := newNotificationOutboxTestDB(t)
	user := createNotificationOutboxUser(t, db)
	notifyDate := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := createNotificationOutboxSubscription(t, db, user.ID, notifyDate)
	now := time.Date(2026, 3, 15, 8, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	t.Cleanup(restoreClock)
	expiredLease := now.Add(-time.Minute)

	job := model.NotificationOutbox{
		DedupeKey:      "expired-processing-lease",
		UserID:         user.ID,
		SubscriptionID: sub.ID,
		ChannelType:    "webhook",
		TriggerType:    notificationTriggerDueDay,
		NotifyDate:     notifyDate,
		ScheduledFor:   now,
		Status:         notificationOutboxStatusProcessing,
		MaxAttempts:    5,
		NextAttemptAt:  now,
		LockedBy:       "stale-owner",
		LockedUntil:    &expiredLease,
		Message:        "hello",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatalf("create outbox job failed: %v", err)
	}

	svc := NewNotificationService(db, nil, nil)
	svc.ownerID = "fresh-owner"
	jobs, err := svc.claimDueNotificationOutbox(context.Background(), 10, time.Minute)
	if err != nil {
		t.Fatalf("claimDueNotificationOutbox() error = %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("claim count = %d, want 1", len(jobs))
	}
	if jobs[0].LockedBy != "fresh-owner" {
		t.Fatalf("locked_by = %q, want fresh-owner", jobs[0].LockedBy)
	}
}
