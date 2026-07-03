package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

const testSQLiteBusyTimeoutMilliseconds = 5000

func openRawSQLiteTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-migration-test.db")
	return openRawSQLiteTestDBAt(t, dbPath)
}

func openRawSQLiteTestDBAt(t *testing.T, dbPath string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite test database: %v", err)
	}
	return db
}

func configureSQLiteConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("access sqlite connection pool: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	return nil
}

func configureSQLiteDatabase(db *gorm.DB) error {
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	if err := db.Exec("PRAGMA journal_mode = WAL").Error; err != nil {
		return fmt.Errorf("enable sqlite wal mode: %w", err)
	}
	if err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout = %d", testSQLiteBusyTimeoutMilliseconds)).Error; err != nil {
		return fmt.Errorf("set sqlite busy_timeout: %w", err)
	}
	return nil
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

	policy := map[string]interface{}{
		"user_id":           primaryUser.ID,
		"days_before":       3,
		"notify_on_due_day": true,
		"created_at":        now,
		"updated_at":        now,
	}
	if err := db.Table("notification_policies").Create(policy).Error; err != nil {
		t.Fatalf("create legacy notification policy error = %v", err)
	}

	logEntry := map[string]interface{}{
		"user_id":         primaryUser.ID,
		"subscription_id": subscription.ID,
		"channel_type":    "email",
		"notify_date":     now,
		"status":          "sent",
		"sent_at":         now,
	}
	if err := db.Table("notification_logs").Create(logEntry).Error; err != nil {
		t.Fatalf("create legacy notification log error = %v", err)
	}

	if err := Run(db); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var migrationCount int64
	if err := db.Model(&schemaMigrationRecord{}).Count(&migrationCount).Error; err != nil {
		t.Fatalf("count schema migrations error = %v", err)
	}
	if migrationCount != int64(len(schemaMigrations)) {
		t.Fatalf("schema migration count = %d, want %d", migrationCount, len(schemaMigrations))
	}
	assertSchemaMigrationRecordsClean(t, db)

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

	actorUserID := primaryUser.ID
	event := model.SubscriptionEvent{
		UserID:           primaryUser.ID,
		ActorUserID:      &actorUserID,
		SubscriptionID:   &migratedSub.ID,
		SubscriptionName: migratedSub.Name,
		Type:             "created",
		ChangedFields:    `["created"]`,
		NewAmount:        &migratedSub.Amount,
		NewMonthlyAmount: &migratedSub.Amount,
		NewCurrency:      migratedSub.Currency,
		NewStatus:        migratedSub.Status,
		NewRenewalMode:   migratedSub.RenewalMode,
		CreatedAt:        now,
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create subscription event after migrations error = %v", err)
	}
	if err := validateSQLiteForeignKeys(db); err != nil {
		t.Fatalf("validate foreign keys after subscription event error = %v", err)
	}

	lease := model.BackgroundTaskLease{
		TaskKey:     "notification_scan",
		OwnerID:     "migration-test",
		LeaseUntil:  now.Add(time.Minute),
		HeartbeatAt: now,
	}
	if err := db.Create(&lease).Error; err != nil {
		t.Fatalf("create background task lease after migrations error = %v", err)
	}

	outbox := model.NotificationOutbox{
		DedupeKey:      "migration-test-dedupe",
		UserID:         primaryUser.ID,
		SubscriptionID: migratedSub.ID,
		ChannelType:    "webhook",
		TriggerType:    "due_day",
		NotifyDate:     now,
		ScheduledFor:   now,
		Status:         "pending",
		MaxAttempts:    5,
		NextAttemptAt:  now,
		Message:        "migration test",
	}
	if err := db.Create(&outbox).Error; err != nil {
		t.Fatalf("create notification outbox after migrations error = %v", err)
	}

	invalidPolicy := model.NotificationPolicy{UserID: otherUser.ID, DaysBefore: 99, NotifyOnDueDay: true}
	if err := db.Create(&invalidPolicy).Error; err == nil {
		t.Fatal("expected notification policy check constraint error, got nil")
	}

	var migratedPolicy model.NotificationPolicy
	if err := db.Where("user_id = ?", primaryUser.ID).First(&migratedPolicy).Error; err != nil {
		t.Fatalf("reload migrated notification policy error = %v", err)
	}
	if migratedPolicy.NotifyManualRenewDaily {
		t.Fatal("notify_manual_renew_daily = true, want false default for migrated policy")
	}

	if err := db.Delete(&model.User{}, primaryUser.ID).Error; err != nil {
		t.Fatalf("direct user delete error = %v", err)
	}

	for _, tc := range []struct {
		name  string
		model interface{}
	}{
		{name: "subscriptions", model: &model.Subscription{}},
		{name: "subscription_events", model: &model.SubscriptionEvent{}},
		{name: "notification_policies", model: &model.NotificationPolicy{}},
		{name: "notification_logs", model: &model.NotificationLog{}},
		{name: "notification_outboxes", model: &model.NotificationOutbox{}},
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

func TestRunSchemaMigrationsConvertsExchangeRatesToUSDBase(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}

	if err := db.Exec(`CREATE TABLE exchange_rates (
		id integer primary key autoincrement,
		base_currency text not null,
		target_currency text not null,
		rate real not null,
		source text not null,
		fetched_at datetime not null,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create legacy exchange_rates error = %v", err)
	}

	now := time.Date(2026, time.July, 3, 0, 0, 0, 0, time.UTC)
	legacyRates := []map[string]interface{}{
		{"base_currency": "usd", "target_currency": "eur", "rate": 0.8, "source": "legacy", "fetched_at": now, "created_at": now, "updated_at": now},
		{"base_currency": "eur", "target_currency": "cny", "rate": 9.0, "source": "legacy", "fetched_at": now.Add(2 * time.Minute), "created_at": now, "updated_at": now},
		{"base_currency": "gbp", "target_currency": "jpy", "rate": 180.0, "source": "legacy", "fetched_at": now.Add(3 * time.Minute), "created_at": now, "updated_at": now},
		{"base_currency": "usd", "target_currency": "usd", "rate": 1.0, "source": "legacy", "fetched_at": now.Add(4 * time.Minute), "created_at": now, "updated_at": now},
		{"base_currency": "usd", "target_currency": "cad", "rate": -1.2, "source": "legacy", "fetched_at": now.Add(5 * time.Minute), "created_at": now, "updated_at": now},
	}
	for _, rate := range legacyRates {
		if err := db.Table("exchange_rates").Create(rate).Error; err != nil {
			t.Fatalf("seed legacy exchange rate error = %v", err)
		}
	}

	if err := Run(db); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	var migrated []model.ExchangeRate
	if err := db.Order("target_currency ASC").Find(&migrated).Error; err != nil {
		t.Fatalf("load migrated exchange rates error = %v", err)
	}

	if len(migrated) != 2 {
		t.Fatalf("migrated exchange rate rows = %d, want 2", len(migrated))
	}
	rates := make(map[string]float64)
	for _, rate := range migrated {
		rates[rate.TargetCurrency] = rate.Rate
	}
	if rates["eur"] != 0.8 {
		t.Fatalf("USD->EUR migrated rate = %v, want 0.8", rates["eur"])
	}
	if rates["cny"] != 7.2 {
		t.Fatalf("USD->CNY migrated rate = %v, want 7.2", rates["cny"])
	}
	for _, discarded := range []string{"jpy", "usd", "cad"} {
		if _, ok := rates[discarded]; ok {
			t.Fatalf("discarded exchange rate %q was preserved: %#v", discarded, migrated)
		}
	}

	var baseCurrencyColumnCount int
	if err := db.Raw(
		"SELECT COUNT(1) FROM pragma_table_info(?) WHERE name = ?",
		"exchange_rates",
		"base_currency",
	).Scan(&baseCurrencyColumnCount).Error; err != nil {
		t.Fatalf("inspect migrated exchange_rates columns error = %v", err)
	}
	if baseCurrencyColumnCount != 0 {
		t.Fatalf("base_currency column count = %d, want 0", baseCurrencyColumnCount)
	}
}

func TestSchemaMigrationTransactionRollsBackExecutionAndRecord(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}
	if err := ensureSchemaMigrationMetadata(db); err != nil {
		t.Fatalf("ensureSchemaMigrationMetadata() error = %v", err)
	}

	migration := schemaMigration{
		Name:     "test_rollback",
		Checksum: strings.Repeat("a", 64),
		Run: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE rollback_probe (id integer primary key)`).Error; err != nil {
				return err
			}
			return errors.New("boom")
		},
	}
	if err := runSchemaMigrationTransaction(db, migration); err == nil {
		t.Fatal("runSchemaMigrationTransaction() error = nil, want failure")
	}
	if db.Migrator().HasTable("rollback_probe") {
		t.Fatal("rollback_probe table exists after failed migration")
	}

	var count int64
	if err := db.Model(&schemaMigrationRecord{}).Where("name = ?", migration.Name).Count(&count).Error; err != nil {
		t.Fatalf("count rollback migration record error = %v", err)
	}
	if count != 0 {
		t.Fatalf("rollback migration record count = %d, want 0", count)
	}
}

