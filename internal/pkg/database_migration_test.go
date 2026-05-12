package pkg

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func openRawSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-migration-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite test database: %v", err)
	}
	return db
}

func TestConfigureSQLiteDatabaseAppliesPragmas(t *testing.T) {
	db := openRawSQLiteTestDB(t)

	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}

	foreignKeys, err := readSQLiteIntPragma(db, "PRAGMA foreign_keys")
	if err != nil {
		t.Fatalf("read foreign_keys pragma error = %v", err)
	}
	if foreignKeys != 1 {
		t.Fatalf("foreign_keys = %d, want 1", foreignKeys)
	}

	journalMode, err := readSQLiteStringPragma(db, "PRAGMA journal_mode")
	if err != nil {
		t.Fatalf("read journal_mode pragma error = %v", err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	busyTimeout, err := readSQLiteIntPragma(db, "PRAGMA busy_timeout")
	if err != nil {
		t.Fatalf("read busy_timeout pragma error = %v", err)
	}
	if busyTimeout < sqliteBusyTimeoutMilliseconds {
		t.Fatalf("busy_timeout = %d, want at least %d", busyTimeout, sqliteBusyTimeoutMilliseconds)
	}
}

func TestRunSchemaMigrationsRebuildsLegacyTablesWithConstraints(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}

	legacySchema := []string{
		`CREATE TABLE users (id integer primary key autoincrement, username text not null, email text not null, password text not null, role text default 'user', status text default 'active', totp_secret text, totp_enabled numeric default false, totp_temp_secret text, created_at datetime, updated_at datetime)`,
		`CREATE TABLE categories (id integer primary key autoincrement, user_id integer not null, name text not null, system_key text, name_customized numeric default false, display_order integer default 0, created_at datetime, updated_at datetime)`,
		`CREATE TABLE subscriptions (id integer primary key autoincrement, user_id integer not null, name text not null, amount real not null, currency text default 'USD', enabled numeric default true, status text default 'active', renewal_mode text default 'auto_renew', ends_at datetime, billing_type text default 'recurring', recurrence_type text, interval_count integer, interval_unit text, monthly_day integer, yearly_month integer, yearly_day integer, next_billing_date datetime, category text, category_id integer, payment_method_id integer, notify_enabled numeric, notify_days_before integer, icon text, url text, notes text, created_at datetime, updated_at datetime)`,
		`CREATE TABLE notification_policies (id integer primary key autoincrement, user_id integer not null, days_before integer default 3, notify_on_due_day numeric default true, created_at datetime, updated_at datetime)`,
		`CREATE TABLE notification_logs (id integer primary key autoincrement, user_id integer not null, subscription_id integer not null, channel_type text not null, notify_date datetime not null, status text not null, error text, sent_at datetime)`,
	}
	for _, stmt := range legacySchema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed legacy schema error = %v", err)
		}
	}

	now := time.Date(2026, time.May, 12, 0, 0, 0, 0, time.UTC)
	primaryUser := model.User{Username: "legacy-user", Email: "legacy@example.com", Password: "hash", Role: "ADMIN", Status: "ACTIVE", CreatedAt: now, UpdatedAt: now}
	otherUser := model.User{Username: "other-user", Email: "other@example.com", Password: "hash", Role: "user", Status: "active", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&primaryUser).Error; err != nil {
		t.Fatalf("create primary user error = %v", err)
	}
	if err := db.Create(&otherUser).Error; err != nil {
		t.Fatalf("create other user error = %v", err)
	}

	foreignCategory := model.Category{UserID: otherUser.ID, Name: "Foreign", CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&foreignCategory).Error; err != nil {
		t.Fatalf("create foreign category error = %v", err)
	}

	nextBilling := now.Add(24 * time.Hour)
	subscription := model.Subscription{
		UserID:          primaryUser.ID,
		Name:            "Legacy Subscription",
		Amount:          9.99,
		Currency:        "USD",
		Enabled:         true,
		Status:          "ACTIVE",
		RenewalMode:     "AUTO_RENEW",
		BillingType:     "recurring",
		NextBillingDate: &nextBilling,
		CategoryID:      &foreignCategory.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(&subscription).Error; err != nil {
		t.Fatalf("create legacy subscription error = %v", err)
	}

	policy := model.NotificationPolicy{UserID: primaryUser.ID, DaysBefore: 3, NotifyOnDueDay: true, CreatedAt: now, UpdatedAt: now}
	if err := db.Create(&policy).Error; err != nil {
		t.Fatalf("create legacy notification policy error = %v", err)
	}

	logEntry := model.NotificationLog{UserID: primaryUser.ID, SubscriptionID: subscription.ID, ChannelType: "email", NotifyDate: now, Status: "sent", SentAt: now}
	if err := db.Create(&logEntry).Error; err != nil {
		t.Fatalf("create legacy notification log error = %v", err)
	}

	if err := runSchemaMigrations(db); err != nil {
		t.Fatalf("runSchemaMigrations() error = %v", err)
	}

	var migrationCount int64
	if err := db.Model(&schemaMigrationRecord{}).Count(&migrationCount).Error; err != nil {
		t.Fatalf("count schema migrations error = %v", err)
	}
	if migrationCount != int64(len(schemaMigrations)) {
		t.Fatalf("schema migration count = %d, want %d", migrationCount, len(schemaMigrations))
	}

	var migratedUser model.User
	if err := db.First(&migratedUser, primaryUser.ID).Error; err != nil {
		t.Fatalf("reload migrated user error = %v", err)
	}
	if migratedUser.Role != "admin" || migratedUser.Status != "active" {
		t.Fatalf("migrated user lifecycle = (%q, %q), want (admin, active)", migratedUser.Role, migratedUser.Status)
	}

	var migratedSub model.Subscription
	if err := db.First(&migratedSub, subscription.ID).Error; err != nil {
		t.Fatalf("reload migrated subscription error = %v", err)
	}
	if migratedSub.CategoryID != nil {
		t.Fatalf("migrated subscription category_id = %v, want nil after cross-user cleanup", *migratedSub.CategoryID)
	}
	if migratedSub.Status != subscriptionStatusActive || migratedSub.RenewalMode != subscriptionRenewalModeAutoRenew {
		t.Fatalf("migrated subscription lifecycle = (%q, %q), want (%q, %q)", migratedSub.Status, migratedSub.RenewalMode, subscriptionStatusActive, subscriptionRenewalModeAutoRenew)
	}

	invalidPolicy := model.NotificationPolicy{UserID: otherUser.ID, DaysBefore: 99, NotifyOnDueDay: true}
	if err := db.Create(&invalidPolicy).Error; err == nil {
		t.Fatal("expected notification policy check constraint error, got nil")
	}

	if err := db.Delete(&model.User{}, primaryUser.ID).Error; err != nil {
		t.Fatalf("direct user delete error = %v", err)
	}

	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{name: "subscriptions", model: &model.Subscription{}},
		{name: "notification_policies", model: &model.NotificationPolicy{}},
		{name: "notification_logs", model: &model.NotificationLog{}},
	} {
		var count int64
		if err := db.Model(tc.model).Where("user_id = ?", primaryUser.ID).Count(&count).Error; err != nil {
			t.Fatalf("count %s error = %v", tc.name, err)
		}
		if count != 0 {
			t.Fatalf("%s count = %d, want 0 after FK cascade", tc.name, count)
		}
	}

	if err := db.First(&model.User{}, primaryUser.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("deleted user lookup error = %v, want %v", err, gorm.ErrRecordNotFound)
	}
}
