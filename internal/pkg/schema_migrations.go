package pkg

import (
	"fmt"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const schemaMigrationTableName = "schema_migrations"

type schemaMigrationRecord struct {
	Name      string    `gorm:"primaryKey;size:191"`
	AppliedAt time.Time `gorm:"not null"`
}

func (schemaMigrationRecord) TableName() string {
	return schemaMigrationTableName
}

type schemaMigration struct {
	Name string
	Run  func(db *gorm.DB) error
}

var applicationModels = []interface{}{
	&model.User{},
	&model.EmailVerificationCode{},
	&model.Subscription{},
	&model.SystemSetting{},
	&model.ExchangeRate{},
	&model.UserPreference{},
	&model.UserCurrency{},
	&model.UserBackupCode{},
	&model.PasskeyCredential{},
	&model.OIDCConnection{},
	&model.Category{},
	&model.PaymentMethod{},
	&model.NotificationChannel{},
	&model.NotificationPolicy{},
	&model.NotificationLog{},
	&model.NotificationTemplate{},
	&model.APIKey{},
	&model.RefreshToken{},
	&model.CalendarToken{},
	&model.AuditEvent{},
}

var postIntegrityApplicationModels = []interface{}{
	&model.SubscriptionEvent{},
	&model.SubscriptionActionSnooze{},
}

var schemaMigrations = []schemaMigration{
	{Name: "20260512_01_create_missing_tables", Run: createMissingTables},
	{Name: "20260512_02_subscription_lifecycle_backfill", Run: backfillSubscriptionLifecycleFields},
	{Name: "20260512_03_sqlite_integrity_hardening", Run: migrateSQLiteIntegrityHardening},
	{Name: "20260512_04_auto_migrate_latest_schema", Run: autoMigrateLatestSchema},
	{Name: "20260525_01_subscription_events", Run: migrateSubscriptionEventsSchema},
	{Name: "20260527_01_subscription_action_snoozes", Run: migrateSubscriptionEventsSchema},
	{Name: "20260622_01_notification_outbox_leases", Run: migrateNotificationOutboxLeases},
	{Name: "20260623_01_api_key_kind_and_audit", Run: migrateAPIKeyKindAndAudit},
	{Name: "20260628_01_manual_renew_daily_notifications", Run: migrateManualRenewDailyNotificationPolicy},
}

func autoMigrateLatestSchema(db *gorm.DB) error {
	return db.AutoMigrate(applicationModels...)
}

func migrateSubscriptionEventsSchema(db *gorm.DB) error {
	return db.AutoMigrate(postIntegrityApplicationModels...)
}

func migrateNotificationOutboxLeases(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.BackgroundTaskLease{},
		&model.NotificationOutbox{},
		&model.NotificationLog{},
	)
}

func migrateAPIKeyKindAndAudit(db *gorm.DB) error {
	if err := db.AutoMigrate(&model.APIKey{}, &model.AuditEvent{}); err != nil {
		return err
	}
	return db.Model(&model.APIKey{}).
		Where("key_kind IS NULL OR TRIM(key_kind) = ''").
		Update("key_kind", "api_integration").Error
}

func migrateManualRenewDailyNotificationPolicy(db *gorm.DB) error {
	return db.AutoMigrate(&model.NotificationPolicy{})
}

func runSchemaMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigrationRecord{}); err != nil {
		return fmt.Errorf("auto-migrate schema_migrations: %w", err)
	}

	for _, migration := range schemaMigrations {
		applied, err := isSchemaMigrationApplied(db, migration.Name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := migration.Run(db); err != nil {
			return fmt.Errorf("apply migration %s: %w", migration.Name, err)
		}

		record := schemaMigrationRecord{Name: migration.Name, AppliedAt: NowUTC()}
		if err := db.Create(&record).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", migration.Name, err)
		}
	}

	return nil
}

func isSchemaMigrationApplied(db *gorm.DB, name string) (bool, error) {
	var count int64
	if err := db.Model(&schemaMigrationRecord{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check migration %s: %w", name, err)
	}
	return count > 0, nil
}