func TestSchemaMigrationChecksumAndDirtyDetection(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}
	if err := ensureSchemaMigrationMetadata(db); err != nil {
		t.Fatalf("ensureSchemaMigrationMetadata() error = %v", err)
	}

	migration := schemaMigrations[0]
	dirty := schemaMigrationRecord{Name: migration.Name, Checksum: migration.Checksum, Dirty: true, AppliedAt: nowUTC()}
	if err := db.Create(&dirty).Error; err != nil {
		t.Fatalf("create dirty migration record error = %v", err)
	}
	if err := validateSchemaMigrationRecords(db); err == nil || !strings.Contains(err.Error(), "dirty") {
		t.Fatalf("validate dirty record error = %v, want dirty error", err)
	}

	if err := db.Model(&schemaMigrationRecord{}).
		Where("name = ?", migration.Name).
		Updates(map[string]interface{}{"dirty": false, "checksum": strings.Repeat("b", 64)}).Error; err != nil {
		t.Fatalf("update bad checksum error = %v", err)
	}
	if err := validateSchemaMigrationRecords(db); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("validate checksum mismatch error = %v, want checksum mismatch", err)
	}
}

func TestSchemaMigrationBackfillsLegacyChecksum(t *testing.T) {
	db := openRawSQLiteTestDB(t)
	if err := configureSQLiteDatabase(db); err != nil {
		t.Fatalf("configureSQLiteDatabase() error = %v", err)
	}
	if err := ensureSchemaMigrationMetadata(db); err != nil {
		t.Fatalf("ensureSchemaMigrationMetadata() error = %v", err)
	}

	migration := schemaMigration{
		Name:     "test_legacy_checksum",
		Checksum: strings.Repeat("c", 64),
		Run: func(tx *gorm.DB) error {
			t.Fatal("already-applied migration should not run")
			return nil
		},
	}
	if err := db.Create(&schemaMigrationRecord{Name: migration.Name, AppliedAt: nowUTC()}).Error; err != nil {
		t.Fatalf("create legacy migration record error = %v", err)
	}
	if err := runSchemaMigrationTransaction(db, migration); err != nil {
		t.Fatalf("runSchemaMigrationTransaction() error = %v", err)
	}

	var record schemaMigrationRecord
	if err := db.First(&record, "name = ?", migration.Name).Error; err != nil {
		t.Fatalf("load legacy migration record error = %v", err)
	}
	if record.Checksum != migration.Checksum || record.Dirty {
		t.Fatalf("record after checksum backfill = (%q, dirty=%v), want (%q, dirty=false)", record.Checksum, record.Dirty, migration.Checksum)
	}
}

