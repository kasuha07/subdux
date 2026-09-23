package notification

import (
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/migrations"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func TestPostgresOutboxClaimSkipsRowsLockedByAnotherWorker(t *testing.T) {
	db := openNotificationPostgresTestDB(t)
	now := pkg.NowUTC().Add(-time.Minute)
	user := model.User{
		Username: "queue-" + uuid.NewString(),
		Email:    uuid.NewString() + "@example.com",
		Password: "password-hash",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create queue user: %v", err)
	}
	subscription := model.Subscription{
		UserID:      user.ID,
		Name:        "queue test",
		Amount:      1,
		Currency:    "USD",
		Enabled:     true,
		Status:      "active",
		RenewalMode: "auto_renew",
		BillingType: "recurring",
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create queue subscription: %v", err)
	}

	jobs := []model.NotificationOutbox{
		{DedupeKey: uuid.NewString(), UserID: user.ID, SubscriptionID: subscription.ID, ChannelType: "email", TriggerType: "due_day", NotifyDate: now, ScheduledFor: now, Status: notificationOutboxStatusPending, NextAttemptAt: now, Message: "first job"},
		{DedupeKey: uuid.NewString(), UserID: user.ID, SubscriptionID: subscription.ID, ChannelType: "email", TriggerType: "due_day", NotifyDate: now, ScheduledFor: now, Status: notificationOutboxStatusPending, NextAttemptAt: now, Message: "second job"},
	}
	if err := db.Create(&jobs).Error; err != nil {
		t.Fatalf("create notification outbox jobs: %v", err)
	}

	locker := db.Begin()
	if locker.Error != nil {
		t.Fatalf("begin row-locking transaction: %v", locker.Error)
	}
	defer locker.Rollback()
	var locked model.NotificationOutbox
	if err := locker.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, jobs[0].ID).Error; err != nil {
		t.Fatalf("lock first queue job: %v", err)
	}

	secondWorker := NewService(db, nil, nil)
	secondWorker.ownerID = "worker-two"
	claimed, err := secondWorker.claimDueNotificationOutbox(nil, 10, time.Minute)
	if err != nil {
		t.Fatalf("second worker claim: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != jobs[1].ID {
		t.Fatalf("second worker claimed IDs = %v, want only %d", notificationOutboxIDs(claimed), jobs[1].ID)
	}
	if claimed[0].LockedBy != secondWorker.ownerID || claimed[0].AttemptCount != 1 {
		t.Fatalf("second worker claim state = owner %q, attempts %d; want owner %q, attempts 1", claimed[0].LockedBy, claimed[0].AttemptCount, secondWorker.ownerID)
	}

	if err := locker.Rollback().Error; err != nil {
		t.Fatalf("release first queue job lock: %v", err)
	}
	firstWorker := NewService(db, nil, nil)
	firstWorker.ownerID = "worker-one"
	claimed, err = firstWorker.claimDueNotificationOutbox(nil, 10, time.Minute)
	if err != nil {
		t.Fatalf("first worker claim after unlock: %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != jobs[0].ID {
		t.Fatalf("first worker claimed IDs = %v, want only %d", notificationOutboxIDs(claimed), jobs[0].ID)
	}
}

func openNotificationPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("SUBDUX_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set SUBDUX_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("SUBDUX_TEST_POSTGRES_DSN must be a postgres:// or postgresql:// URL")
	}

	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("access PostgreSQL admin connection: %v", err)
	}
	schema := "subdux_queue_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		_ = adminSQL.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop isolated PostgreSQL schema: %v", err)
		}
		_ = adminSQL.Close()
	})

	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access isolated PostgreSQL database: %v", err)
	}
	if err := migrations.Run(db, migrations.SecretCodec{
		Encrypt: pkg.EncryptSystemSettingValue,
		Decrypt: pkg.DecryptSystemSettingValue,
	}); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrate isolated PostgreSQL database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close isolated PostgreSQL database: %v", err)
		}
	})
	return db
}

func notificationOutboxIDs(jobs []model.NotificationOutbox) []uint {
	ids := make([]uint, 0, len(jobs))
	for _, job := range jobs {
		ids = append(ids, job.ID)
	}
	return ids
}
