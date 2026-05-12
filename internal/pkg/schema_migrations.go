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
}

var schemaMigrations = []schemaMigration{
	{Name: "20260512_01_create_missing_tables", Run: createMissingTables},
	{Name: "20260512_02_subscription_lifecycle_backfill", Run: backfillSubscriptionLifecycleFields},
	{Name: "20260512_03_sqlite_integrity_hardening", Run: migrateSQLiteIntegrityHardening},
	{Name: "20260512_04_auto_migrate_latest_schema", Run: autoMigrateLatestSchema},
}

func autoMigrateLatestSchema(db *gorm.DB) error {
	return db.AutoMigrate(applicationModels...)
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