func TestSchemaMigrationLockSerializesConcurrentStartup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "subdux-lock-test.db")
	db1 := openRawSQLiteTestDBAt(t, dbPath)
	db2 := openRawSQLiteTestDBAt(t, dbPath)
	for name, db := range map[string]*gorm.DB{"db1": db1, "db2": db2} {
		if err := configureSQLiteConnectionPool(db); err != nil {
			t.Fatalf("%s configureSQLiteConnectionPool() error = %v", name, err)
		}
		if err := configureSQLiteDatabase(db); err != nil {
			t.Fatalf("%s configureSQLiteDatabase() error = %v", name, err)
		}
	}
	if err := ensureSchemaMigrationMetadata(db1); err != nil {
		t.Fatalf("ensureSchemaMigrationMetadata() error = %v", err)
	}

	tx := db1.Begin()
	if tx.Error != nil {
		t.Fatalf("begin holder transaction error = %v", tx.Error)
	}
	defer tx.Rollback()
	if err := acquireSchemaMigrationLock(tx); err != nil {
		t.Fatalf("acquire holder migration lock error = %v", err)
	}

	started := make(chan struct{})
	done := make(chan error, 1)
	var runCount int32
	go func() {
		done <- runSchemaMigrationTransaction(db2, schemaMigration{
			Name:     "test_lock_wait",
			Checksum: strings.Repeat("d", 64),
			Run: func(tx *gorm.DB) error {
				atomic.AddInt32(&runCount, 1)
				close(started)
				return nil
			},
		})
	}()

	select {
	case <-started:
		t.Fatal("migration ran before held schema migration lock was released")
	case err := <-done:
		t.Fatalf("migration completed while lock was held: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("commit holder transaction error = %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("locked migration error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("migration did not complete after lock was released")
	}
	if got := atomic.LoadInt32(&runCount); got != 1 {
		t.Fatalf("migration run count = %d, want 1", got)
	}
}

func TestPublishedSchemaMigrationManifestIsImmutable(t *testing.T) {
	type manifestEntry struct {
		Name                     string
		Checksum                 string
		DisableSQLiteForeignKeys bool
		Destructive              bool
		DiscardPolicy            string
	}
	got := make([]manifestEntry, 0, len(schemaMigrations))
	for _, migration := range schemaMigrations {
		if len(migration.Checksum) != 64 {
			t.Fatalf("migration %s checksum length = %d, want 64", migration.Name, len(migration.Checksum))
		}
		if _, err := hex.DecodeString(migration.Checksum); err != nil {
			t.Fatalf("migration %s checksum is not hex: %v", migration.Name, err)
		}
		if migration.Destructive && strings.TrimSpace(migration.DiscardPolicy) == "" {
			t.Fatalf("destructive migration %s has no discard policy", migration.Name)
		}
		got = append(got, manifestEntry{
			Name:                     migration.Name,
			Checksum:                 migration.Checksum,
			DisableSQLiteForeignKeys: migration.DisableSQLiteForeignKeys,
			Destructive:              migration.Destructive,
			DiscardPolicy:            migration.DiscardPolicy,
		})
	}

	want := []manifestEntry{
		{Name: "20260512_01_create_missing_tables", Checksum: "245c0c8016fb52e36d24e5fc475208d011aa694f0c8ea5b74310e4fad99a2e57"},
		{Name: "20260512_02_subscription_lifecycle_backfill", Checksum: "5808302cc9f5c723cf7fdc0a57abc88e6655195ab37172c9d11ee9e2d1ee9ef8"},
		{Name: "20260512_03_sqlite_integrity_hardening", Checksum: "3fc886fe568666161a3198031cb174a76f63aa5fc0537de256b24d3faa4a68cc", DisableSQLiteForeignKeys: true},
		{Name: "20260512_04_auto_migrate_latest_schema", Checksum: "4eb2d348b78dc3c8216fb3d3e0d61754c48a231ea8effd23174f879399cba720"},
		{Name: "20260525_01_subscription_events", Checksum: "b562049e926f70372b47f45cd680dc13fa21c78b9113741a1ab472ee537aeb74"},
		{Name: "20260527_01_subscription_action_snoozes", Checksum: "21b028f8c331b01ccad0b0c9f8c90c913bbf6228fa9138afeb937b04aac8056c"},
		{Name: "20260622_01_notification_outbox_leases", Checksum: "774a578619b78de5464eb0f0a2ba3a90b0a792b03f615f930f15cd8a57f325d0"},
		{Name: "20260623_01_api_key_kind_and_audit", Checksum: "bf185c37f9c1889ac9b2eb12e593d4481b5df5b53367be061900306501bf530e"},
		{Name: "20260628_01_manual_renew_daily_notifications", Checksum: "56b5d9babd3f0ad8b0971aa5f1a7761970027102ad012bc4adb00caa4c056500"},
		{Name: "20260628_02_mcp_idempotency_keys", Checksum: "d0198cc337bda90a1531f6e419f7a538cb506d24af6b1e4686f96ebe29c6762d"},
		{Name: "20260628_03_performance_composite_indexes", Checksum: "271bab2eac3f658250896444c658f5bb5a730cd57f2eed77cfb7fd3e9bbfbc42"},
		{
			Name:          "20260703_01_usd_base_exchange_rates",
			Checksum:      "8444b4c52d92a7ff1ed722104a330c7be64a2f9e4828bead6ae133d479dcaa52",
			Destructive:   true,
			DiscardPolicy: "Rebuild exchange_rates as USD-base only. Preserve direct USD pairs and rates derivable through an existing USD pair; discard invalid rows, self-pairs, and cross-rates that cannot be expressed from available USD data.",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("schema migration manifest changed\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPublishedSchemaMigrationSourcesAreImmutable(t *testing.T) {
	// Published migration sources are append-only. If a released migration needs
	// different behavior, add a new migration instead of editing these files.
	want := map[string]string{
		"migration_20260512_01_create_missing_tables.go":           "aca700b695e6769f6b5f2277d9278367ff8e3960f1a1ffda1492c0fe8bb0697b",
		"migration_20260512_02_subscription_lifecycle_backfill.go": "ae5999e185f6a2457fa6ee42b7409ae03f6b770c501820a1e6a6a08ac6b7a64b",
		"migration_20260512_03_sqlite_integrity_hardening.go":      "e5917afe801db4f076f4c48a0a0e4111159ece4bc436fa4944cc4e00c5b267b5",
		"schema_migration_registry.go":                             "c1889dd866e3037e898d40fbc5798f235e1daba635f1434d07da9adb539b868a",
		"schema_migration_steps.go":                                "d24175ab071d227a95a5a0302200cf501fde129c12d45f4fcea7659ab6b00dfa",
	}
	for path, expected := range want {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s error = %v", path, err)
		}
		sum := sha256.Sum256(contents)
		if got := hex.EncodeToString(sum[:]); got != expected {
			t.Fatalf("%s checksum = %s, want %s; add a new migration instead of editing published migration code", path, got, expected)
		}
	}
}

func assertSchemaMigrationRecordsClean(t *testing.T, db *gorm.DB) {
	t.Helper()

	checksums := make(map[string]string, len(schemaMigrations))
	for _, migration := range schemaMigrations {
		checksums[migration.Name] = migration.Checksum
	}

	var records []schemaMigrationRecord
	if err := db.Find(&records).Error; err != nil {
		t.Fatalf("load schema migration records error = %v", err)
	}
	for _, record := range records {
		if record.Dirty {
			t.Fatalf("migration record %s is dirty", record.Name)
		}
		if want := checksums[record.Name]; record.Checksum != want {
			t.Fatalf("migration record %s checksum = %q, want %q", record.Name, record.Checksum, want)
		}
	}
}
